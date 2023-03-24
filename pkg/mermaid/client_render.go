package mermaid

import (
	"go.abhg.dev/goldmark/mermaid"
	"html/template"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// ClientRenderer renders Mermaid diagrams as HTML,
// to be rendered into images client side.
//
// It operates by installing a <script> tag into the document
// that renders the Mermaid diagrams client-side.
type ClientRenderer struct {
	// URL of Mermaid Javascript to be included in the page.
	//
	// Defaults to the latest version available on cdn.jsdelivr.net.
	MermaidJS string
}

// RegisterFuncs registers the renderer for Mermaid blocks with the provided
// Goldmark Registerer.
func (r *ClientRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(mermaid.Kind, r.Render)
}

// Render renders mermaid.Block nodes.
func (*ClientRenderer) Render(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*mermaid.Block)
	if entering {
		w.WriteString(`<pre class="mermaid">`)
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			template.HTMLEscape(w, line.Value(src))
		}
	} else {
		w.WriteString("</pre>")
	}
	return ast.WalkContinue, nil
}
