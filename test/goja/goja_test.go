package goja

import (
	"github.com/dop251/goja"
	"github.com/zbysir/gojsx/internal/pkg/goja_nodejs/require"
	"testing"
)

func TestRun(t *testing.T) {
	v := goja.New()
	require.NewRegistry().Enable(v)
	va, err := v.RunString("module.exports = { name: \"gojsx\" };")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", va)
}
