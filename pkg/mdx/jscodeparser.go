package mdx

import (
	"bytes"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"regexp"
)

type jsCodeParser struct {
}

// NewJsCodeParser 解析在最开始的 js 代码，支持 import、function、const、let 开头的代码，已空行结束
func NewJsCodeParser() parser.BlockParser {
	return &jsCodeParser{}
}

func (b *jsCodeParser) Trigger() []byte {
	return nil
}

type jsCodeNode struct {
	ast.BaseBlock
}

var jsCodeKind = ast.NewNodeKind("JsCode")

func (j *jsCodeNode) Kind() ast.NodeKind {
	return jsCodeKind
}

// IsRaw return true 不解析 block 中的内容
func (j *jsCodeNode) IsRaw() bool {
	return true
}

func (j *jsCodeNode) Dump(source []byte, level int) {
	ast.DumpHelper(j, source, level, nil, nil)
}

func (b *jsCodeParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	// 只支持放在头部的代码
	if parent.Type() != ast.TypeDocument || parent.HasChildren() {
		return nil, parser.NoChildren
	}

	line, segment := reader.PeekLine()
	segment = segment.TrimLeftSpace(reader.Source())
	if segment.IsEmpty() {
		return nil, parser.NoChildren
	}

	if !jsCodeRegexp.Match(line) {
		return nil, parser.NoChildren
	}

	node := &jsCodeNode{}
	node.Lines().Append(segment)
	reader.Advance(segment.Len() - 1)
	return node, parser.NoChildren
}

var jsCodeRegexp = regexp.MustCompile("^(import )|(const )|(let )|(export )|(function )")

func (b *jsCodeParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	if util.IsBlank(line) {
		return parser.Close | parser.NoChildren
	}

	node.Lines().Append(segment)
	reader.Advance(segment.Len() - 1)
	return parser.Continue | parser.NoChildren
}

var jsCodeKey = parser.NewContextKey()

func (b *jsCodeParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	lines := node.Lines()
	before := GetJsCode(pc)

	var buf bytes.Buffer
	if before != "" {
		buf.WriteString(before)
		buf.WriteString(";\n")
	}

	for i := 0; i < lines.Len(); i++ {
		segment := lines.At(i)
		buf.Write(segment.Value(reader.Source()))
	}

	pc.Set(jsCodeKey, buf.String())

	// remove self
	node.Parent().RemoveChild(node.Parent(), node)
}

func (b *jsCodeParser) CanInterruptParagraph() bool {
	return false
}

func (b *jsCodeParser) CanAcceptIndentedLine() bool {
	return false
}

func GetJsCode(pc parser.Context) string {
	v := pc.Get(jsCodeKey)
	if v == nil {
		return ""
	}
	d := v.(string)
	return d
}
