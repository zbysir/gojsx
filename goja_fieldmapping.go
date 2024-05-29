package gojsx

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	"reflect"
	"strings"
)

type tagFieldNameMapper struct {
	tagName         string
	uncapMethods    bool
	fallbackToUncap bool //  If the tag is not found, use lowercase initials.
}

func (tfm tagFieldNameMapper) FieldName(_ reflect.Type, f reflect.StructField) string {
	tag := f.Tag.Get(tfm.tagName)
	if idx := strings.IndexByte(tag, ','); idx != -1 {
		tag = tag[:idx]
	}
	if parser.IsIdentifier(tag) {
		return tag
	}
	if tfm.fallbackToUncap {
		return uncapitalize(f.Name)
	}
	return ""
}

func uncapitalize(s string) string {
	return strings.ToLower(s[0:1]) + s[1:]
}

func (tfm tagFieldNameMapper) MethodName(_ reflect.Type, m reflect.Method) string {
	if tfm.uncapMethods {
		return uncapitalize(m.Name)
	}
	return m.Name
}

func TagFieldNameMapper(tagName string, uncapMethods bool, fallbackToUncap bool) goja.FieldNameMapper {
	return tagFieldNameMapper{tagName, uncapMethods, fallbackToUncap}
}
