Base on github.com/tdewolff/parse/v2/html

The purpose is to parse html and jsx

Jsx and html syntax are different, so I made the following modifications:

- `<></>`：现在会按照 tag 处理，原来会处理成 text。
- 兼容 `<A name={name}/>` 语法，能正常解析出 attribute 和 闭合。
