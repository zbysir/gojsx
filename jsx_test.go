package gojsx

import (
	"embed"
	_ "embed"
	"github.com/stretchr/testify/assert"
	"net/http"
	"sync"
	"testing"
	"time"
)

//go:embed test
var srcfs embed.FS

func TestJsx(t *testing.T) {
	j, err := NewJsx(Option{
		SourceCache: nil,
		Fs:          srcfs,
		Debug:       false,
	})
	if err != nil {
		t.Fatal(err)
	}

	j.RegisterModule("react", map[string]interface{}{
		"useEffect": func() {},
	})

	s, err := j.Render("./test/Index", map[string]interface{}{"li": []int64{1, 2, 3, 4}, "html": `<h1>dangerouslySetInnerHTML</h1>`})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "<!DOCTYPE html><html datatype lang=\"zh\"><head><meta charSet=\"UTF-8\"/><title>UnTitled</title><link href=\"https://unpkg.com/tailwindcss@^2/dist/tailwind.min.css\" rel=\"stylesheet\"/></head><body><div a b=\"1\" c=\"1.1\"></div><div class=\"bg-red-50 border-black text-black\">a /2<div b=\"2\" class=\"form\" style=\"font-size: 1px; padding: 2px;\"> f <ul><li> 1 </li><li> 2 </li><li> 3 </li><li> 4 </li></ul> x:2c: c1</div><img alt=\"asdfsf&#34;12312\" data-x=\"{&#34;a&#34;:&#34;`&#39;&#34;}\" src=\"a.jpb\"/><p>&lt;h1&gt;dangerouslySetInnerHTML&lt;/h1&gt;</p><p><h1>dangerouslySetInnerHTML</h1></p></div><button class=\"btn btn-square btn-xs\"><svg class=\"h-6 w-6\" fill=\"none\" stroke=\"currentColor\" viewBox=\"0 0 24 24\" xmlns=\"http://www.w3.org/2000/svg\"><path d=\"M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"2\"></path></svg></button></body></html>", s)
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

func TestRunJs(t *testing.T) {
	j, err := NewJsx(Option{})
	if err != nil {
		t.Fatal(err)
	}
	v, err := j.ExecCode([]byte(`function HelloJSX(){return <p>123</p>}; export default <HelloJSX></HelloJSX>`), WithFileName("1.tsx"))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", v)
}

func TestRunMd(t *testing.T) {
	j, err := NewJsx(Option{})
	if err != nil {
		t.Fatal(err)
	}
	v, err := j.ExecCode([]byte(`## h2`), WithFileName("1.md"))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", v)
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
	n, err := j.Exec("./test/md.md")
	if err != nil {
		t.Fatal(err)
	}

	v, _ := n.Default.(Callable)(nil, nil)
	vd, _ := tryToVDom(v.Export())
	t.Logf("%+v", vd)
	t.Logf("%+v", n.Exports)

	n, err = j.Exec("./test/md.md", WithAutoExecJsx(nil))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", n.Default.(VDom))
}

func TestExecJson(t *testing.T) {
	j, err := NewJsx(Option{})
	if err != nil {
		t.Fatal(err)
	}
	n, err := j.Exec("./test/a.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", n.Exports)
}

type Model struct {
	ID uint
}

func TestEmbeddingStruct(t *testing.T) {
	j, err := NewJsx(Option{})
	if err != nil {
		t.Fatal(err)
	}

	props := struct {
		Model
		Name     string `json:"name"`
		Age      int
		FullName string
	}{
		Model{ID: 233},
		"abc",
		23,
		"bysir",
	}

	v, _, err := j.RenderCode([]byte(`export default (props)=><p>{props.iD +' ' + props.name + ' '+ props.fullName + ' ' + props.age}</p>`), props, WithFileName("1.tsx"))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, `<p>233 abc bysir 23</p>`, v)
}

func TestHydrate(t *testing.T) {
	j, err := NewJsx(Option{})
	if err != nil {
		t.Fatal(err)
	}

	v, ctx, err := j.RenderCode([]byte(`function HelloJSX(props){return <p onClick={()=>alert(props.a)} hydrate-a={JSON.stringify(props.a)}></p>}; export default (props)=><HelloJSX {...props}></HelloJSX>`), map[string]interface{}{
		"a": map[string]interface{}{"name": "1"},
	}, WithFileName("1.tsx"))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, `<p data-hydrate="0" onClick></p>`, v)
	assert.Equal(t, map[string]map[string]string{
		"0": {
			"hydrate-a": `{"name":"1"}`,
		},
	}, ctx.Hydrate)
}
