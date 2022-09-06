package jsx

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"testing"
)

// https://github.com/dop251/goja_nodejs/issues/35
func TestRegistryUtil(t *testing.T) {
	vm := goja.New()

	require.NewRegistry().Enable(vm)
	console.Enable(vm)

	_, err := vm.RunScript("root.js", "require('./util')")
	if err == nil {
		t.Fatal("err is not nil")
	}
}
