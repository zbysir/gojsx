package gojsx

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEx(t *testing.T) {
	cases := []struct {
		In  string
		Out string
	}{
		{
			In: `GoError: Invalid module: 'test/App2'
  at github.com/zbysir/gojsx/internal/pkg/goja_nodejs/require.(*RequireModule).require-fm (native)
  at test/Index.jsx:1:16(55)
  at github.com/zbysir/gojsx/internal/pkg/goja_nodejs/require.(*RequireModule).require-fm (native)
  at root.js:1:1(1)`,
			Out: `GoError: Invalid module: 'test/App2'
	at test/Index.jsx:1:16
	at root.js:1:1`,
		},
		{
			In: `ReferenceError: i is not defined
  at Index (test/Index.jsx:14:23(64))
  at root.js:1:32(5)`,
			Out: `ReferenceError: i is not defined
	at Index (test/Index.jsx:14:23)
	at root.js:1:32`,
		},
		{
			In: `GoError: load file (test/Index.jsx) error :test/Index.jsx: (4:24) 
        export default function 1 Index(props) {
                                ^ Expected "(" but found "1"
         at github.com/zbysir/gojsx/internal/pkg/goja_nodejs/require.(*RequireModule).require-fm (native)`,
			Out: `GoError: load file (test/Index.jsx) error :test/Index.jsx: (4:24) 
        export default function 1 Index(props) {
                                ^ Expected "(" but found "1"`,
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.Out, parseException(c.In).Error())
	}
}
