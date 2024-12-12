# gojsx

Render Jsx / Tsx / MD / MDX by Golang.

使用 Go 渲染 Jsx、Tsx、MD、MDX。

Features:
- Pure Golang, fast and simple

## Install

```shell
go get github.com/zbysir/gojsx
```

## Example

### TSX
```jsx
import App from "./App";

export default function Index(props) {
  return <html lang="en">
  <head>
    <meta charSet="UTF-8"/>
    <title>Title</title>
    <link href="https://unpkg.com/tailwindcss@^2/dist/tailwind.min.css" rel="stylesheet"/>
  </head>
  <body>
  <App {...props}></App>
  </body>
  </html>
}
```

### Mdx
```mdx
---
title: "Hi"
---

import Footer from "./footer.md"

# {meta.title}

<Footer/>

```

### Render File

Then use `gojsx` to render .tsx or .mdx file.

```go
package jsx

func TestJsx(t *testing.T) {
	j, err := gojsx.NewJsx(gojsx.Option{})
	if err != nil {
		t.Fatal(err)
	}

	s, err := j.Render("./test/Index.jsx", map[string]interface{}{"li": []int64{1, 2, 3, 4}})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", s)
}
```

## Extended syntax
In addition to supporting most of the syntax of jsx, gojsx also supports some special syntax

### Render raw html 

Use `{{__dangerousHTML}}` to render raw html without any tag.

```jsx
export default function Index(props) {
    return <>
      {{__dangerousHTML: props.rawHtml}}
    </>
}
```

## Defects

### How to bind event? e.g. onClick
Since the binding event must happen on the browser, and jsx is js code, the browser needs to run the entire jsx component to bind the event correctly,
which requires the introduction of react at the front end, otherwise it is very complicated to implement,
but the use react will cause jsx to no longer be pure jsx, which in turn will cause `gojsx` to become more complicated.

So `gojsx` can't implement event binding that uses simple react syntax.

To save the day, you can either write your own js to manipulate the dom (as everyone did in the JQuery days), or use a library like AlpineJs.

## Dependents
- [goja](https://github.com/dop251/goja)
- [esbuild](https://github.com/evanw/esbuild)
- [goldmark](github.com/yuin/goldmark)
