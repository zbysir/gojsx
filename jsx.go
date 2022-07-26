package ticktick

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/zbysir/gojsx/internal/js"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Transformer 负责将高级语法（包括 jsx，ts）转为 goja 能运行的 ES5.1
type Transformer struct {
	vm *goja.Runtime
}

func NewTransformer() (*Transformer, error) {
	vm := goja.New()

	require.NewRegistry().Enable(vm)
	console.Enable(vm)

	_, err := vm.RunScript("babel", js.Babel)
	if err != nil {
		return nil, err
	}
	return &Transformer{
		vm: vm,
	}, nil
}

func (t *Transformer) transform(filePath string, c string) (string, error) {
	t.vm.Set("code", c)
	_, name := filepath.Split(filePath)
	t.vm.Set("filepath", filePath)
	t.vm.Set("filename", name)

	v, err := t.vm.RunString(fmt.Sprintf(`Babel.transform(code, { presets: ["react","es2015"], sourceMaps: 'inline', sourceFileName: filename, filename: filepath, plugins: [
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
  ] }).code`))
	if err != nil {
		return "", err
	}

	return v.String(), nil
}

func (j *Jsx) RunJs(fileName string, src string, transform bool) (v goja.Value, err error) {
	if transform {
		src, err = j.tr.transform(fileName, src)
		if err != nil {
			return nil, err
		}
		//fmt.Printf("comp: %v %v\n", fileName, c)
	}

	v, err = j.vm.RunString(src)
	return v, err
}

// Compile 预编译多个文件
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
	res, err := j.RunJs(component, fmt.Sprintf(`require("%v").default(props)`, component), false)
	if err != nil {
		return "", err
	}

	vdom := VDom{}
	err = j.vm.ExportTo(res, &vdom)
	if err != nil {
		return "", err
	}
	return vdom.Render(), nil
}

// Mount render component to tpl
func (j *Jsx) Mount(indexTpl string, es ...MountEndpoint) (h string, err error) {
	var oldnew []string
	for _, e := range es {
		n, err := j.Render(e.Component, e.Props)
		if err != nil {
			return "", err
		}
		oldnew = append(oldnew, e.Endpoint, n)
		//fmt.Printf("vdom: \n%+v", vdom)
	}
	re := strings.NewReplacer(oldnew...)
	s := re.Replace(indexTpl)
	return s, err
}

type Jsx struct {
	vm *goja.Runtime
	tr *Transformer

	// cache transform result, md5 => body
	transformCache map[[16]byte][]byte

	// goja is not goroutine-safe
	lock     sync.Mutex
	sourceFs fs.FS
}

type StdFileSystem struct {
}

func (f StdFileSystem) Open(name string) (fs.File, error) {
	return os.Open(name)
}

type Option interface {
	apply(jsx *Jsx)
}

type OptionFunc func(jsx *Jsx)

func (o OptionFunc) apply(jsx *Jsx) {
	o(jsx)
}

func WithFS(f fs.FS) Option {
	return OptionFunc(func(jsx *Jsx) {
		jsx.sourceFs = f
	})
}

func NewJsx(ops ...Option) (*Jsx, error) {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	transformer, err := NewTransformer()
	if err != nil {
		return nil, err
	}

	j := &Jsx{
		vm:             vm,
		tr:             transformer,
		lock:           sync.Mutex{},
		transformCache: map[[16]byte][]byte{},
		sourceFs:       StdFileSystem{},
	}
	for _, o := range ops {
		o.apply(j)
	}
	j.reload()
	return j, nil
}

func (j *Jsx) reload() {
	// 使用 transformer 预编译 src 的文件，能加快速度

	// 当文件更改时，可以新 new registry 来拿到最新的文件
	registry := require.NewRegistryWithLoader(func(path string) ([]byte, error) {
		var fileBody []byte
		var filePath string

		fmt.Printf("load: %v\n", path)
		if strings.Contains(path, "node_modules/react/jsx-runtime") {
			fileBody = js.Jsx
			filePath = "builtin"
		} else {
			filePath = path + ".jsx"
			bs, err := fs.ReadFile(j.sourceFs, filePath)
			if err != nil {
				return nil, fmt.Errorf("can't load module: %v, error: %w", path, err)
			}
			fileBody = bs
		}
		m5 := md5.Sum(fileBody)
		if x, ok := j.transformCache[m5]; ok {
			return x, nil
		}

		s, err := j.tr.transform(filePath, string(fileBody))
		if err != nil {
			return nil, err
		}
		j.transformCache[m5] = []byte(s)
		return []byte(s), nil
	})
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
		if k != "style" && k != "className" && !strings.HasSuffix(k, "data-") {
			continue
		}

		s.WriteString(" ")

		if k == "className" {
			s.WriteString("class")
		} else {
			s.WriteString(k)
		}
		s.WriteString(`="`)

		switch k {
		case "className":
			v.renderClassName(s, val, true)
		case "style":
			v.renderStyle(s, val)
		default:
			v.renderAttrValue(s, val)
		}

		s.WriteString(`"`)
	}
}

func (v VDom) renderAttrValue(s *strings.Builder, val interface{}) {
	bs, err := json.Marshal(val)
	if err != nil {
		panic(err)
	}
	s.Write(bs)
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
			s.WriteString(k)
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

	if children != nil {
		s.WriteString(fmt.Sprintf("<%v>", nodeName))
		v.printAttr(&s, attr)
		s.WriteString("\n")
		v.printChild(&s, indent, children)
	} else {
		s.WriteString(fmt.Sprintf("<%v>", nodeName))
		v.printAttr(&s, attr)
		s.WriteString("\n")
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
			v.renderChildren(s, c)
		}
	default:
		s.WriteString(fmt.Sprintf("%v", c))

	}
}

func (v VDom) Render() string {
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

	if children != nil {
		if nodeName != "" {
			s.WriteString("<")
			s.WriteString(nodeName)
			if attr != nil {
				v.RenderAttributes(&s, attr.(map[string]interface{}))
			}
			s.WriteString(">")
		}

		v.renderChildren(&s, children)

		if nodeName != "" {
			s.WriteString(fmt.Sprintf("<%v/>", nodeName))
		}
	} else {
		if nodeName != "" {
			s.WriteString("<")
			s.WriteString(nodeName)
			if attr != nil {
				v.RenderAttributes(&s, attr.(map[string]interface{}))
			}
			s.WriteString("><")
			s.WriteString(nodeName)
			s.WriteString("/>")
		}
	}

	return s.String()
}
