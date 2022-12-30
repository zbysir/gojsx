# 源

github.com/tdewolff/parse/v2/html

# 修改

- `<></>`：原来会处理成 text，现在会按照 tag 处理
- `<A name={name}/>`：原来不能正确匹配 tag
- `<A name={} b={} >`：原来会解析每个attr，现在只会解析出一个（优化速度与解决位置问题）