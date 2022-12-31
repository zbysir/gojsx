package gojsx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/zbysir/gojsx/pkg/mdx"
	"go.abhg.dev/goldmark/toc"
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

type tableOfContentItem struct {
	Items []tableOfContentItem `json:"items"`
	Title string               `json:"title"`
	Id    string               `json:"id"`
}
type tableOfContent struct {
	Items []tableOfContentItem `json:"items"`
}

func trToc(t *toc.TOC) *tableOfContent {
	return &tableOfContent{
		Items: trTocItems(t.Items),
	}
}

func trTocItems(t toc.Items) []tableOfContentItem {
	ts := make([]tableOfContentItem, len(t))
	for i, v := range t {
		ts[i] = tableOfContentItem{
			Items: trTocItems(v.Items),
			Title: string(v.Title),
			Id:    string(v.ID),
		}
	}
	return ts
}

// TODO SourceMap
// 如果是 md 格式，则直接当成 raw text 处理，如果是 mdx 格式，则按照 jsx 格式处理
func (e *EsBuildTransform) transformMarkdown(ext string, src []byte) (out []byte, err error) {
	// 将 md 处理成 xhtml
	var mdHtml bytes.Buffer
	ctx := parser.NewContext()
	opts := []goldmark.Option{
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			html.WithXHTML(),
		),
		goldmark.WithExtensions(
			meta.Meta,
			extension.GFM,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // for toc
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

	doc := md.Parser().Parse(text.NewReader(trapBOM(src)))
	tocTree, err := toc.Inspect(doc, src)
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
		"meta": ToStrMap(m),
		"toc":  trToc(tocTree),
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

// ToStrMap gopkg.in/yaml.v2 会解析出 map[interface{}]interface{} 这样的结构，不支持 json 序列化。需要手动转一次
func ToStrMap(i interface{}) interface{} {
	switch t := i.(type) {
	case map[string]interface{}:
		m := map[string]interface{}{}
		for k, v := range t {
			m[k] = ToStrMap(v)
		}
		return m
	case map[interface{}]interface{}:
		m := map[string]interface{}{}
		for k, v := range t {
			m[k.(string)] = ToStrMap(v)
		}
		return m
	case []interface{}:
		m := make([]interface{}, len(t))
		for i, v := range t {
			m[i] = ToStrMap(v)
		}
		return m
	default:
		return i
	}
}

func (e *EsBuildTransform) Transform(filePath string, code []byte, format TransformerFormat) (out []byte, err error) {
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

		//log.Printf("transformMarkdown code: %s", code)
	default:
		var ok bool
		loader, ok = defaultExtensionToLoaderMap[ext]
		if !ok {
			return nil, fmt.Errorf("unsupport file extension(%s) for transform", ext)
		}
	}

	result := api.Transform(string(code), api.TransformOptions{
		Loader:            loader,
		Target:            api.ESNext,
		JSXMode:           api.JSXModeAutomatic,
		Format:            esFormat,
		Platform:          api.PlatformNode,
		Sourcemap:         api.SourceMapInline,
		SourceRoot:        "",
		Sourcefile:        file,
		MinifyIdentifiers: e.minify,
		MinifySyntax:      e.minify,
		MinifyWhitespace:  e.minify,
		GlobalName:        globalName,
	})

	if len(result.Errors) != 0 {
		e := result.Errors[0]
		if e.Location != nil {
			err = fmt.Errorf("%v: (%v:%v) \n%v\n%v^ %v\n", filePath, e.Location.Line, e.Location.Column, e.Location.LineText, strings.Repeat(" ", e.Location.Column), e.Text)
		} else {
			err = fmt.Errorf("%v\n", e.Text)
		}
		return
	}

	code = result.Code
	if globalName != "" {
		// 如果是 IIFE 格式，则始终将结果导出
		code = bytes.TrimPrefix(code, []byte(fmt.Sprintf("var %s = ", globalName)))
	}
	return code, nil
}
