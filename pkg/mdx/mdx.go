package mdx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"regexp"
)

type MdFormat string

const (
	Mdx MdFormat = "mdx"
	Md  MdFormat = "md"
)

type mdJsx struct {
	format MdFormat
}

func NewMdJsx(format MdFormat) goldmark.Extender {
	return &mdJsx{format: format}
}

func (e *mdJsx) Extend(m goldmark.Markdown) {
	switch e.format {
	case Mdx:
		m.Parser().AddOptions(
			parser.WithBlockParsers(
				util.Prioritized(NewJsCodeParser(), 0),
				util.Prioritized(NewJsxParser(), 10),
			),
		)
	}

	// 如果是 md 格式，则需要将 jsx 语法字符（如 {}）解析为 html 编码
	writer := &jsxWriter{encode: e.format == Md}
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&JsxRender{w: writer}, 10),
	))

	m.Renderer().AddOptions(
		html.WithUnsafe(),
		html.WithXHTML(),
		html.WithWriter(writer),
	)
}

type jsxWriter struct {
	encode bool
}

// Processing jsx syntax strings
// {: &#123;
// }: &#125;
// <: &lt;
// >: &gt;
func (j *jsxWriter) encodeJsxInsecure(s []byte) []byte {
	s = bytes.ReplaceAll(s, []byte("{"), []byte("&#123;"))
	s = bytes.ReplaceAll(s, []byte("}"), []byte("&#125;"))
	s = bytes.ReplaceAll(s, []byte("<"), []byte("&lt;"))
	s = bytes.ReplaceAll(s, []byte(">"), []byte("&gt;"))
	return s
}

func (j *jsxWriter) encodeJsxTag(s []byte) []byte {
	s = jsxTagStartOrEndReg.ReplaceAllFunc(s, func(i []byte) []byte {
		return bytes.ToLower(i)
	})
	return s
}

func (j *jsxWriter) Write(writer util.BufWriter, source []byte) {
	j.SecureWrite(writer, source)
}

func (j *jsxWriter) RawWrite(writer util.BufWriter, source []byte) {
	if j.encode {
		writer.Write(j.encodeJsxTag(source))
	} else {
		writer.Write(source)
	}
}

// SecureWrite 用于写入存文本
func (j *jsxWriter) SecureWrite(writer util.BufWriter, source []byte) {
	if j.encode {
		writer.Write(j.encodeJsxInsecure(source))
	} else {
		writer.Write(source)
	}
}

type JsxRender struct {
	w html.Writer
}

func (r *JsxRender) writeLines(w util.BufWriter, source []byte, n ast.Node) {
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		line := n.Lines().At(i)
		w.Write(line.Value(source))
	}
}

func (j *JsxRender) writeHtmlAttr(w util.BufWriter, source string) {
	w.WriteString(" dangerouslySetInnerHTML={{ __html: ")
	json.NewEncoder(w).Encode(source)
	w.WriteString("}}")
}

func (j *JsxRender) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.FencedCodeBlock)
	if entering {
		_, _ = w.WriteString("<pre><code")
		language := n.Language(source)
		if language != nil {
			_, _ = w.WriteString(" class=\"language-")
			j.w.Write(w, language)
			_, _ = w.WriteString("\"")
		}

		var body bytes.Buffer
		l := n.Lines().Len()
		for i := 0; i < l; i++ {
			line := n.Lines().At(i)
			body.Write(line.Value(source))
		}
		if body.Len() > 0 {
			j.writeHtmlAttr(w, body.String())
		}

		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString("</code></pre>\n")
	}
	return ast.WalkContinue, nil
}

func (r *JsxRender) renderCodeBlock(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<pre><code")

		var body bytes.Buffer
		l := n.Lines().Len()
		for i := 0; i < l; i++ {
			line := n.Lines().At(i)
			body.Write(line.Value(source))
		}

		r.writeHtmlAttr(w, body.String())
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString("</code></pre>\n")
	}
	return ast.WalkContinue, nil
}

func (j *JsxRender) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.HTMLBlock)

	if entering {
		// script|pre|style|textarea
		if n.HTMLBlockType == ast.HTMLBlockType1 {
			var body bytes.Buffer
			l := n.Lines().Len()
			for i := 0; i < l; i++ {
				line := n.Lines().At(i)
				body.Write(line.Value(source))
			}

			// tag
			bodys := body.Bytes()
			tagStartIndex := bytes.Index(bodys, []byte(">"))
			tagEndIndex := bytes.LastIndex(bodys, []byte("</"))

			var tagBody []byte
			var tagEnd []byte
			if tagEndIndex != -1 {
				tagBody = bodys[tagStartIndex+1 : tagEndIndex]
				tagEnd = bodys[tagEndIndex:]
			} else {
				tagBody = bodys[tagStartIndex+1:]
			}

			tagStart := bodys[:tagStartIndex]
			w.Write(tagStart)
			j.writeHtmlAttr(w, string(tagBody))
			w.Write(tagEnd)
		} else {
			l := n.Lines().Len()
			for i := 0; i < l; i++ {
				line := n.Lines().At(i)
				j.w.SecureWrite(w, line.Value(source))
			}
		}
	} else {
		if n.HasClosure() {
			closure := n.ClosureLine
			j.w.SecureWrite(w, closure.Value(source))
		}
	}
	return ast.WalkContinue, nil
}

func (r *JsxRender) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	n := node.(*ast.RawHTML)
	l := n.Segments.Len()
	for i := 0; i < l; i++ {
		segment := n.Segments.At(i)

		//w.Write(segment.Value(s))
		r.w.RawWrite(w, segment.Value(source))
	}
	return ast.WalkSkipChildren, nil
}

func (j *JsxRender) renderJsxBlock(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		j.w.RawWrite(w, line.Value(src))
	}

	return ast.WalkContinue, nil
}

func (j *JsxRender) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, j.renderFencedCodeBlock)
	reg.Register(ast.KindHTMLBlock, j.renderHTMLBlock)
	reg.Register(ast.KindRawHTML, j.renderRawHTML)
	reg.Register(ast.KindCodeBlock, j.renderCodeBlock)
	//reg.Register(ast.KindParagraph, j.renderCodeBlock)
	reg.Register(jsxKind, j.renderJsxBlock)
}

var jsxKind = ast.NewNodeKind("Jsx")

type JsxNode struct {
	ast.BaseBlock
	pc  parser.Context
	tag string
}

func (j *JsxNode) Kind() ast.NodeKind {
	return jsxKind
}

func (j *JsxNode) GetContent() parser.Context {
	return j.pc
}

// IsRaw return true 不解析 block 中的内容
func (j *JsxNode) IsRaw() bool {
	return true
}

func (j *JsxNode) Dump(source []byte, level int) {
	ast.DumpHelper(j, source, level, nil, nil)
}

func (j *JsxNode) HasBlankPreviousLines() bool {
	return true
}

func (j *JsxNode) SetBlankPreviousLines(v bool) {
	return
}

type jsxParser struct {
}

func NewJsxParser() parser.BlockParser {
	return &jsxParser{}
}

// InlineParser 暂时不实现
//func (j *jsxParser) Parse(parent ast.Node, reader text.Reader, pc parser.Context) ast.Node {
//	panic("implement me")
//}

var _ parser.BlockParser = (*jsxParser)(nil)

//var _ parser.InlineParser = (*jsxParser)(nil) // InlineParser 暂时不实现

func (j *jsxParser) Trigger() []byte {
	return []byte{'<'}
}

// 匹配 <A> or <>
var jsxTagStartReg = regexp.MustCompile(`^ {0,3}<(([A-Z]+[a-zA-Z0-9\-]*)|>)`)
var jsxTagStartOrEndReg = regexp.MustCompile(`^ {0,3}</?(([A-Z]+[a-zA-Z0-9\-]*)|>)`)

func (j *jsxParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	node := &JsxNode{
		BaseBlock: ast.BaseBlock{},
		pc:        pc,
	}
	line, segment := reader.PeekLine()
	if pos := pc.BlockOffset(); pos < 0 || line[pos] != '<' {
		return nil, parser.NoChildren
	}
	match := jsxTagStartReg.FindAllSubmatch(line, -1)
	if match == nil {
		return nil, parser.NoChildren
	}

	tagName := string(match[0][2])

	_, s := reader.Position()
	offset := s.Start
	bs := reader.Source()[offset:]

	buf := bytes.NewBufferString(string(bs))
	start, end, ok, err := parseTagToClose(buf)
	if err != nil {
		return nil, parser.NoChildren
	}
	if !ok {
		return nil, parser.NoChildren
	}

	node.tag = tagName
	code := GetJsCode(pc)

	if tagName != "" {
		// 简单判断 变量是否存在于 code，如果存在则说明是 JsxElement
		tr := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, tagName))
		if !tr.MatchString(code) {
			return nil, parser.NoChildren
		}
	}

	segment = text.NewSegment(start+offset, end+offset)
	node.Lines().Append(segment)
	reader.Advance(segment.Len() - 1)
	return node, parser.Close

}

func (j *jsxParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return parser.Close
}

func (j *jsxParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
}

func (j *jsxParser) CanInterruptParagraph() bool {
	return true
}

func (j *jsxParser) CanAcceptIndentedLine() bool {
	return true
}
