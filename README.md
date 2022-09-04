# gojsx

Render React Jsx by Golang

使用 Go 渲染 Jsx。

Features:
- Pure Golang, fast and simple

Jsx Features:

- It's actually javascript code, it's Turing complete, also don't worry about [v-for with v-if](https://cn.vuejs.org/guide/essentials/list.html#v-for-with-v-if)
- Consistent with javascript ecological behavior, no need to learn more syntax

## Example

Write the .jsx file (or .tsx) as follows

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

Then use `gojsx` to render it

```go
package jsx

func TestJsx(t *testing.T) {
	j, err := NewJsx(Option{})
	if err != nil {
		t.Fatal(err)
	}

	s, err := j.Render("./test/Index", map[string]interface{}{"li": []int64{1, 2, 3, 4}})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", s)
}
```

## How it works

由于 Jsx 实际上就是 js 代码，如果要渲染 jsx，则需要在 Golang 中运行 js 代码，感谢伟大的 [goja](https://github.com/dop251/goja) 库。

由于 goja 只支持 es5.1 语法，高级语法如 TS、ES6 则需要通过 babel 转换，babel 提供一个浏览器运行版本，刚好 goja 可以运行它。不过 babel 编译是巨慢的，好在还有 [esbuild](https://github.com/evanw/esbuild) 可以做同样的事。所以 gojsx 使用 esbuild 作为编译器。

将编译之后的 jsx 交给 goja 运行，能得到一个虚拟节点树，然后再由 golang 进行渲染得到 HTML。

## Performance

gojsx 默认使用 [esbuild](https://github.com/evanw/esbuild) 来编译文件，它比 babel 快至少一个数量级，增量编译文件通常只需要几毫秒。

除此之外运行编译好的 js 文件是很快的（ goja 本身很快），不必担心。

另外 这个项目应该是性能不敏感的，我想用它来生成静态文件（例如制作官网与博客），而不是实时渲染。

## FQA

### 支持 React 的 UI 库吗？ 如 ant

不支持