package jsx

import (
	"embed"
	_ "embed"
	"net/http"
	"sync"
	"testing"
	"time"
)

//go:embed test
var srcfs embed.FS

func TestJs(t *testing.T) {
	j, err := NewJsx(WithFS(srcfs), WithSourceCache(NewFileCache("./.cache")))
	//j, err := NewJsx()
	if err != nil {
		t.Fatal(err)
	}

	s, err := j.Render("./test/Index", map[string]interface{}{"li": []int64{1, 2, 3, 4}})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", s)
}

func TestHttp(t *testing.T) {
	j, err := NewJsx(WithSourceCache(NewFileCache("./.cache")))
	if err != nil {
		t.Fatal(err)
	}

	err = http.ListenAndServe(":8082", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		j.reload()
		ti := time.Now()
		s, err := j.Render("./test/Index",
			map[string]interface{}{
				"a":     1,
				"title": "Jsx",
			})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("t: %v", time.Now().Sub(ti))

		writer.Write([]byte(s))
	}))
	if err != nil {
		t.Fatal(err)
	}
}

// cpu: Intel(R) Core(TM) i5-8279U CPU @ 2.40GHz
// 116,239 ns/op
func BenchmarkJsx(b *testing.B) {
	j, err := NewJsx()
	if err != nil {
		b.Fatal(err)
	}

	// render first to enable cache
	_, err = j.Render("./test/Index", map[string]interface{}{"a": 1})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := j.Render("./test/Index", map[string]interface{}{"a": 1})
		if err != nil {
			b.Fatal(err)
		}
	}
}
func TestP(t *testing.T) {
	j, err := NewJsx()
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := j.Render("./test/Index", map[string]interface{}{"a": 1})
			if err != nil {
				t.Fatal(err)
			}
		}()
	}

	wg.Wait()
}
