package mdx

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"os"
	"testing"
)

func readFile(name string) string {
	bs, err := os.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return string(bs)
}

func TestMdx(t *testing.T) {
	cases := []struct {
		Name string
		In   string
		Out  string
	}{
		{
			Name: "Base",
			In: `
import Logo from "./logo"
import Footer from "./footer.md"
const a = "3233"

{a}

<>
<div>
  <Logo></Logo>
  <h1 className="text-center">
    Hollow
  </h1>

</div>

  <Footer></Footer>
</>
`,
			Out: `<p>{a}</p>
<>
<div>
  <Logo></Logo>
  <h1 className="text-center">
    Hollow
  </h1>

</div>

  <Footer></Footer>
</>`,
		},
		{
			Name: "FULL",
			In:   readFile("./testdata/fulldemo.md"),
			Out:  readFile("./testdata/fulldemo.md.out.txt"),
		},
	}

	opts := []goldmark.Option{
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			html.WithXHTML(),
		),
		goldmark.WithExtensions(
			meta.Meta,
			extension.GFM,
			NewMdJsx("mdx"),
		),
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {

			var buf bytes.Buffer
			md := goldmark.New(opts...)
			err := md.Convert([]byte(c.In), &buf)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, c.Out, buf.String())
		})
	}
}
