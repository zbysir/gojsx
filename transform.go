package gojsx

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/go-sourcemap/sourcemap"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/zbysir/gojsx/pkg/mdx"
	"go.abhg.dev/goldmark/mermaid"
	"log"
	"path/filepath"
	"strings"
)

type Transformer interface {
	Transform(filePath string, src []byte, format TransformerFormat) (out []byte, err error)
}

type TransformerFormat uint8

const (
	TransformerNone           TransformerFormat = 0
	TransformerFormatDefault  TransformerFormat = 1
	TransformerFormatIIFE     TransformerFormat = 2
	TransformerFormatCommonJS TransformerFormat = 3
	TransformerFormatESModule TransformerFormat = 4
)

type EsBuildTransform struct {
	debug           bool
	minify          bool
	markdownOptions []goldmark.Option
	markdownExport  func(ctx parser.Context, n ast.Node, src []byte) map[string]interface{}
}

type EsBuildTransformOptions struct {
	Minify          bool
	MarkdownOptions []goldmark.Option
	MarkdownExport  func(ctx parser.Context, n ast.Node, src []byte) map[string]interface{}
}

func NewEsBuildTransform(o EsBuildTransformOptions) *EsBuildTransform {
	return &EsBuildTransform{
		minify:          o.Minify,
		markdownOptions: o.MarkdownOptions,
	}
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

func trapBOM(fileBytes []byte) []byte {
	trimmedBytes := bytes.Trim(fileBytes, "\xef\xbb\xbf")
	return trimmedBytes
}

// TODO SourceMap
// 将 md 转换成 jsx 语法
func (e *EsBuildTransform) transformMarkdown(ext string, src []byte) (out []byte, err error) {
	// 将 md 处理成 xhtml
	var mdHtml bytes.Buffer
	ctx := parser.NewContext()
	opts := []goldmark.Option{
		goldmark.WithExtensions(
			meta.Meta,
			extension.GFM,
			&mermaid.Extender{
				RenderMode: mermaid.RenderModeClient,
				MermaidJS:  "https://unpkg.com/mermaid@9/dist/mermaid.min.js",
				NoScript:   false,
				MMDC:       nil,
				Theme:      "",
			},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			// https://github.com/mdx-js/mdx/issues/1279
			parser.WithHeadingAttribute(), // handles special case like ### heading ### {#id}
		),
	}
	switch ext {
	case ".mdx":
		opts = append(opts, goldmark.WithExtensions(
			mdx.NewMdJsx("mdx"),
		))
	case ".md":
		opts = append(opts, goldmark.WithExtensions(
			mdx.NewMdJsx("md"),
		))
	}

	opts = append(opts, e.markdownOptions...)
	md := goldmark.New(opts...)

	doc := md.Parser().Parse(text.NewReader(src), parser.WithContext(ctx))

	if e.debug {
		doc.Dump(src, 1)
	}

	err = md.Renderer().Render(&mdHtml, src, doc)
	if err != nil {
		return
	}

	m := meta.Get(ctx)
	jsCode := mdx.GetJsCode(ctx)

	var code bytes.Buffer
	code.WriteString(jsCode)
	code.WriteString(";\n")

	exportObj := map[string]interface{}{
		"meta": toStrMap(m),
	}

	if e.markdownExport != nil {
		export := e.markdownExport(ctx, doc, src)
		for k, v := range export {
			exportObj[k] = v
		}
	}

	for k, v := range exportObj {
		code.WriteString(fmt.Sprintf("export let %s = ", k))
		bs, _ := json.Marshal(v)
		code.Write(bs)
		code.WriteString(";\n")
	}

	// write jsx
	code.WriteString("export default (props)=> <>")
	mdHtml.WriteTo(&code)
	code.WriteString("</>")

	return code.Bytes(), nil
}

// toStrMap gopkg.in/yaml.v2 会解析出 map[interface{}]interface{} 这样的结构，不支持 json 序列化。需要手动转一次
func toStrMap(i interface{}) interface{} {
	switch t := i.(type) {
	case map[string]interface{}:
		m := map[string]interface{}{}
		for k, v := range t {
			m[k] = toStrMap(v)
		}
		return m
	case map[interface{}]interface{}:
		m := map[string]interface{}{}
		for k, v := range t {
			m[k.(string)] = toStrMap(v)
		}
		return m
	case []interface{}:
		m := make([]interface{}, len(t))
		for i, v := range t {
			m[i] = toStrMap(v)
		}
		return m
	default:
		return i
	}
}

func (e *EsBuildTransform) Transform(filePath string, code []byte, format TransformerFormat) (out []byte, err error) {
	code = trapBOM(code)

	var esFormat api.Format
	var globalName string
	switch format {
	case TransformerNone:
		return code, nil
	case TransformerFormatDefault:
		esFormat = api.FormatDefault
	case TransformerFormatIIFE:
		// 如果是 IIFE 格式，则始终将结果导出
		esFormat = api.FormatIIFE
		globalName = "__export__"
	case TransformerFormatCommonJS:
		esFormat = api.FormatCommonJS
	case TransformerFormatESModule:
		esFormat = api.FormatESModule
	default:
		return code, nil
	}

	_, file := filepath.Split(filePath)
	ext := filepath.Ext(filePath)

	var loader api.Loader
	switch ext {
	case ".md", ".mdx":
		code, err = e.transformMarkdown(ext, code)
		if err != nil {
			return
		}
		loader = api.LoaderTSX
		if e.debug {
			log.Printf("transformMarkdown code: %s", code)
		}

	default:
		var ok bool
		loader, ok = defaultExtensionToLoaderMap[ext]
		if !ok {
			return nil, fmt.Errorf("unsupport file extension(%s) for transform", ext)
		}
	}

	var sourcemapx api.SourceMap
	switch ext {
	case ".jsx", ".tsx", ".mdx", ".md", ".js", ".ts", ".mjs", ".cjs":
		sourcemapx = api.SourceMapInline
	default:
		// .json 不生成 sourcemap，因为会 esbuild 生成空 sourcemap，但 goja 执行空 sourcemap 会报错。
		sourcemapx = api.SourceMapNone
	}

	result := api.Transform(string(code), api.TransformOptions{
		Loader:            loader,
		Target:            api.ESNext,
		JSXMode:           api.JSXModeAutomatic,
		Format:            esFormat,
		Platform:          api.PlatformNode,
		Sourcemap:         sourcemapx,
		SourceRoot:        "",
		Sourcefile:        file,
		MinifyIdentifiers: e.minify,
		MinifySyntax:      e.minify,
		MinifyWhitespace:  e.minify,
		GlobalName:        globalName,
		Footer:            globalName,
	})

	if len(result.Errors) != 0 {
		er := result.Errors[0]
		if er.Location != nil {
			location := e.trySourcemapLocation(er.Location, code)
			err = fmt.Errorf("%v: (%v:%v) \n%v\n%v^ %v\n", filePath, location.Line, location.Column, location.LineText, strings.Repeat(" ", location.Column), er.Text)
		} else {
			err = fmt.Errorf("%v\n", er.Text)
		}
		return
	}

	code = result.Code
	return code, nil
}

// 将 esbuild 报错位置信息通过 sourcemap 转换
func (e *EsBuildTransform) trySourcemapLocation(l *api.Location, source []byte) *api.Location {
	sms := bytes.Split(source, []byte(`sourceMappingURL=data:application/json;base64,`))
	if len(sms) != 2 {
		return l
	}

	sourcemapJson, _ := base64.URLEncoding.DecodeString(string(sms[1]))
	if sourcemapJson == nil {
		return l
	}

	c, err := sourcemap.Parse("./", sourcemapJson)
	if err != nil {
		return l
	}

	file, _, line, column, ok := c.Source(l.Line, l.Column)
	if !ok {
		return l
	}

	return &api.Location{
		File:       file,
		Namespace:  l.Namespace,
		Line:       line,
		Column:     column,
		Length:     l.Length,
		LineText:   sourceLine(c.SourceContent(file), line),
		Suggestion: l.Suggestion,
	}
}

func sourceLine(s string, i int) string {
	return strings.SplitN(s, "\n", i)[i-1]
}
