package gojsx

import (
	"embed"
	_ "embed"
	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

//go:embed test
var srcfs embed.FS

func TestJsx(t *testing.T) {
	j, err := NewJsx(Option{
		SourceCache: nil,
		//SourceFs:    srcfs,
		Debug: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	j.RegisterModule("react", map[string]interface{}{
		"useEffect": func() {},
	})

	s, err := j.Render("./test/Index", map[string]interface{}{"li": []int64{1, 2, 3, 4}, "html": `<h1>dangerouslySetInnerHTML</h1>`},
		WithFs(srcfs),
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", s)
}

//go:embed test/blog/tailwind.css
var tailwind []byte

func TestHttp(t *testing.T) {
	j, err := NewJsx(Option{
		SourceCache: NewMemSourceCache(),
		Debug:       true,
		VmMaxTotal:  10,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = http.ListenAndServe(":8082", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		//j.RefreshRegistry()

		pageData := map[string]interface{}{}
		page := ""
		switch request.URL.Path {
		case "/tailwind.css":
			writer.Header().Set("Content-Type", "text/css")
			writer.Write(tailwind)
			return
		case "/", "":
			page = "home"
			pageData = map[string]interface{}{
				"blogs": []interface{}{
					map[string]interface{}{
						"name": "如何渲染 jsx",
					},
					map[string]interface{}{
						"name": "关于我",
					},
				},
			}
		case "/detail":
			page = "blog-detail"
			pageData = map[string]interface{}{
				"title": "如何渲染 jsx",
				"html":  "html",
			}
		default:
			page = request.URL.Path
		}
		ti := time.Now()
		s, err := j.Render("./test/blog/Index",
			map[string]interface{}{
				"a":        1,
				"title":    "bysir' blog",
				"me":       "bysir",
				"page":     page,
				"pageData": pageData,
				"time":     "",
			})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("t: %v", time.Now().Sub(ti))
		s += time.Now().Sub(ti).String()
		writer.Write([]byte(s))
	}))
	if err != nil {
		t.Fatal(err)
	}
}

// cpu: Intel(R) Core(TM) i5-8279U CPU @ 2.40GHz
// 71415 ns/op
func BenchmarkJsx(b *testing.B) {
	j, err := NewJsx(Option{})
	if err != nil {
		b.Fatal(err)
	}

	// render first to enable cache
	_, err = j.Render("./test/Index", map[string]interface{}{"a": 1})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := j.Render("./test/Index", map[string]interface{}{"a": 1}, WithCache(true))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestP(t *testing.T) {
	j, err := NewJsx(Option{})
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

func TestArrowFunc(t *testing.T) {
	v := goja.New()
	x, err := v.RunString("({a: ()=> 1, b: function(){}})")
	if err != nil {
		t.Fatal(err)
	}

	o := x.(*goja.Object)

	// https://github.com/dop251/goja/pull/419
	t.Logf("b: %T", o.Get("b").Export()) // func(goja.FunctionCall) goja.Value
	t.Logf("a: %T", o.Get("a").Export()) // got 'map', but expect 'func(goja.FunctionCall) goja.Value'
}

func TestCleanClass(t *testing.T) {
	v := VDom{}
	var s strings.Builder
	v.renderClassName(&s, "a1 a12\n b1  \n\n\n c1\nd1 d12", true)

	assert.Equal(t, "a1 a12 b1 c1 d1 d12", s.String())
}

func TestCamelString(t *testing.T) {
	v := VDom{}
	var s strings.Builder

	v.renderAttributes(&s, map[string]interface{}{"strokeWidth": 1})

	t.Logf("%+v", s.String())
	assert.Equal(t, " stroke-width=\"1\"", s.String())
}

func TestRunJs(t *testing.T) {
	j, err := NewJsx(Option{})
	if err != nil {
		t.Fatal(err)
	}
	v, err := j.RunJs([]byte(`function HelloJSX(){return <p></p>}; export default <HelloJSX></HelloJSX>`), WithFileName("1.tsx"), WithTransform(TransformerFormatIIFE))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", v.Export())
}

func TestRunMd(t *testing.T) {
	j, err := NewJsx(Option{})
	if err != nil {
		t.Fatal(err)
	}
	v, err := j.RunJs([]byte(`## h2`), WithFileName("1.md"), WithTransform(TransformerFormatIIFE))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", v.Export())
}

func TestRenderMd(t *testing.T) {
	j, err := NewJsx(Option{})
	if err != nil {
		t.Fatal(err)
	}
	n, err := j.Render("./test/md.md", map[string]interface{}{"a": 1})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%v", n)
}

func TestRenderMdx(t *testing.T) {
	j, err := NewJsx(Option{})
	if err != nil {
		t.Fatal(err)
	}
	n, err := j.Render("./test/mdx.mdx", map[string]interface{}{"a": 1})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%v", n)
}

func TestExec(t *testing.T) {
	j, err := NewJsx(Option{})
	if err != nil {
		t.Fatal(err)
	}
	n, err := j.Exec("./test/md.md", map[string]interface{}{"a": 1})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", n.Def.ToHtml())
}
