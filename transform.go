package jsx

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/zbysir/gojsx/internal/js"
	"html/template"
	"path/filepath"
	"strings"
	"sync"
)

type Transformer interface {
	Transform(filePath string, code []byte) (out []byte, err error)
}

type EsBuildTransform struct {
	minify bool
}

func NewEsBuildTransform(minify bool) *EsBuildTransform {
	return &EsBuildTransform{minify: minify}
}

var defaultExtensionToLoaderMap = map[string]api.Loader{
	"":      api.LoaderJS, // default
	".js":   api.LoaderJS,
	".mjs":  api.LoaderJS,
	".cjs":  api.LoaderJS,
	".jsx":  api.LoaderJSX,
	".ts":   api.LoaderTS,
	".tsx":  api.LoaderTSX,
	".css":  api.LoaderCSS,
	".json": api.LoaderJSON,
	".txt":  api.LoaderText,
}

func (e EsBuildTransform) Transform(filePath string, code []byte) (out []byte, err error) {
	_, file := filepath.Split(filePath)
	ext := filepath.Ext(filePath)

	loader, ok := defaultExtensionToLoaderMap[ext]
	if !ok {
		return nil, fmt.Errorf("unsupport file extension(%s) for transform", ext)
	}

	//api.LoaderNone
	result := api.Transform(string(code), api.TransformOptions{
		Loader:            loader,
		Target:            api.ESNext,
		JSXMode:           api.JSXModeAutomatic,
		Format:            api.FormatCommonJS,
		Platform:          api.PlatformNode,
		Sourcemap:         api.SourceMapInline,
		SourceRoot:        "",
		Sourcefile:        file,
		MinifyIdentifiers: e.minify,
		MinifySyntax:      e.minify,
		MinifyWhitespace:  e.minify,
	})

	if len(result.Errors) != 0 {
		e := result.Errors[0]
		err = fmt.Errorf("%v: (%v:%v) \n%v\n%v^ %v\n", filePath, e.Location.Line, e.Location.Column, e.Location.LineText, strings.Repeat(" ", e.Location.Column), e.Text)
		return
	}
	return result.Code, nil
}

// NewBabelTransformer deprecated
func NewBabelTransformer() *BabelTransformer {
	return &BabelTransformer{
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
		c: make(chan struct{}, 20),
	}
}

// BabelTransformer 负责将高级语法（包括 jsx，ts）转为 goja 能运行的 ES5.1
type BabelTransformer struct {
	p     sync.Pool
	c     chan struct{}
	cache SourceCache
}

func (t *BabelTransformer) Transform(filePath string, code []byte) ([]byte, error) {
	// 并行
	t.c <- struct{}{}
	defer func() { <-t.c }()

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
		return nil, err
	}
	bs := []byte(v.String())
	return bs, nil
}
