package gojsx

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// ToKebabCase 用于实现 hyphenateStyleName
// /node_modules/react-dom/cjs/react-dom-server.node.development.js hyphenateStyleName
func TestToKebabCase(t *testing.T) {
	cases := map[string]string{
		"fontWidth": "font-width",
		"FontWidth": "font-width",
		"color":     "color",
		"Color":     "color",
		"--color":   "--color",
	}

	for in, out := range cases {
		assert.Equal(t, out, ToKebabCase(in))
	}
}
