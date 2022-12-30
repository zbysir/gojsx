package mdx

import (
	"bytes"
	"fmt"
	"github.com/tdewolff/parse/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"github.com/zbysir/gojsx/internal/pkg/htmlparser"
	"io"
	"regexp"
)

type mdJsx struct {
	jsxRender renderer.NodeRendererFunc
}

// NewMdJsx 如果 jsxRender 为空，则会原样返回 jsx element 代码
func NewMdJsx(jsxRender renderer.NodeRendererFunc) *mdJsx {
	return &mdJsx{jsxRender: jsxRender}
}

func (e *mdJsx) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(NewJsCodeParser(), 0),
			util.Prioritized(NewJsxParser(), 10),
		),
		//parser.WithInlineParsers(util.Prioritized(&jsxParser{}, 0)),
	)
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(JsxNodeRender(e.jsxRender), 100),
	))
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
	start, end, ok, err := ParseToClose(buf)
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
	//node.RemoveChildren(node)
}

func (j *jsxParser) CanInterruptParagraph() bool {
	return true
}

func (j *jsxParser) CanAcceptIndentedLine() bool {
	return true
}

type JsxNodeRender renderer.NodeRendererFunc

func (j JsxNodeRender) Render(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if j == nil {
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
	return j(w, src, node, entering)
}

func (j JsxNodeRender) RegisterFuncs(registerer renderer.NodeRendererFuncRegisterer) {
	registerer.Register(jsxKind, j.Render)
}

func ParseToClose(buf *bytes.Buffer) (start, end int, ok bool, err error) {
	input := parse.NewInput(buf)

	l := htmlparser.NewLexer(input)

	nesting := 0
	var currTag []byte
	var matchTag []byte
	pos := 0

	for end == 0 {
		err := l.Err()
		if err != nil {
			if err == io.EOF {
				break
			}

			return 0, 0, false, err
		}

		tp, bs := l.Next()

		//log.Printf("%s %s", tp, bs)

		begin := pos
		pos += len(bs)
		switch tp {
		case htmlparser.StartTagToken:
			currTag = bs[1:]
			if matchTag == nil {
				matchTag = bs[1:]
				nesting += 1
				start = begin
			} else if bytes.Equal(matchTag, bs[1:]) {
				nesting += 1
			}
		case htmlparser.StartTagVoidToken:
			if bytes.Equal(matchTag, currTag) {
				nesting -= 1
				if nesting == 0 {
					end = pos
					break
				}
			}
		case htmlparser.EndTagToken:
			if bytes.Equal(matchTag, bs[2:len(bs)-1]) {
				nesting -= 1
				if nesting == 0 {
					end = pos
					break
				}
			}
		}
	}
	if end != 0 {
		return start, end, true, nil
	}

	return
}
