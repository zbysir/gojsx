package gojsx

import "testing"

// 优化前
//   noCache 	1,919,804 ns/op
// 优化 编译缓存
//                354,871 ns/op
func BenchmarkName(b *testing.B) {
	j, err := NewJsx(Option{})
	//j.debug = true
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	j.RegisterModule("@bysir/hollow", map[string]interface{}{
		"getConfig": func() interface{} {
			return map[string]interface{}{}
		},
		"getContents": func() interface{} {
			return map[string]interface{}{
				"list": []interface{}{},
			}
		},
	})

	b.Logf("-----begin-----")
	for i := 0; i < b.N; i++ {
		_, err := j.Exec("./test/Index", WithCache(false), WithAutoExecJsx(map[string]interface{}{}))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestOne(t *testing.T) {
	j, err := NewJsx(Option{})
	j.debug = true
	if err != nil {
		t.Fatal(err)
	}

	j.RegisterModule("@bysir/hollow", map[string]interface{}{
		"getConfig": func() interface{} {
			return map[string]interface{}{}
		},
		"getContents": func() interface{} {
			return map[string]interface{}{
				"list": []interface{}{},
			}
		},
	})

	t.Logf("----- begin -----")
	_, err = j.Exec("./test/Index", WithCache(false))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("----- second time -----")
	e, err := j.Exec("./test/Index", WithCache(false), WithAutoExecJsx(map[string]interface{}{}))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", e.Default.(VDom))
}
