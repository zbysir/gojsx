fork from https://github.com/dop251/goja_nodejs

## Change
- 优先使用主动注册的 module
- 编译缓存使用 md5(body) 作为缓存 key
- 优化 InvalidModuleError 报错信息
