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

	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&JsxRender{}, 10),
	))
}

type JsxRender struct {
	htmlrender html.Renderer
}

func (r *JsxRender) writeLines(w util.BufWriter, source []byte, n ast.Node) {
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		line := n.Lines().At(i)
		w.Write(line.Value(source))
	}
}

func (j *JsxRender) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.FencedCodeBlock)
	if entering {
		_, _ = w.WriteString("<pre><code")
		language := n.Language(source)
		if language != nil {
			_, _ = w.WriteString(" class=\"language-")
			_, _ = w.WriteString("\"")
		}
		w.WriteString(" dangerouslySetInnerHTML={{ __html: ")

		var body bytes.Buffer
		l := n.Lines().Len()
		for i := 0; i < l; i++ {
			line := n.Lines().At(i)
			body.Write(line.Value(source))
		}
		json.NewEncoder(w).Encode(body.String())

		w.WriteString("}}")
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString("</code></pre>\n")
	}
	return ast.WalkContinue, nil
}

func (r *JsxRender) renderCodeBlock(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<pre><code")
		// den
		w.WriteString(" dangerouslySetInnerHTML={{ __html: ")

		var body bytes.Buffer
		l := n.Lines().Len()
		for i := 0; i < l; i++ {
			line := n.Lines().At(i)
			body.Write(line.Value(source))
		}
		json.NewEncoder(w).Encode(body.String())

		w.WriteString("}}")
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

			w.WriteString(" dangerouslySetInnerHTML={{ __html: ")

			json.NewEncoder(w).Encode(string(tagBody))

			w.WriteString("}}>")
			w.Write(tagEnd)

		} else {
			l := n.Lines().Len()
			for i := 0; i < l; i++ {
				line := n.Lines().At(i)
				w.Write(line.Value(source))
			}
		}
	} else {
		if n.HasClosure() {
			closure := n.ClosureLine
			w.Write(closure.Value(source))
		}
	}
	return ast.WalkContinue, nil
}

func (j *JsxRender) renderJsxBlock(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		w.Write(line.Value(src))
	}

	return ast.WalkContinue, nil

}

func (j *JsxRender) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, j.renderFencedCodeBlock)
	reg.Register(ast.KindHTMLBlock, j.renderHTMLBlock)
	reg.Register(ast.KindCodeBlock, j.renderCodeBlock)
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
var htmlTagStartReg = regexp.MustCompile(`^ {0,3}<(([A-Z]+[a-zA-Z0-9\-]*)|>)`)

func (j *jsxParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	node := &JsxNode{
		BaseBlock: ast.BaseBlock{},
		pc:        pc,
	}
	line, segment := reader.PeekLine()
	if pos := pc.BlockOffset(); pos < 0 || line[pos] != '<' {
		return nil, parser.NoChildren
	}
	match := htmlTagStartReg.FindAllSubmatch(line, -1)
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
