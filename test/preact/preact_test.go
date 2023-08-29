package preact

import (
	"fmt"
	"github.com/zbysir/gojsx"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

//var src embed.FS

func TestPreactSSR(t *testing.T) {

	var src fs.FS

	src = os.DirFS(".")

	file := "./src/index.tsx"
	t.Run("tsx", func(t *testing.T) {
		x := gojsx.NewEsBuildTransform(gojsx.EsBuildTransformOptions{
			Minify: true,
		})

		rootBs, err := fs.ReadFile(src, "src/root.tsx")
		if err != nil {
			t.Fatal(err)
		}

		b, err := x.Transform("./src/root.tsx", rootBs, gojsx.TransformerFormatESModule)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s", b)

		j, err := gojsx.NewJsx(gojsx.Option{
			Fs: src,
		})
		if err != nil {
			t.Fatal(err)
		}

		j.RegisterModule("preact/hooks", map[string]interface{}{
			"useState": func(i any) interface{} {
				return []interface{}{i, func(i any) {}}
			},
		})

		v, err := j.Exec(file, gojsx.WithAutoExecJsx(nil))
		if err != nil {
			t.Fatal(err)
		}

		s := gojsx.Render(v.Default)
		t.Logf("%s", s)
	})

	t.Run("http", func(t *testing.T) {
		// listen 9091
		// http://localhost:9091

		m := http.NewServeMux()
		m.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			j, err := gojsx.NewJsx(gojsx.Option{
				Fs: src,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, fileName := filepath.Split(r.URL.Path)
			if fileName == "" {
				fileName = "index"
			}

			j.RegisterModule("preact/hooks", map[string]interface{}{
				"useState": func(i any) interface{} {
					return []interface{}{i, func(i any) {}}
				},
			})

			file := fmt.Sprintf("./src/page/%s.tsx", fileName)

			var code []byte
			if r.URL.Query().Has("ssr") {
				code = []byte(fmt.Sprintf(`import Layout from "./src/index.tsx"; import Page from "%v" ;
export default function a ({js}){
    return <Layout js={js}> <Page/></Layout>}`, file))
			} else {
				code = []byte(fmt.Sprintf(`import Layout from "./src/index.tsx"; 
export default function a ({js}){
    return <Layout js={js}></Layout>}`))
			}

			log.Printf("code: %s", code)

			start := time.Now()
			v, err := j.ExecCode(code, gojsx.WithAutoExecJsx(map[string]interface{}{"js": fmt.Sprintf(`import {h, hydrate} from "https://cdn.skypack.dev/preact";
	import root from "./js/page/%s.tsx";
    hydrate(h(root), document.body);
`, fileName)}), gojsx.WithFileName("root.tsx"))
			if err != nil {
				t.Fatal(err)
			}

			s := gojsx.Render(v.Default)
			s = strings.ReplaceAll(s, "<body>", fmt.Sprintf("<body>%s", time.Now().Sub(start).String()))
			w.Write([]byte(s))
			return
		}))

		m.Handle("/jslib/react/jsx-runtime", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bs, err := fs.ReadFile(src, "src/jslib/react/jsx-runtime.js")
			if err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/javascript")
			w.Write(bs)
		}))

		m.Handle("/js/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			x := gojsx.NewEsBuildTransform(gojsx.EsBuildTransformOptions{
				Minify: true,
			})

			fileName := strings.TrimPrefix(r.URL.Path, "/js/")

			page := filepath.Join("src", fileName)

			log.Printf("page: %s", fileName)

			rootBs, err := fs.ReadFile(src, page)
			if err != nil {
				w.Write([]byte(err.Error()))
				w.WriteHeader(400)
				return
			}
			pageJs, err := x.Transform(page, rootBs, gojsx.TransformerFormatESModule)
			if err != nil {
				w.Write([]byte(err.Error()))
				w.WriteHeader(400)
				return
			}

			// import{jsx as t,jsxs as i}from"react/jsx-runtime";import{useState as c}from"preact/hooks"

			sJs := string(pageJs)
			//实现在浏览器中import内联JS模块
			// https://juejin.cn/post/7070339012933713956

			// replace used module
			// 这个后期可以做成由用户配置，和 ImportMap 类似
			//sJs = strings.ReplaceAll(sJs, "react/jsx-runtime", "/jslib/react/jsx-runtime")
			sJs = strings.ReplaceAll(sJs, "react/jsx-runtime", "https://cdn.skypack.dev/preact/compat/jsx-runtime")
			//sJs = strings.ReplaceAll(sJs, "react/jsx-runtime", "https://cdn.skypack.dev/react/jsx-runtime")
			sJs = strings.ReplaceAll(sJs, `"preact/hooks"`, `"https://cdn.skypack.dev/preact/hooks"`)
			//sJs = strings.ReplaceAll(sJs, `"preact/hooks"`, `"https://cdn.skypack.dev/react"`)
			sJs = strings.ReplaceAll(sJs, `"preact"`, `"https://cdn.skypack.dev/preact"`)

			w.Header().Set("Content-Type", "application/javascript")

			w.Write([]byte(sJs))
		}))

		http.ListenAndServe(":9091", m)

	})

}
