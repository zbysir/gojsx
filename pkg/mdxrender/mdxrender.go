package mdxrender

import (
	"bytes"
	"fmt"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	jsx "github.com/zbysir/gojsx"
	"github.com/zbysir/gojsx/pkg/mdx"
	"io/fs"
)

func NewJsxRender(x *jsx.Jsx, fs fs.FS) renderer.NodeRendererFunc {
	return func(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		lines := node.Lines()
		var b bytes.Buffer
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			b.Write(line.Value(src))
		}
		jsxCode := b.String()

		ctx := node.(interface{ GetContent() parser.Context }).GetContent()
		jsCode := mdx.GetJsCode(ctx)

		code := fmt.Sprintf("%s; module.exports = %s", jsCode, jsxCode)

		//log.Infof("NewJsxRender code: %s", code)

		v, err := x.RunJs([]byte(code), jsx.WithTransform(jsx.TransformerFormatIIFE), jsx.WithFileName("root.tsx"), jsx.WithFs(fs))
		if err != nil {
			return ast.WalkStop, fmt.Errorf("render jsx error: %v", err)
		}

		vd := jsx.VDom(v.Export().(map[string]interface{}))
		w.Write([]byte(vd.Render()))

		return ast.WalkContinue, nil
	}
}
