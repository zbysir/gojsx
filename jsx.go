package jsx

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/zbysir/gojsx/internal/js"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Transformer 负责将高级语法（包括 jsx，ts）转为 goja 能运行的 ES5.1
type Transformer struct {
	p     sync.Pool
	c     chan struct{}
	cache SourceCache
}

type SourceCache interface {
	Get(key string) (f *Source, exist bool, err error)
	Set(key string, f *Source) (err error)
}

type Source struct {
	SrcMd5    string
	Body      []byte
	CreatedAt string
}

type FileCache struct {
	cachePath string
}

func NewFileCache(cachePath string) *FileCache {
	return &FileCache{cachePath: cachePath}
}

// Get
// TODO lock on one file
func (fc *FileCache) Get(key string) (f *Source, exist bool, err error) {
	cacheFilePath := filepath.Join(fc.cachePath, key)

	cbs, err := os.ReadFile(cacheFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return
	}
	if err == nil {
		x := bytes.IndexByte(cbs, '\n')
		if x == -1 {
			return
		}
		f = &Source{
			SrcMd5:    string(cbs[:x]),
			Body:      cbs[x+1:],
			CreatedAt: "",
		}
		exist = true
		return
	}

	return
}

func (fc *FileCache) Set(key string, f *Source) (err error) {
	_ = os.MkdirAll(fc.cachePath, os.ModePerm)

	cacheFilePath := filepath.Join(fc.cachePath, key)
	fi, err := os.OpenFile(cacheFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	defer fi.Close()
	if err != nil {
		return fmt.Errorf("os.OpenFile error: %w", err)
	}

	_, err = fi.WriteString(f.SrcMd5)
	if err != nil {
		return err
	}
	_, err = fi.Write([]byte("\n"))
	if err != nil {
		return err
	}
	_, err = fi.Write(f.Body)
	if err != nil {
		return
	}

	return
}

func NewTransformer(fileCache SourceCache) (*Transformer, error) {
	return &Transformer{
		p: sync.Pool{New: func() any {
			vm := goja.New()

			require.NewRegistry().Enable(vm)
			console.Enable(vm)

			_, err := vm.RunScript("babel", js.Babel)
			if err != nil {
				panic(err)
			}
			return vm
		}},
		c:     make(chan struct{}, 20),
		cache: fileCache,
	}, nil
}

func mD5(v []byte) string {
	m := md5.New()
	m.Write(v)
	return hex.EncodeToString(m.Sum(nil))
}

func (t *Transformer) transform(filePath string, code []byte) ([]byte, bool, error) {
	// 并行
	t.c <- struct{}{}
	defer func() { <-t.c }()

	cacheKey := mD5([]byte(filePath))
	srcMd5 := mD5(code)
	if t.cache != nil {
		fi, exist, err := t.cache.Get(cacheKey)
		if err != nil {
			return nil, false, err
		}
		if exist && fi.SrcMd5 == srcMd5 {
			return fi.Body, true, nil
		}
	}

	vm := t.p.Get().(*goja.Runtime)
	defer t.p.Put(vm)

	_, name := filepath.Split(filePath)
	vm.Set("filepath", filePath)
	vm.Set("filename", name)

	v, err := vm.RunString(fmt.Sprintf(`Babel.transform('%s', { presets: ["react","es2015"], sourceMaps: 'inline', sourceFileName: filename, filename: filepath, plugins: [
    [
      "transform-react-jsx",
      {
        "runtime": "automatic", // defaults to classic
      }
    ],
	[
		"transform-typescript",
		{
			"isTSX": true,
		}
	]
  ] }).code`, template.JSEscapeString(string(code))))
	if err != nil {
		return nil, false, err
	}
	bs := []byte(v.String())
	if t.cache != nil {
		err = t.cache.Set(cacheKey, &Source{
			SrcMd5:    srcMd5,
			Body:      bs,
			CreatedAt: "",
		})
		if err != nil {
			return nil, false, nil
		}
	}
	return bs, false, nil
}

func (j *Jsx) RunJs(fileName string, src []byte, transform bool) (v goja.Value, err error) {
	if transform {
		src, _, err = j.tr.transform(fileName, src)
		if err != nil {
			return nil, err
		}
	}

	v, err = j.vm.RunString(string(src))
	return v, err
}

// Compile 预编译多个文件，生成 cache 文件
func (j *Jsx) Compile(path string) (err error) {

	//j.tr.transform(path)
	return nil
}

type MountEndpoint struct {
	Endpoint  string
	Component string
	Props     interface{}
}

// Render a component to html
func (j *Jsx) Render(component string, props interface{}) (n string, err error) {
	j.lock.Lock()
	defer j.lock.Unlock()

	err = j.vm.Set("props", j.vm.ToValue(props))
	if err != nil {
		return "", err
	}
	res, err := j.RunJs(component, []byte(fmt.Sprintf(`require("%v").default(props)`, component)), false)
	if err != nil {
		return "", err
	}

	vdom := VDom{}
	err = j.vm.ExportTo(res, &vdom)
	if err != nil {
		return "", err
	}
	//fmt.Printf("vdom: \n%+v", vdom)

	return vdom.Render(), nil
}

type Jsx struct {
	vm *goja.Runtime
	tr *Transformer

	// goja is not goroutine-safe
	lock     sync.Mutex
	sourceFs fs.FS

	debug bool
}

type StdFileSystem struct {
}

func (f StdFileSystem) Open(name string) (fs.File, error) {
	return os.Open(name)
}

type option struct {
	sourceCache SourceCache
	sourceFs    fs.FS
	debug       bool // enable to get more log
}

type Option func(jsx *option)

func WithFS(f fs.FS) Option {
	return func(jsx *option) {
		jsx.sourceFs = f
	}
}

func WithDebug(d bool) Option {
	return func(jsx *option) {
		jsx.debug = d
	}
}

func WithSourceCache(ss SourceCache) Option {
	return func(jsx *option) {
		jsx.sourceCache = ss
	}
}

func NewJsx(ops ...Option) (*Jsx, error) {
	var op option
	for _, o := range ops {
		o(&op)
	}

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	transformer, err := NewTransformer(op.sourceCache)
	if err != nil {
		return nil, err
	}

	if op.sourceFs == nil {
		op.sourceFs = StdFileSystem{}
	}

	j := &Jsx{
		vm:       vm,
		tr:       transformer,
		lock:     sync.Mutex{},
		sourceFs: op.sourceFs,
		debug:    false,
	}

	vm.Set("process", map[string]interface{}{
		"env": map[string]interface{}{
			"NODE_ENV": "production",
		},
	})

	j.reload()
	return j, nil
}

func (j *Jsx) reload() {
	// 使用 transformer 预编译 src 的文件，能加快速度

	// 当文件更改时，可以新 new registry 来拿到最新的文件
	registry := require.NewRegistryWithLoader(func(path string) ([]byte, error) {
		var fileBody []byte
		//var filePath string
		needTrans := false
		if j.debug {
			fmt.Printf("tryload: %v\n", path)
		}

		s := time.Now()
		if strings.Contains(path, "node_modules/react/jsx-runtime") {
			fileBody = js.Jsx
			//filePath = path
			needTrans = true
		} else {
			find := false
			paths := []string{""}
			if strings.HasSuffix(path, ".js") {
				needTrans = true
				paths = append(paths, ".jsx")
			}
			for _, p := range paths {
				if p != "" {
					path = strings.TrimSuffix(path, ".js") + p
				}
				//filePath = path + ".jsx"
				bs, err := fs.ReadFile(j.sourceFs, path)
				if err != nil {
					if errors.Is(err, fs.ErrNotExist) || strings.Contains(err.Error(), "is a directory") {
						continue
					}
					return nil, fmt.Errorf("can't load module: %v, error: %w", path, err)
				}
				find = true
				fileBody = bs
				break
			}
			if !find {
				return nil, require.ModuleFileDoesNotExistError
			}
		}
		fmt.Printf("load: %v", path)
		var err error
		var cached bool
		if needTrans {
			fmt.Printf(" transform")
			fileBody, cached, err = j.tr.transform(path, fileBody)
			if err != nil {
				return nil, err
			}
			if cached {
				fmt.Printf(" cached")
			}
		}
		fmt.Printf(" %v\n", time.Now().Sub(s))

		return fileBody, nil
	})
	//registry := require.NewRegistry()
	registry.Enable(j.vm)
	console.Enable(j.vm)
}

//type VDom struct {
//	NodeName   string                 `json:"nodeName"`
//	Attributes map[string]interface{} `json:"attributes"`
//}

type VDom map[string]interface{}

func (v VDom) RenderAttributes(s *strings.Builder, ps map[string]interface{}) {
	if len(ps) == 0 {
		return
	}

	for k, val := range ps {
		if k == "children" {
			continue
		}
		//if k != "style" && k != "className" && !strings.HasSuffix(k, "data-") {
		//	continue
		//}

		s.WriteString(" ")

		if k == "className" {
			s.WriteString("class")
		} else {
			s.WriteString(k)
		}
		s.WriteString(`=`)

		switch k {
		case "className":
			if val != nil {
				s.WriteString(`"`)
				v.renderClassName(s, val, true)
				s.WriteString(`"`)
			}
		case "style":
			s.WriteString(`"`)
			v.renderStyle(s, val)
			s.WriteString(`"`)
		default:
			v.renderAttrValue(s, val)
		}
	}
}

func (v VDom) renderAttrValue(s *strings.Builder, val interface{}) {
	switch t := val.(type) {
	case string:
		s.WriteString(`"`)
		s.WriteString(template.HTMLEscapeString(t))
		s.WriteString(`"`)
	default:
		s.WriteString(`"`)
		bs, err := json.Marshal(val)
		if err != nil {
			panic(err)
		}
		s.WriteString(template.HTMLEscapeString(string(bs)))
		s.WriteString(`"`)
	}
}

func (v VDom) renderClassName(s *strings.Builder, className interface{}, isFirst bool) {
	switch t := className.(type) {
	case []interface{}:
		for index, i := range t {
			v.renderClassName(s, i, isFirst && index == 0)
		}
	case string:
		if !isFirst {
			s.WriteString(" ")
		}
		s.WriteString(strings.Trim(t, " "))
	}
}

func snakeString(s string) string {
	data := make([]byte, 0, len(s)*3/2)
	j := false
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 && d >= 'A' && d <= 'Z' && j {
			data = append(data, '-')
		}
		if d != '-' {
			j = true
		}
		data = append(data, d)
	}
	return strings.ToLower(string(data[:]))
}

func (v VDom) renderStyle(s *strings.Builder, val interface{}) {
	isFirst := true
	switch t := val.(type) {
	case map[string]interface{}:
		for k, v := range t {
			if isFirst {
				isFirst = false
			} else {
				s.WriteString(" ")
			}
			// /node_modules/react-dom/cjs/react-dom-server-legacy.node.development.js hyphenateStyleName
			s.WriteString(snakeString(k))
			s.WriteString(":")
			s.WriteString(" ")
			s.WriteString(fmt.Sprintf("%v", v))
			s.WriteString(";")
		}
	default:
		panic(val)
	}
}

func (v VDom) printChild(s *strings.Builder, indent int, c interface{}) {
	switch t := c.(type) {
	case []interface{}:
		for _, c := range t {
			v.printChild(s, indent, c)
		}
	case map[string]interface{}:
		s.WriteString(VDom(t).string(indent + 1))
	default:
		v.printIndent(s, indent+1)

		s.WriteString(fmt.Sprintf("\"%v\"", t))
		s.WriteString("\n")
	}
}
func (v VDom) printAttr(s *strings.Builder, attr interface{}) {
	if attr == nil {
		return
	}
	m := map[string]interface{}{}
	for k, v := range attr.(map[string]interface{}) {
		if k == "children" {
			continue
		}
		m[k] = v
	}
	if len(m) == 0 {
		return
	}

	s.WriteString(fmt.Sprintf(" %+v", m))
}
func (v VDom) printIndent(s *strings.Builder, indent int) {
	s.WriteString(strings.Repeat("  |", indent))
}

func (v VDom) String() string {
	return v.string(0)
}

func (v VDom) string(indent int) string {
	var s strings.Builder

	nodeName := v["nodeName"].(string)
	attr := v["attributes"]
	var children interface{}
	if attr != nil {
		ci := attr.(map[string]interface{})["children"]
		if ci != nil {
			children = ci
		}
	}
	v.printIndent(&s, indent)

	if v["jsxs"] != nil {
		s.WriteString(fmt.Sprintf("[%v]", nodeName))
	} else {
		s.WriteString(fmt.Sprintf("<%v>", nodeName))
	}

	v.printAttr(&s, attr)
	s.WriteString("\n")

	if children != nil {
		v.printChild(&s, indent, children)
	}

	return s.String()
}

func (v VDom) renderChildren(s *strings.Builder, c interface{}) {
	switch t := c.(type) {
	case string:
		s.WriteString(t)
	case map[string]interface{}:
		s.WriteString(VDom(t).Render())
	case []interface{}:
		for _, c := range t {
			if c != nil {
				v.renderChildren(s, c)
			}
		}
	default:
		s.WriteString(fmt.Sprintf("%v", c))
	}
}

func (v VDom) Render() string {
	var s strings.Builder
	v.render(&s)
	return s.String()
}

func (v VDom) render(s *strings.Builder) {
	i := v["nodeName"]
	nodeName, _ := i.(string)
	attr := v["attributes"]
	var children interface{}
	if attr != nil {
		ci := attr.(map[string]interface{})["children"]
		if ci != nil {
			children = ci
		}
	}
	// Fragment 只渲染子节点
	if nodeName == "" && children != nil {
		v.renderChildren(s, children)
		return
	}

	selfclose := false
	switch nodeName {
	// Omitted close tags
	case "input":
		selfclose = true
	case "area", "base", "br", "col", "embed", "hr", "img", "keygen", "link", "meta", "param", "source", "track", "wbr":
		selfclose = true
	case "html":
		s.WriteString("<!DOCTYPE html>")
	}

	if nodeName != "" {
		s.WriteString("<")
		s.WriteString(nodeName)
		if attr != nil {
			v.RenderAttributes(s, attr.(map[string]interface{}))
		}
	}

	if selfclose {
		s.WriteString("/>")
		// 自闭合标签没有 children
		return
	}

	s.WriteString(">")
	if children != nil {
		v.renderChildren(s, children)
	}

	s.WriteString(fmt.Sprintf("</%v>", nodeName))

	return
}
