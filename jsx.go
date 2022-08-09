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
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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

func (j *Jsx) RunJs(fileName string, src []byte, transform bool) (v goja.Value, err error) {
	vm, err := j.getVm()
	if err != nil {
		return nil, err
	}
	defer j.putVm(vm)

	return j.runJs(vm.vm, fileName, src, transform)
}

func (j *Jsx) runJs(vm *goja.Runtime, fileName string, src []byte, transform bool) (v goja.Value, err error) {
	if transform {
		src, err = j.tr.Transform(fileName, src)
		if err != nil {
			return nil, err
		}
	}

	v, err = vm.RunScript(fileName, string(src))
	if err != nil {
		// fix Invalid module message
		if strings.Contains(err.Error(), "Invalid module") {
			err = errors.New(strings.ReplaceAll(err.Error(), "Invalid module", fmt.Sprintf("Invalid module '%v'", j.lastLoadModule)))
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
	vm      *goja.Runtime
	version int32
}

func (j *Jsx) getVm() (*versionedVm, error) {
	vm, err := j.vmPool.Get()
	if err != nil {
		return nil, fmt.Errorf("pool.Get error: %w", err)
	}

	// pool 元素是协程安全的，不必考虑并发
	if vm.version != j.version {
		j.initVm(vm.vm)
		vm.version = j.version
	}
	return vm, nil
}

func (j *Jsx) putVm(v *versionedVm) error {
	return j.vmPool.Put(v)
}

// Render a component to html
func (j *Jsx) Render(component string, props interface{}) (n string, err error) {
	vm, err := j.getVm()
	if err != nil {
		return "", err
	}
	defer j.putVm(vm)

	err = vm.vm.Set("props", vm.vm.ToValue(props))
	if err != nil {
		return "", err
	}
	res, err := j.runJs(vm.vm, "root.js", []byte(fmt.Sprintf(`require("%v").default(props)`, component)), false)
	if err != nil {
		return "", err
	}

	vdom := VDom{}
	err = vm.vm.ExportTo(res, &vdom)
	if err != nil {
		return "", err
	}
	//fmt.Printf("vdom: \n%+v", vdom)

	return vdom.Render(), nil
}

func (j *Jsx) RegisterModule(name string, obj map[string]interface{}) {
	if j.module == nil {
		j.module = map[string]func(runtime *goja.Runtime, module *goja.Object){}
	}

	j.module[name] = func(runtime *goja.Runtime, module *goja.Object) {
		o := module.Get("exports").(*goja.Object)
		for k, v := range obj {
			_ = o.Set(k, v)
		}
	}

	j.RefreshRegistry(nil)
}

type Jsx struct {
	//vm *goja.Runtime
	vmPool *tPool[*versionedVm]
	tr     Transformer

	// goja is not goroutine-safe
	lock     sync.Mutex
	sourceFs fs.FS

	debug bool

	lastLoadModule string

	// 额外注入的 module
	module map[string]func(runtime *goja.Runtime, module *goja.Object)

	cache SourceCache

	version int32
}

type StdFileSystem struct {
}

func (f StdFileSystem) Open(name string) (fs.File, error) {
	return os.Open(name)
}

type Option struct {
	SourceCache SourceCache
	SourceFs    fs.FS
	Debug       bool // enable to get more log
	Transformer Transformer
	VmMaxTotal  int
}

func NewJsx(op Option) (*Jsx, error) {
	var transformer Transformer = NewEsBuildTransform(true)
	if op.Transformer != nil {
		transformer = op.Transformer
	}

	if op.SourceFs == nil {
		op.SourceFs = StdFileSystem{}
	}

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
				vm:      vm,
				version: -1,
			}
		}),
		tr:       transformer,
		lock:     sync.Mutex{},
		sourceFs: op.SourceFs,
		debug:    op.Debug,
		cache:    op.SourceCache,
	}

	j.RefreshRegistry(nil)
	return j, nil
}

func (j *Jsx) registryLoader(path string) ([]byte, error) {
	var fileBody []byte
	//var filePath string
	needTrans := false
	if j.debug {
		fmt.Printf("tryload: %v\n", path)
	}
	j.lastLoadModule = path

	s := time.Now()
	if strings.Contains(path, "node_modules/react/jsx-runtime") {
		fileBody = js.JsxRuntime
		//filePath = path
		needTrans = true
	} else {
		find := false
		trySuffix := []string{""}
		if strings.HasSuffix(path, ".js") {
			needTrans = true
			trySuffix = append(trySuffix, ".jsx")
			trySuffix = append(trySuffix, ".tsx")
			trySuffix = append(trySuffix, ".ts")
		} else if strings.HasSuffix(path, ".ts") {
			needTrans = true
		}
		tryPath := path
		for _, p := range trySuffix {
			if p != "" {
				tryPath = strings.TrimSuffix(path, ".js") + p
			}
			bs, err := fs.ReadFile(j.sourceFs, tryPath)
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
				return nil, err
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

// RefreshRegistry will load module from file instead of cache
// 当文件更改时，可以调用 RefreshRegistry 来拿到最新的文件
func (j *Jsx) RefreshRegistry(newFs fs.FS) {
	atomic.AddInt32(&j.version, 1)
	if newFs != nil {
		j.sourceFs = newFs
	}
}

func (j *Jsx) initVm(vm *goja.Runtime) {
	if j.debug {
		log.Printf("initVm")
	}
	registry := require.NewRegistryWithLoader(j.registryLoader)
	registry.Enable(vm)
	console.Enable(vm)

	if j.module != nil {
		for k, obj := range j.module {
			registry.RegisterNativeModule(k, obj)
		}
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

	for _, k := range keys {
		f(k, ps[k])
	}
}

func (v VDom) renderAttributes(s *strings.Builder, ps map[string]interface{}) {
	if len(ps) == 0 {
		return
	}

	// 排序
	sortMap(ps, func(k string, val interface{}) {
		if k == "children" {
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
	})
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
			v.renderAttributes(s, attr.(map[string]interface{}))
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
