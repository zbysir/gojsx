package gojsx

import (
	"strings"
	"unicode"
)

// ToKebabCase converts a CamelCase string to a hyphen-separated lowercase string.
// 用于实现
// /node_modules/react-dom/cjs/react-dom-server.node.development.js hyphenateStyleName
// github.com/gobeam/stringy 和 github.com/stoewer/go-strcase 在处理 “--color“ 都有问题。
func ToKebabCase(s string) string {
	var result strings.Builder

	for i, r := range s {
		// 检查是否是大写字母
		if unicode.IsUpper(r) || unicode.IsDigit(r) {
			// 如果不是第一个字符，则在前面添加连字符
			if i > 0 {
				// 检查前一个字符是否是小写字母或数字
				if unicode.IsLower(rune(s[i-1])) || !unicode.IsDigit(rune(s[i-1])) {
					result.WriteRune('-')
				} else if i > 1 && unicode.IsUpper(rune(s[i-1])) {
					// 处理如 "XMLHttpRequest" 这样的情况
					if unicode.IsLower(rune(s[i-2])) {
						result.WriteRune('-')
					}
				}
			}
			// 将当前字符转换为小写并写入结果
			result.WriteRune(unicode.ToLower(r))
		} else {
			// 直接写入小写字符
			result.WriteRune(r)
		}
	}
	return result.String()
}
