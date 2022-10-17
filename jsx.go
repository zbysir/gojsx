package jsx

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dop251/goja"
	"github.com/zbysir/gojsx/internal/js"
	"github.com/zbysir/gojsx/internal/pkg/goja_nodejs/console"
	"github.com/zbysir/gojsx/internal/pkg/goja_nodejs/require"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

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

func mD5(v []byte) string {
	m := md5.New()
	m.Write(v)
	return hex.EncodeToString(m.Sum(nil))
}

type RunJsParams struct {
	Fs           fs.FS
	FileName     string
	Src          []byte
	Transform    bool
	GlobalValues map[string]interface{}
	NoCache      bool
}

func (j *Jsx) RunJs(params *RunJsParams) (v goja.Value, err error) {
	vm, err := j.getVm()
	if err != nil {
		return nil, err
	}
	defer j.putVm(vm)

	fileSys := params.Fs
	if fileSys == nil {
		fileSys = StdFileSystem{}
	}

	vm.registry.SrcLoader = j.registryLoader(fileSys)

	if params.NoCache {
		vm.registry.Enable(vm.vm) // to clear modules cache
		vm.registry.Clear()       // to clear compiled cache
	}

	for k, v := range params.GlobalValues {
		err = vm.vm.Set(k, v)
		if err != nil {
			return nil, err
		}
	}

	return j.runJs(vm.vm, params.FileName, params.Src, params.Transform)
}

func (j *Jsx) runJs(vm *goja.Runtime, fileName string, src []byte, transform bool) (v goja.Value, err error) {
	if transform {
		src, err = j.tr.Transform(fileName, src)
		if err != nil {
			return nil, fmt.Errorf("load file (%s) error :%w", fileName, err)
		}
	}

	v, err = vm.RunScript(fileName, string(src))
	if err != nil {
		// fix Invalid module message
		if strings.Contains(err.Error(), "Invalid module") {
			mod := strings.TrimSuffix(j.lastLoadModule, "/index.json")
			err = errors.New(strings.ReplaceAll(err.Error(), "Invalid module", fmt.Sprintf("Invalid module '%v'", mod)))
		}
	}
	return v, err
}

type MountEndpoint struct {
	Endpoint  string
	Component string
	Props     interface{}
}

type versionedVm struct {
	once     sync.Once
	vm       *goja.Runtime
	registry *require.Registry
	version  int32
}

func (j *Jsx) getVm() (*versionedVm, error) {
	vm, err := j.vmPool.Get()
	if err != nil {
		return nil, fmt.Errorf("pool.Get error: %w", err)
	}

	vm.once.Do(func() {
		vm.registry.Enable(vm.vm)
		console.Enable(vm.vm, nil)
	})

	return vm, nil
}

func (j *Jsx) putVm(v *versionedVm) error {
	return j.vmPool.Put(v)
}

type RenderParam struct {
	Fs      fs.FS
	File    string
	Props   interface{}
	NoCache bool
}

// Render a component to html
func (j *Jsx) Render(p *RenderParam) (n string, err error) {
	res, err := j.RunJs(&RunJsParams{
		Fs:           p.Fs,
		FileName:     "root.js",
		Src:          []byte(fmt.Sprintf(`require("%v").default(props)`, p.File)),
		Transform:    false,
		GlobalValues: map[string]interface{}{"props": p.Props},
		NoCache:      p.NoCache,
	})
	if err != nil {
		return "", err
	}

	vdom := tryToVDom(res.Export())
	return vdom.Render(), nil
}

func tryToVDom(i interface{}) VDom {
	switch t := i.(type) {
	case map[string]interface{}:
		return t
	}

	return VDom{}
}

func (j *Jsx) RegisterModule(name string, obj map[string]interface{}) {
	require.RegisterNativeModule(name, func(runtime *goja.Runtime, module *goja.Object) {
		o := module.Get("exports").(*goja.Object)
		for k, v := range obj {
			_ = o.Set(k, v)
		}
	})

}

type Jsx struct {
	//vm *goja.Runtime
	vmPool *tPool[*versionedVm]
	tr     Transformer

	// goja is not goroutine-safe
	lock sync.Mutex
	//sourceFs fs.FS

	debug bool

	lastLoadModule string

	cache SourceCache
}

type StdFileSystem struct {
}

func (f StdFileSystem) Open(name string) (fs.File, error) {
	return os.Open(name)
}

type Option struct {
	SourceCache SourceCache
	Debug       bool // enable to get more log
	VmMaxTotal  int
}

func NewJsx(op Option) (*Jsx, error) {
	var transformer Transformer = NewEsBuildTransform(false)

	if op.VmMaxTotal <= 0 {
		op.VmMaxTotal = 20
	}

	j := &Jsx{
		vmPool: newTPool(op.VmMaxTotal, func() *versionedVm {
			vm := goja.New()
			vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

			if op.Debug {
				log.Printf("new vm")
			}
			return &versionedVm{
				once:     sync.Once{},
				vm:       vm,
				registry: require.NewRegistry(),
				version:  -1,
			}
		}),
		tr:    transformer,
		lock:  sync.Mutex{},
		debug: op.Debug,
		cache: op.SourceCache,
	}

	return j, nil
}

func (j *Jsx) registryLoader(filesys fs.FS) func(path string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		var fileBody []byte
		//var filePath string

		// 只支持转换 js/ts/tsx/jsx 文件格式
		needTrans := false
		if j.debug {
			fmt.Printf("tryload: %v\n", path)
		}
		j.lastLoadModule = path

		s := time.Now()

		if strings.HasSuffix(path, "node_modules/react/jsx-runtime") {
			fileBody = js.JsxRuntime
			needTrans = true
		}

		if fileBody == nil {
			find := false
			trySuffix := []string{""}

			ext := filepath.Ext(path)
			switch ext {
			case ".js":
				needTrans = true
				trySuffix = append(trySuffix, ".jsx")
				trySuffix = append(trySuffix, ".tsx")
				trySuffix = append(trySuffix, ".ts")
			case ".tsx", "jsx", "ts":
				needTrans = true
			}

			tryPath := path
			for _, p := range trySuffix {
				if p != "" {
					tryPath = strings.TrimSuffix(path, ".js") + p
				}
				bs, err := fs.ReadFile(filesys, tryPath)
				if err != nil {
					if errors.Is(err, fs.ErrNotExist) || strings.Contains(err.Error(), "is a directory") {
						continue
					}
					return nil, fmt.Errorf("can't load module: %v, error: %w", path, err)
				}
				find = true
				path = tryPath
				fileBody = bs
				break
			}
			if !find {
				return nil, require.ModuleFileDoesNotExistError
			}
		}

		fmt.Printf("load: %v", path)

		var err error
		if needTrans {
			fmt.Printf(" transform")

			cacheKey := mD5([]byte(path))
			srcMd5 := mD5(fileBody)
			var cached bool
			if j.cache != nil {
				fi, exist, err := j.cache.Get(cacheKey)
				if err != nil {
					return nil, err
				}
				if exist && fi.SrcMd5 == srcMd5 {
					cached = true
					fileBody = fi.Body
				}
			}

			if cached {
				fmt.Printf(" cached")
			} else {
				fileBody, err = j.tr.Transform(path, fileBody)
				if err != nil {
					return nil, fmt.Errorf("load file (%s) error :%w", path, err)
				}

				if j.cache != nil {
					err = j.cache.Set(cacheKey, &Source{
						SrcMd5:    srcMd5,
						Body:      fileBody,
						CreatedAt: "",
					})
					if err != nil {
						return nil, nil
					}
				}
			}
		}
		fmt.Printf(" %v\n", time.Now().Sub(s))

		return fileBody, nil
	}
}

//type VDom struct {
//	NodeName   string                 `json:"nodeName"`
//	Attributes map[string]interface{} `json:"attributes"`
//}

type VDom map[string]interface{}

// A few React string attributes have a different name.
// This is a mapping from React prop names to the attribute names.
var propsToAttr = map[string]string{
	"acceptCharset": "accept-charset",
	"className":     "class",
	"htmlFor":       "for",
	"httpEquiv":     "http-equiv",
}

func sortMap(ps map[string]interface{}, f func(k string, v interface{})) {
	keys := make([]string, 0, len(ps))
	for k := range ps {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		f(k, ps[k])
	}
}

func (v VDom) renderAttributes(s *strings.Builder, ps map[string]interface{}) {
	if len(ps) == 0 {
		return
	}

	// 如果 attr 里同时存在 class 和 className，则会将 class 放到 className 里统一处理。
	if c, ok := ps["class"]; ok {
		if cn, ok := ps["className"]; ok {
			ps["className"] = []interface{}{cn, c}
		} else {
			ps["className"] = c
		}

		delete(ps, "class")
	}

	// 排序
	// TODO 考虑直接使用 goja.Object 用作参数，不直接使用 Export 出来的 map，这样能保留字段排序。
	sortMap(ps, func(k string, val interface{}) {
		if k == "children" || k == "dangerouslySetInnerHTML" {
			return
		}

		s.WriteString(" ")

		if n, ok := propsToAttr[k]; ok {
			s.WriteString(n)
		} else {
			s.WriteString(k)
		}
		s.WriteString(`=`)

		switch k {
		case "className":
			s.WriteString(`"`)
			if val != nil {
				v.renderClassName(s, val, true)
			}
			s.WriteString(`"`)
		case "style":
			s.WriteString(`"`)
			v.renderStyle(s, val)
			s.WriteString(`"`)
		default:
			v.renderAttrValue(s, val)
		}
	})
}

func (v VDom) renderAttrValue(s *strings.Builder, val interface{}) {
	// 只支持 string/int
	switch t := val.(type) {
	case string:
		s.WriteString(`"`)
		s.WriteString(template.HTMLEscapeString(t))
		s.WriteString(`"`)
	case int64, int32, int16, int8, float64, float32:
		s.WriteString(`"`)
		s.WriteString(fmt.Sprintf("%v", t))
		s.WriteString(`"`)
	}
}

// cleanClass delete \n and extra space
func cleanClass(c string) string {
	var s strings.Builder
	n := len(c)
	r := 0

	cx := []rune(c)

	for r < n {
		if cx[r] == '\n' {
			cx[r] = ' '
		}

		if cx[r] == ' ' && (r+1 == n || (cx[r+1] == ' ' || cx[r+1] == '\n')) {
		} else {
			if s.Len() != 0 || cx[r] != ' ' {
				s.WriteRune(cx[r])
			}
		}
		r++
	}

	return s.String()
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
		s.WriteString(template.HTMLEscapeString(cleanClass(t)))
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
		s.WriteString(template.HTMLEscapeString(t))
	case map[string]interface{}:
		s.WriteString(VDom(t).Render())
	case []interface{}:
		for _, c := range t {
			if c != nil {
				v.renderChildren(s, c)
			}
		}
	default:
		s.WriteString(template.HTMLEscapeString(fmt.Sprintf("%v", c)))
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
	var attrMap map[string]interface{}
	if attr != nil {
		attrMap = attr.(map[string]interface{})
		ci := attrMap["children"]
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
			v.renderAttributes(s, attrMap)
		}
	}

	if selfclose {
		s.WriteString("/>")
		// 自闭合标签没有 children
		return
	}

	s.WriteString(">")
	html, ok := lockupMap[map[string]interface{}](attrMap, "dangerouslySetInnerHTML")
	if ok {
		h, ok := lockupMap[string](html, "__html")
		if ok {
			s.WriteString(h)
		} else {
			v.renderChildren(s, html)
		}
	} else {
		if children != nil {
			v.renderChildren(s, children)
		}
	}

	s.WriteString(fmt.Sprintf("</%v>", nodeName))

	return
}

func lockupMapInterface(m interface{}, keys ...string) (interface{}, bool) {
	if len(keys) == 0 {
		return m, true
	}
	mm, ok := m.(map[string]interface{})
	if !ok {
		return nil, false
	}
	i, ok := mm[keys[0]]
	if !ok {
		return nil, false
	}
	return lockupMapInterface(i, keys[1:]...)
}

func lockupMap[T any](m interface{}, keys ...string) (t T, b bool) {
	m, ok := lockupMapInterface(m, keys...)
	if ok {
		if m, ok := m.(T); ok {
			return m, true
		}
	}
	return t, false
}
