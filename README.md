# gojsx

Render Jsx / Tsx / MD / MDX by Golang.

使用 Go 渲染 Jsx、Tsx、MD、MDX。

Features:
- Pure Golang, fast and simple

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

## Dependents
- [goja](https://github.com/dop251/goja)
- [esbuild](https://github.com/evanw/esbuild)
- [goldmark](github.com/yuin/goldmark)
