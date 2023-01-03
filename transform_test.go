package gojsx

import (
	"testing"
)

func TestTransform(t *testing.T) {
	x := NewEsBuildTransform(EsBuildTransformOptions{})
	x.debug = true

	t.Run("json", func(t *testing.T) {
		b, err := x.Transform("1.json", []byte(`{"a":1}`), TransformerFormatCommonJS)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s", b)

	})

	t.Run("css", func(t *testing.T) {
		b, err := x.Transform("1.css", []byte(`.a{color: red}`), TransformerFormatCommonJS)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s", b)
	})

	t.Run("tsx", func(t *testing.T) {
		b, err := x.Transform("1.tsx", []byte(`import HelloJSX from './index.tsx'; module.exports = <HelloJSX></HelloJSX>`), TransformerFormatIIFE)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s", b)
	})

	t.Run("md", func(t *testing.T) {
		b, err := x.Transform("1.md", []byte(`
---
{a: 1}
---
{ffff: ffdaf}
<>
dfafefdf: fwe :

f{}fsdfsdfas d{}

fsd<><@EOI3u4iuO#$U#($U#94u8u8
<?fdf>

"'""'""
""
"

##￥77&￥&￥&7&&&&4uhefuhwf c$&&$
;;;
<><<<><?>

<Toc items = {toc}></Toc>

`+"有不闭合的标签，如 `<meta charset=\"UTF-8\"> `"+`

`+"我们要渲染的模板是这个样子的\n```vue\n<template>\n  <div>\n    <span class=\"bg-gray\" :class=\"cus_class\" :style=\"{'font-size': fontSize+'px'}\"> {{msg}} </span>\n  </div>\n</template>\n```"+`
## h2`), TransformerFormatIIFE)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s", b)
	})
	t.Run("mdx", func(t *testing.T) {
		b, err := x.Transform("1.mdx", []byte(`
## h2 {1}

<>
{[].map(i=>(8))}
</>
`), TransformerFormatIIFE)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s", b)
	})
	t.Run("mdx2", func(t *testing.T) {
		b, err := x.Transform("1.mdx", []byte(`
---
logo: Hollow
---

import Logo from "./logo"
import Footer from "./footer.md"
const history = [
  {
    time: "2020.01",
    msgs: ["诞生", "hh"],
  }
]

`), TransformerFormatIIFE)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s", b)
	})
}

//func TestHtml(t *testing.T) {
//	url.PathEscape()
//
//}
