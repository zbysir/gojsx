package mdx

import (
	"bytes"
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	html2jsx2 "github.com/zbysir/gojsx/pkg/html2jsx"
	"io"
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

var jsxCodeKey = parser.NewContextKey()

func (e *mdJsx) Extend(m goldmark.Markdown) {
	switch e.format {
	case Mdx:
		m.Parser().AddOptions(
			parser.WithBlockParsers(
				util.Prioritized(NewJsCodeParser(), 0),
				util.Prioritized(NewJsxParser(), 10),
			),
			parser.WithInlineParsers(util.Prioritized(NewJsxParser(), 0)),
		)
		m.Renderer().AddOptions(renderer.WithNodeRenderers(
			util.Prioritized(&JsxRender{}, 10),
		))
	}

	m.Renderer().AddOptions(
		html.WithUnsafe(),
		html.WithXHTML(),
	)

	parse := &WrapParser{Parser: m.Parser()}
	m.SetParser(parse)
	m.SetRenderer(WrapRender{Renderer: m.Renderer(), enableJsx: e.format == Mdx, parser: parse})
}

type WrapParser struct {
	parser.Parser
	ctx parser.Context
}

func (p *WrapParser) GetContext() parser.Context {
	return p.ctx
}

func (p *WrapParser) Parse(reader text.Reader, opts ...parser.ParseOption) ast.Node {
	var c parser.ParseConfig
	for _, o := range opts {
		o(&c)
	}
	if c.Context == nil {
		c.Context = parser.NewContext()
		opts = append(opts, parser.WithContext(c.Context))
	}
	p.ctx = c.Context

	return p.Parser.Parse(reader, opts...)
}

type WrapRender struct {
	renderer.Renderer
	enableJsx bool
	parser    *WrapParser
}

func (r WrapRender) Render(w io.Writer, source []byte, n ast.Node) error {
	var buf bytes.Buffer
	err := r.Renderer.Render(&buf, source, n)
	if err != nil {
		return err
	}

	var out bytes.Buffer

	err = html2jsx2.Convert(&buf, &out, r.enableJsx)
	if err != nil {
		return err
	}

	outbs := out.Bytes()

	ctx := r.parser.GetContext()
	if ctx != nil {
		ts := GetJsxCode(ctx)
		if ts != nil {
			for i := 0; i < ts.Len(); i++ {
				s := ts.At(i)
				outbs = bytes.ReplaceAll(outbs, jsxNodePlaceholder(i), s.Value(source))
			}
		}
	}

	w.Write(outbs)
	return nil
}

type JsxRender struct {
}

func (j *JsxRender) renderJsxBlock(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	jsxNode := node.(*JsxNode)
	w.Write(jsxNodePlaceholder(jsxNode.index))
	return ast.WalkContinue, nil
}

func jsxNodePlaceholder(index int) []byte {
	return []byte(fmt.Sprintf("JSXNODE_%d_EDONXSJ", index))
}

func (j *JsxRender) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(jsxKind, j.renderJsxBlock)
}

var jsxKind = ast.NewNodeKind("Jsx")

type JsxNode struct {
	ast.BaseBlock
	tag   string
	index int // 几个
}

func (j *JsxNode) Kind() ast.NodeKind {
	return jsxKind
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
func (j *jsxParser) saveJsxNode(node *JsxNode, segment text.Segment, pc parser.Context) {
	jsxs := GetJsxCode(pc)
	if jsxs == nil {
		jsxs = text.NewSegments()
	}
	node.index = jsxs.Len()
	jsxs.Append(segment)
	pc.Set(jsxCodeKey, jsxs)
}

func (j *jsxParser) Parse(parent ast.Node, reader text.Reader, pc parser.Context) ast.Node {
	line, segment := reader.PeekLine()

	match := jsxTagStartReg.FindAllSubmatch(line, -1)
	if match == nil {
		return nil
	}
	bs := reader.Value(segment)

	start, end, ok, err := parseTagToClose(bytes.NewBuffer(bs))
	if err != nil {
		return nil
	}
	if !ok {
		return nil
	}

	node := &JsxNode{
		BaseBlock: ast.BaseBlock{},
	}

	segment = text.NewSegment(start+segment.Start, end+segment.Start)

	node.Lines().Append(segment)
	reader.Advance(segment.Len())

	j.saveJsxNode(node, segment, pc)
	return node
}

var _ parser.BlockParser = (*jsxParser)(nil)
var _ parser.InlineParser = (*jsxParser)(nil)

func (j *jsxParser) Trigger() []byte {
	return []byte{'<'}
}

// 匹配 <A> or <>
var jsxTagStartReg = regexp.MustCompile(`^ {0,3}<(([A-Z]+[a-zA-Z0-9\-]*)|>)`)

func (j *jsxParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	node := &JsxNode{
		BaseBlock: ast.BaseBlock{},
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

	start, end, ok, err := parseTagToClose(bytes.NewBuffer(bs))
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

	jsxs := GetJsxCode(pc)
	if jsxs == nil {
		jsxs = text.NewSegments()
	}
	node.index = jsxs.Len()
	jsxs.Append(segment)
	pc.Set(jsxCodeKey, jsxs)

	node.Lines().Append(segment)
	reader.Advance(segment.Len() - 1)
	return node, parser.Close

}

func (j *jsxParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return parser.Close
}

func (j *jsxParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	// remove self
	//node.Parent().RemoveChild(node.Parent(), node)
}

func GetJsxCode(pc parser.Context) *text.Segments {
	i := pc.Get(jsxCodeKey)
	if i != nil {
		jsxs := i.(*text.Segments)
		return jsxs
	}

	return nil
}

func (j *jsxParser) CanInterruptParagraph() bool {
	return true
}

func (j *jsxParser) CanAcceptIndentedLine() bool {
	return true
}
