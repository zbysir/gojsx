package gojsx

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dop251/goja"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stoewer/go-strcase"
	"github.com/zbysir/gojsx/internal/js"
	"github.com/zbysir/gojsx/internal/pkg/goja_nodejs/console"
	"github.com/zbysir/gojsx/internal/pkg/goja_nodejs/require"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Jsx struct {
	vmPool *tPool[*vmWithRegistry]
	tr     Transformer

	// goja is not goroutine-safe
	lock sync.Mutex
	//sourceFs fs.FS

	debug bool

	cache SourceCache

	modulesCache *lru.Cache[string, *goja.Program]
}

type SourceCache interface {
	Get(key string) (f []byte, exist bool, err error)
	Set(key string, f []byte) (err error)
}

type Source struct {
	SrcMd5    string
	Body      []byte
	CreatedAt string
}

// TODO use LRU
type memSourceCache struct {
	m *sync.Map
}

func NewMemSourceCache() *memSourceCache {
	return &memSourceCache{m: &sync.Map{}}
}

func (m *memSourceCache) Get(key string) (body []byte, exist bool, err error) {
	i, ok := m.m.Load(key)
	if !ok {
		return nil, false, nil
	}

	return i.([]byte), true, nil
}

func (m *memSourceCache) Set(key string, body []byte) (err error) {
	m.m.Store(key, body)
	return nil
}

func mD5(v []byte) string {
	m := md5.New()
	m.Write(v)
	return hex.EncodeToString(m.Sum(nil))
}

type OptionExec interface {
	applyRunOptions(*execOptions)
}

type OptionRender interface {
	applyRenderOptions(*renderOptions)
}

type nativeModule struct {
	Path string
	Obj  map[string]interface{}
}

func (n nativeModule) applyRenderOptions(options *renderOptions) {
	options.NativeModules = append(options.NativeModules, n)
}

func (n nativeModule) applyRunOptions(options *execOptions) {
	options.NativeModules = append(options.NativeModules, n)
}

type execOptions struct {
	GlobalVars map[string]interface{}
	// cache modules value, does not respond to file changes if true.
	// 如果想要更改源码立即生效则不要设置 true，只有当一次性运行或生产环境为了更好的性能可以设置为 true。
	Cache            bool
	FileName         string
	AutoExecJsx      bool
	AutoExecJsxProps AutoExecJsxProps
	NativeModules    []nativeModule
}

// WithNativeModule 注意，由于有 vm 对象池公用 vm 的情况，所以只能保证同步执行的代码能正确拿到本次运行的值，如果是第一次运行导出 function 再执行的情况，可能拿到的是第二次运行指定的 Module。
func WithNativeModule(path string, obj map[string]interface{}) interface {
	OptionExec
	OptionRender
} {
	return nativeModule{
		Path: path,
		Obj:  obj,
	}
}

type AutoExecJsxProps interface{}

type renderOptions struct {
	Cache         bool
	NativeModules []nativeModule
}

type RunJsOption func(*execOptions)

type autoExecJsxOption struct {
	props AutoExecJsxProps
}

func (t autoExecJsxOption) applyRunOptions(options *execOptions) {
	options.AutoExecJsxProps = t.props
	options.AutoExecJsx = true
}

func WithAutoExecJsx(t AutoExecJsxProps) OptionExec {
	return autoExecJsxOption{props: t}
}

type cacheOption bool

func (c cacheOption) applyRunOptions(options *execOptions) {
	options.Cache = bool(c)
}

func (c cacheOption) applyRenderOptions(options *renderOptions) {
	options.Cache = bool(c)
}

func WithCache(cache bool) interface {
	OptionExec
	OptionRender
} {
	return cacheOption(cache)
}

type globalVarOption struct {
	k string
	v interface{}
}

func (g globalVarOption) applyRunOptions(options *execOptions) {
	if options.GlobalVars == nil {
		options.GlobalVars = map[string]interface{}{}
	}

	options.GlobalVars[g.k] = g.v
}

// WithGlobalVar 注意，由于有 vm 对象池公用 vm 的情况，所以只能保证同步执行的代码能正确拿到本次运行的值，如果是第一次运行导出 function 再执行的情况，可能拿到的是第二次运行指定的 GlobalVar。
func WithGlobalVar(k string, v interface{}) OptionExec {
	return globalVarOption{
		k: k,
		v: v,
	}
}

type fileNameOption string

func (f fileNameOption) applyRunOptions(options *execOptions) {
	options.FileName = string(f)
}

func WithFileName(fn string) OptionExec {
	return fileNameOption(fn)
}

// ExecCode code 需要是 ESModule 格式，如 export default () => <></>
func (j *Jsx) ExecCode(src []byte, opts ...OptionExec) (ex *ModuleExport, err error) {
	var p execOptions
	for _, o := range opts {
		o.applyRunOptions(&p)
	}

	vm, err := j.getVm()
	if err != nil {
		return nil, err
	}
	defer j.putVm(vm)

	if !p.Cache {
		vm.requireModule.Clean() // to clear modules cache
	}

	for _, mod := range p.NativeModules {
		vm.registry.RegisterNativeModule(mod.Path, func(runtime *goja.Runtime, module *goja.Object) {
			o := module.Get("exports").(*goja.Object)
			for k, v := range mod.Obj {
				_ = o.Set(k, v)
			}
		})
	}

	for k, v := range p.GlobalVars {
		err = vm.vm.Set(k, v)
		if err != nil {
			return nil, err
		}
	}

	fileName := p.FileName
	if fileName == "" {
		fileName = "root.js"
	}

	v, err := j.runJs(vm, fileName, src, TransformerFormatIIFE)
	if err != nil {
		return
	}
	ex, err = parseModuleExport(exportGojaValue(v), p.AutoExecJsx, vm.vm)
	if err != nil {
		return
	}

	if p.AutoExecJsx {
		switch t := ex.Default.(type) {
		case Callable:
			v, err := t(p.AutoExecJsxProps)
			if err != nil {
				return nil, err
			}
			vd, _ := tryToVDom(v.Export())
			if vd != nil {
				ex.Default = vd
			}
		}
	}
	return
}

func (j *Jsx) runJs(vm *vmWithRegistry, fileName string, src []byte, transform TransformerFormat) (v goja.Value, err error) {
	if transform != 0 {
		key := mD5(src)
		s, exist, err := j.cache.Get(key)
		if err != nil {
			return nil, err
		}
		if exist {
			src = s
		} else {
			src, err = j.tr.Transform(fileName, src, transform)
			if err != nil {
				return nil, fmt.Errorf("load file error: %w", err)
			}
			j.cache.Set(key, src)
		}
	}

	// 缓存 compile
	var p *goja.Program
	cp, ok := j.modulesCache.Get(string(src))
	if ok {
		p = cp
	} else {
		p, err = goja.Compile(fileName, string(src), false)
		if err != nil {
			return
		}
		j.modulesCache.Add(string(src), p)
	}

	v, err = vm.vm.RunProgram(p)
	if err != nil {
		return nil, PrettifyException(err)
	}
	return v, nil
}

type vmWithRegistry struct {
	once          sync.Once
	vm            *goja.Runtime
	registry      *require.Registry
	requireModule *require.RequireModule
}

func (j *Jsx) getVm() (*vmWithRegistry, error) {
	vm, err := j.vmPool.Get()
	if err != nil {
		return nil, fmt.Errorf("pool.Get error: %w", err)
	}

	return vm, nil
}

func (j *Jsx) putVm(v *vmWithRegistry) error {
	return j.vmPool.Put(v)
}

// Render a component to html
func (j *Jsx) Render(file string, props interface{}, opts ...OptionRender) (n string, err error) {
	n, _, err = j.RenderCtx(file, props, opts...)
	return n, err
}

// RenderCode code to html
func (j *Jsx) RenderCode(code []byte, props interface{}, opts ...OptionExec) (n string, ctx *RenderCtx, err error) {
	opts = append(opts, WithAutoExecJsx(props))
	ex, err := j.ExecCode(code, opts...)
	if err != nil {
		return
	}

	switch t := ex.Default.(type) {
	case VDom:
		s, ctx := t.Render()
		return s, ctx, nil
	default:
		log.Panicf("unspoort export type: %T, shound be a vdom", ex.Default)
	}
	return
}

// RenderCtx a component to html
func (j *Jsx) RenderCtx(file string, props interface{}, opts ...OptionRender) (n string, ctx *RenderCtx, err error) {
	var p renderOptions
	for _, o := range opts {
		o.applyRenderOptions(&p)
	}

	var eo = []OptionExec{
		WithCache(p.Cache), WithAutoExecJsx(props),
	}
	for _, m := range p.NativeModules {
		eo = append(eo, WithNativeModule(m.Path, m.Obj))
	}
	ex, err := j.Exec(file, eo...)
	if err != nil {
		return
	}

	switch t := ex.Default.(type) {
	case VDom:
		s, ctx := t.Render()
		return s, ctx, nil
	default:
		panic(t)
	}
	return
}

// 和 goja 自己的 export 不一样的是，不会尝试导出单个变量为 golang 基础类型，而是保留 goja.Value，只是展开 Object
func exportGojaValue(i interface{}) interface{} {
	switch t := i.(type) {
	case *goja.Object:
		switch t.ExportType() {
		case reflect.TypeOf(map[string]interface{}{}):
			m := map[string]interface{}{}
			for _, k := range t.Keys() {
				m[k] = exportGojaValue(t.Get(k))
			}
			return m
		case reflect.TypeOf([]interface{}{}):
			arr := make([]interface{}, len(t.Keys()))
			for _, k := range t.Keys() {
				index, _ := strconv.ParseInt(k, 10, 64)
				arr[index] = exportGojaValue(t.Get(k))
			}
			return arr
		}
	case interface{ Export() interface{} }:
		return t.Export()
	}

	return i
}

func (j *Jsx) Exec(file string, opts ...OptionExec) (ex *ModuleExport, err error) {
	var p execOptions
	for _, o := range opts {
		o.applyRunOptions(&p)
	}

	var code = []byte(fmt.Sprintf(`module.exports = require("%v")`, file))

	ex, err = j.ExecCode(code, opts...)
	if err != nil {
		return
	}

	return
}

func (V VDom) _default() {
}

type Callable func(args ...interface{}) (v goja.Value, err error)

func (c Callable) _default() {
}

type Any struct {
	Any interface{}
}

func (a Any) _default() {

}

type ExportDefault interface {
	_default()
}

type ModuleExport struct {
	// One of VDom, Callable, Any
	// VDom if WithAutoExecJsx
	// Callable if export a function
	Default ExportDefault
	Exports map[string]interface{}
}

type VDomOrInterface struct {
	// VDom     VDom
	Callable goja.Callable

	Any interface{}
}

//func (v *VDomOrInterface) render(props goja.Value) (string, error) {
//	if v.Callable != nil {
//		val, err := v.Callable(props)
//		if err != nil {
//			return "", err
//		}
//		vdom, err := tryToVDom(val.Export())
//		if err != nil {
//			return "", err
//		}
//
//		return vdom.Render(), nil
//	}
//
//	return "", nil
//}

func parseModuleExport(i interface{}, tryVDom bool, vm *goja.Runtime) (m *ModuleExport, err error) {
	var vDomOrInterface ExportDefault

	switch t := i.(type) {
	case map[string]interface{}:
		switch t := t["default"].(type) {
		case *goja.Object:
			c, ok := AssertFunction(t)
			if ok {
				vDomOrInterface = Callable(func(args ...interface{}) (v goja.Value, err error) {
					as := make([]goja.Value, len(args))
					for i, arg := range args {
						as[i] = vm.ToValue(arg)
					}
					return c(nil, as...)
				})
			} else {
				vDomOrInterface = Any{t.Export()}
			}
		default:
			if tryVDom {
				// for WithAutoExecJsx
				v, _ := tryToVDom(t)
				if v != nil {
					vDomOrInterface = v
				}
			}

			if vDomOrInterface == nil {
				vDomOrInterface = Any{t}
			}
		}

		delete(t, "default")
		return &ModuleExport{
			Default: vDomOrInterface,
			Exports: t,
		}, nil
	default:
		return nil, fmt.Errorf("export value type expect 'map[string]interface{}', actual '%T'", i)
	}
}

func tryToVDom(i interface{}) (VDom, error) {
	if i == nil {
		return nil, nil
	}
	switch t := i.(type) {
	case map[string]interface{}:
		return t, nil
	default:
		return nil, fmt.Errorf("ToVDom error: export value type expect 'map[string]interface{}', actual '%T'", i)
	}
}

func (j *Jsx) RegisterModule(name string, obj map[string]interface{}) {
	require.RegisterNativeModule(name, func(runtime *goja.Runtime, module *goja.Object) {
		o := module.Get("exports").(*goja.Object)
		for k, v := range obj {
			_ = o.Set(k, v)
		}
	})
}

type stdFileSystem struct {
}

var StdFileSystem = stdFileSystem{}

func (f stdFileSystem) Open(name string) (fs.File, error) {
	return os.Open(name)
}

type Option struct {
	SourceCache SourceCache
	Debug       bool // enable to get more log
	// 最多的 vm 对象数量，指定为 1 表示只会同时有一个 vm 运行，默认为 2000
	VmMaxTotal  int
	Transformer Transformer
	// Fs 没办法做到每次执行代码时指定，因为 require 可能会发生在异步 function 里，fs 改变会导致加载文件错误
	Fs fs.FS // default is StdFileSystem

	// GojaFieldNameMapper Specify the mapping of field names in go struct and js.
	// via: https://github.com/dop251/goja#mapping-struct-field-and-method-names
	GojaFieldNameMapper goja.FieldNameMapper
}

var defaultFieldNameMapper = TagFieldNameMapper("json", true, true)

func NewJsx(op Option) (*Jsx, error) {
	if op.VmMaxTotal <= 0 {
		op.VmMaxTotal = 2000
	}

	if op.Transformer == nil {
		op.Transformer = NewEsBuildTransform(EsBuildTransformOptions{})
	}

	if op.SourceCache == nil {
		op.SourceCache = NewMemSourceCache()
	}

	if op.Fs == nil {
		op.Fs = StdFileSystem
	}
	if op.GojaFieldNameMapper == nil {
		op.GojaFieldNameMapper = defaultFieldNameMapper
	}

	jsProgramCache, err := lru.New[string, *goja.Program](100)
	if err != nil {
		return nil, err
	}

	j := &Jsx{
		vmPool: newTPool(op.VmMaxTotal, func() *vmWithRegistry {
			vm := goja.New()
			vm.SetFieldNameMapper(op.GojaFieldNameMapper)

			if op.Debug {
				log.Printf("new vm")
			}
			registry := require.NewRegistryWithLoader(registryLoader(op.Fs, op.SourceCache, op.Transformer))
			requireModule := registry.Enable(vm)

			console.Enable(vm, nil)

			return &vmWithRegistry{
				once:          sync.Once{},
				vm:            vm,
				registry:      registry,
				requireModule: requireModule,
			}
		}),
		tr:           op.Transformer,
		lock:         sync.Mutex{},
		debug:        op.Debug,
		cache:        op.SourceCache,
		modulesCache: jsProgramCache,
	}

	return j, nil
}

func registryLoader(fileSys fs.FS, cache SourceCache, tr Transformer) require.SourceLoader {
	return func(path string) ([]byte, error) {
		var fileBody []byte

		if strings.HasSuffix(path, "node_modules/react/jsx-runtime") {
			fileBody = js.JsxRuntime
		}

		if fileBody == nil {
			find := false
			trySuffix := []string{""}

			ext := filepath.Ext(path)
			switch ext {
			case ".js":
				trySuffix = append(trySuffix, ".jsx")
				trySuffix = append(trySuffix, ".tsx")
				trySuffix = append(trySuffix, ".ts")
				trySuffix = append(trySuffix, ".md")
				trySuffix = append(trySuffix, ".mdx")
			}

			tryPath := path
			for _, p := range trySuffix {
				if p != "" {
					tryPath = strings.TrimSuffix(path, ".js") + p
				}
				bs, err := fs.ReadFile(fileSys, tryPath)
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

		var err error

		srcMd5 := mD5(fileBody)
		var cached bool
		if cache != nil {
			fi, exist, err := cache.Get(srcMd5)
			if err != nil {
				return nil, err
			}
			if exist {
				cached = true
				fileBody = fi
			}
		}

		if cached {

		} else {
			fileBody, err = tr.Transform(path, fileBody, TransformerFormatCommonJS)
			if err != nil {
				return nil, fmt.Errorf("load file error: %w", err)
			}

			if cache != nil {
				err = cache.Set(srcMd5, fileBody)
				if err != nil {
					return nil, nil
				}
			}
		}

		return fileBody, nil
	}
}

//type VDomx struct {
//	NodeName   string                 `json:"nodeName"`
//	Attributes map[string]interface{} `json:"attributes"`
//}

type VDom map[string]interface{}

// 处理 React 中标签语法与标准标签的对应关系。如将 strokeWidth 换为 stroke-width。
// 参考 react-dom/cjs/react-dom-server-legacy.node.development.js 实现
var propsToAttr = map[string]string{}
var boolAttr = map[string]bool{} // 如果是 boolAttr，当传递了 attr value 则会渲染，否则不会渲染整个 attr key

func init() {
	// A few React string attributes have a different name. This is a mapping from React prop names to the attribute names.
	for k, v := range map[string]string{
		"acceptCharset": "accept-charset",
		"className":     "class",
		"htmlFor":       "for",
		"httpEquiv":     "http-equiv",
	} {
		propsToAttr[k] = v
	}

	// This is a list of all SVG attributes that need special casing.
	svgAttr := []string{"accent-height", "alignment-baseline", "arabic-form", "baseline-shift", "cap-height", "clip-path", "clip-rule", "color-interpolation", "color-interpolation-filters", "color-profile", "color-rendering", "dominant-baseline", "enable-background", "fill-opacity", "fill-rule", "flood-color", "flood-opacity", "font-family", "font-size", "font-size-adjust", "font-stretch", "font-style", "font-variant", "font-weight", "glyph-name", "glyph-orientation-horizontal", "glyph-orientation-vertical", "horiz-adv-x", "horiz-origin-x", "image-rendering", "letter-spacing", "lighting-color", "marker-end", "marker-mid", "marker-start", "overline-position", "overline-thickness", "paint-order", "panose-1", "pointer-events", "rendering-intent", "shape-rendering", "stop-color", "stop-opacity", "strikethrough-position", "strikethrough-thickness", "stroke-dasharray", "stroke-dashoffset", "stroke-linecap", "stroke-linejoin", "stroke-miterlimit", "stroke-opacity", "stroke-width", "text-anchor", "text-decoration", "text-rendering", "underline-position", "underline-thickness", "unicode-bidi", "unicode-range", "units-per-em", "v-alphabetic", "v-hanging", "v-ideographic", "v-mathematical", "vector-effect", "vert-adv-y", "vert-origin-x", "vert-origin-y", "word-spacing", "writing-mode", "xmlns:xlink", "x-height"}
	for _, a := range svgAttr {
		propsToAttr[strcase.LowerCamelCase(a)] = a
	}

	// These attribute exists both in HTML and SVG. The attribute name is case-sensitive in SVG so we can't just use the React name like we do for attributes that exist only in HTML.
	for _, a := range []string{"tabIndex", "crossOrigin"} {
		propsToAttr[a] = a
	}

	// These are HTML boolean attributes.
	for _, a := range []string{"allowFullScreen", "async", "autoFocus", "autoPlay", "controls", "default", "defer", "disabled", "disablePictureInPicture", "disableRemotePlayback", "formNoValidate", "hidden", "loop", "noModule", "noValidate", "open", "playsInline", "readOnly", "required", "reversed", "scoped", "seamless", "itemScope"} {
		propsToAttr[a] = strings.ToLower(a)
		boolAttr[strings.ToLower(a)] = true
	}
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

var currHydrateId = 0

func renderAttributes(s *strings.Builder, ctx *RenderCtx, props map[string]interface{}) {
	if len(props) == 0 {
		return
	}

	// 如果 attr 里同时存在 class 和 className，则会将 class 放到 className 里统一处理。
	if c, ok := props["class"]; ok {
		if cn, ok := props["className"]; ok {
			props["className"] = []interface{}{cn, c}
		} else {
			props["className"] = c
		}

		delete(props, "class")
	}

	var hydrate = map[string]string{}
	// 转为 16 进制
	var hydrateId = fmt.Sprintf("%x", currHydrateId)
	currHydrateId++

	// 稳定顺序
	// TODO 考虑直接使用 goja.Object 用作参数，不直接使用 Export 出来的 map，这样能保留字段排序。
	sortMap(props, func(k string, val interface{}) {
		if k == "children" || k == "dangerouslySetInnerHTML" {
			return
		}
		if strings.HasPrefix(k, "hydrate") {
			s.WriteString(` data-hydrate="`)
			s.WriteString(hydrateId)
			s.WriteString(`"`)
			//
			//log.Printf("renderAttributes: %s,val: %#v", k, val)
			hydrate[k] = fmt.Sprintf(`%s`, val)
			return
		}
		switch k {
		case "className":
			s.WriteString(` class="`)
			if val != nil {
				renderClassName(s, val, true)
			}
			s.WriteString(`"`)
		case "style":
			s.WriteString(` style="`)
			renderStyle(s, val)
			s.WriteString(`"`)
		default:
			if n, ok := propsToAttr[k]; ok {
				k = n
			}

			if boolAttr[k] {
				tr := false
				switch t := val.(type) {
				case string:
					tr = strings.ToLower(t) == "true"
				case int, int32, int16, int8, int64, float64, float32:
					tr = val != 0
				case bool:
					tr = t
				}
				if tr {
					s.WriteString(" ")
					s.WriteString(k)
				}
			} else {
				vs := attributeValueToString(val)
				if vs != "" {
					s.WriteString(" ")
					if n, ok := propsToAttr[k]; ok {
						s.WriteString(n)
					} else {
						s.WriteString(k)
					}

					s.WriteString(`=`)
					s.WriteString(`"`)
					s.WriteString(vs)
					s.WriteString(`"`)
				}
			}
		}
	})

	if len(hydrate) > 0 {
		ctx.AddHydrate(hydrateId, hydrate)
	}
}

func renderAttributeValue(s *strings.Builder, val interface{}) {
	// 只支持 string/int
	switch t := val.(type) {
	case string:
		s.WriteString(`"`)
		s.WriteString(template.HTMLEscapeString(t))
		s.WriteString(`"`)
	case int, int64, int32, int16, int8, float64, float32:
		s.WriteString(`"`)
		s.WriteString(fmt.Sprintf("%v", t))
		s.WriteString(`"`)
	}
}

func attributeValueToString(val interface{}) string {
	// 只支持 string/int
	switch t := val.(type) {
	case string:
		return template.HTMLEscapeString(t)
	case int, int64, int32, int16, int8, float64, float32, bool, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf(`%v`, t)
	}
	return ""
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

func renderClassName(s *strings.Builder, className interface{}, isFirst bool) {
	switch t := className.(type) {
	case []interface{}:
		for index, i := range t {
			renderClassName(s, i, isFirst && index == 0)
		}
	case string:
		if !isFirst {
			s.WriteString(" ")
		}
		s.WriteString(template.HTMLEscapeString(cleanClass(t)))
	}
}

func renderStyle(s *strings.Builder, val interface{}) {
	isFirst := true
	switch t := val.(type) {
	case map[string]interface{}:
		// for style={{color: "red"}}
		sortMap(t, func(k string, v interface{}) {
			if isFirst {
				isFirst = false
			} else {
				s.WriteString(" ")
			}
			// /node_modules/react-dom/cjs/react-dom-server-legacy.node.development.js hyphenateStyleName
			s.WriteString(strcase.KebabCase(k))
			s.WriteString(":")
			s.WriteString(" ")
			s.WriteString(fmt.Sprintf("%v", v))
			s.WriteString(";")
		})
	case string:
		// for style=""
		s.WriteString(t)
	default:
		s.WriteString(fmt.Sprintf("%v", t))
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
	v.printIndent(&s, indent)
	s.WriteString(fmt.Sprintf("<%v>", nodeName))

	v.printAttr(&s, attr)
	s.WriteString("\n")

	if children != nil {
		v.printChild(&s, indent, children)
	}

	return s.String()
}

func (v VDom) Render() (string, *RenderCtx) {
	return Render(v)
}

type RenderCtx struct {
	// Hydrate 用于将组件的数据提取到单独的文件中。
	Hydrate map[string]map[string]string // id => [event type => event code]
}

// AddHydrate add hydrate
func (ctx *RenderCtx) AddHydrate(id string, props map[string]string) {
	if ctx.Hydrate == nil {
		ctx.Hydrate = map[string]map[string]string{}
	}
	ctx.Hydrate[id] = props
}

func Render(i interface{}) (string, *RenderCtx) {
	var s strings.Builder
	var ctx RenderCtx
	render(&s, &ctx, i)
	return s.String(), &ctx
}

func render(s *strings.Builder, ctx *RenderCtx, c interface{}) {
	var v map[string]interface{}

	switch t := c.(type) {
	case string:
		s.WriteString(template.HTMLEscapeString(t))
		return
	case []interface{}:
		for _, c := range t {
			if c != nil {
				render(s, ctx, c)
			}
		}
		return
	case map[string]interface{}:
		v = t
	case VDom:
		v = t
	default:
		s.WriteString(template.HTMLEscapeString(fmt.Sprintf("%v", c)))
		return
	}

	if v == nil {
		return
	}
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
	if nodeName == "" {
		if children != nil {
			render(s, ctx, children)
		}
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

	s.WriteString("<")
	s.WriteString(nodeName)
	if attr != nil {
		renderAttributes(s, ctx, attrMap)
	}

	if selfclose {
		s.WriteString("/>")
		// 自闭合标签没有 children
		return
	}

	s.WriteString(">")
	html, ok := lookupMap[map[string]interface{}](attrMap, "dangerouslySetInnerHTML")
	if ok {
		h, ok := lookupMap[string](html, "__html")
		if ok {
			s.WriteString(h)
		} else {
			render(s, ctx, html)
		}
	} else {
		if children != nil {
			render(s, ctx, children)
		}
	}

	s.WriteString(fmt.Sprintf("</%v>", nodeName))

}

func lookupMapI(m interface{}, keys ...string) (interface{}, bool) {
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
	return lookupMapI(i, keys[1:]...)
}

// lookupMap({a: {b: 1}}, "a", "b") => 1
func lookupMap[T any](m interface{}, keys ...string) (t T, b bool) {
	m, ok := lookupMapI(m, keys...)
	if ok {
		if m, ok := m.(T); ok {
			return m, true
		}
	}
	return t, false
}

// AssertFunction wrap goja.AssertFunction and add prettify error message
func AssertFunction(v goja.Value) (goja.Callable, bool) {
	c, ok := goja.AssertFunction(v)
	if !ok {
		return c, false
	}

	return func(this goja.Value, args ...goja.Value) (goja.Value, error) {
		val, err := c(this, args...)
		if err != nil {
			return val, PrettifyException(err)
		}
		return val, nil
	}, true
}
