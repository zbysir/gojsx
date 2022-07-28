# gojsx

Render React Jsx by Golang

使用 Go 渲染 Jsx。

Jsx 优势：

- 实际上就是 js 代码，它是图灵完备的。
- 和 js 生态行为一致，不用学习更多语法。

## 例子

编写 jsx 文件（或者 tsx）

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

```go
package jsx

func TestJs(t *testing.T) {
	j, err := NewJsx()
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

## 实现原理

由于 Jsx 实际上就是 js 代码，如果要渲染 jsx，则需要在 Golang 中运行 js 代码，感谢伟大的 goja 库。

由于 goja 只支持 es5.1 语法，高级语法如 TS、ES6 则需要通过 babel 转换，babel 提供一个浏览器运行版本，刚好 goja 可以运行它。

将编译之后的 jsx 交给 goja 运行，能得到一个虚拟节点树，然后再由 golang 进行渲染得到 HTML。

## 性能

babel 是十分慢的，相信开发过前端的朋友都深有体会，但我们可以通过预编译来减少影响。除此之外运行编译好的 jsx 模板是很快的（ goja 本身很快），不必担心。

另外 这个项目应该是性能不敏感的，我想用它来生成静态文件（例如制作官网与博客），而不是实时渲染。

## FQA

### 支持 React 的 UI 库吗？ 如 ant

不支持，由于库的复杂依赖关系，会出现意料之外的错误，也会导致加载变得很慢。

如果你非要使用，尝试使用 webpack 将依赖打包成独立的 js 文件，然后引入它（待测试）。
