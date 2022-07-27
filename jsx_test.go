package ticktick

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
	j, err := NewJsx(WithFS(srcfs))
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
	j, err := NewJsx()
	if err != nil {
		t.Fatal(err)
	}

	http.ListenAndServe(":8081", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		j.reload()
		ti := time.Now()
		s, err := j.Render("./test/Index",
			map[string]interface{}{"a": 1})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("t: %v", time.Now().Sub(ti))

		writer.Write([]byte(s))
	}))
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

			_, err := j.Render("./test/App", map[string]interface{}{"a": 1})
			if err != nil {
				t.Fatal(err)
			}
		}()
	}

	wg.Wait()
}
