package esbuild

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	sourcemap2 "github.com/go-sourcemap/sourcemap"
	"github.com/zbysir/gojsx"
	"github.com/zbysir/gojsx/test/sourcemap"
	"strings"
	"testing"
)

func mockSourcemap() []byte {
	m := sourcemap.Map{
		Version:    3,
		File:       "a.md",
		SourceRoot: "",
		Sources:    []string{"a.md"},
		SourcesContent: []string{`# h1
hihi {a}
`},
		Names:    nil,
		Mappings: "",
	}

	// 0 开始计数, [0, 2] => [0, 6], [1, 0] => [0, 17], [1, 6] => [0, 22]
	// https://www.murzwin.com/base64vlq.html
	// MAAE,WACF,KAAM
	//
	// # h1
	//   ^2
	// hihi {a}
	// ^0    ^6
	// <><h1>h1 </h1><p>hih {a}</p></>
	//       ^6         ^17  ^22
	//

	m.AddMapping(&sourcemap.Mapping{
		GeneratedLine:   0,
		GeneratedColumn: 6,
		OriginalFile:    "a.md",
		OriginalLine:    0,
		OriginalColumn:  2,
		OriginalName:    "",
	})
	m.AddMapping(&sourcemap.Mapping{
		GeneratedLine:   0,
		GeneratedColumn: 17,
		OriginalFile:    "a.md",
		OriginalLine:    1,
		OriginalColumn:  0,
		OriginalName:    "",
	})
	m.AddMapping(&sourcemap.Mapping{
		GeneratedLine:   0,
		GeneratedColumn: 22,
		OriginalFile:    "a.md",
		OriginalLine:    1,
		OriginalColumn:  6,
		OriginalName:    "",
	})

	m.EncodeMappings()
	sm := bytes.Buffer{}
	err := m.WriteTo(&sm)
	if err != nil {
	}

	return sm.Bytes()
}

func sourceLine(s string, i int) string {
	return strings.Split(s, "\n")[i-1]
}

func TestEsbuild(t *testing.T) {
	m := mockSourcemap()
	bm := "," + base64.URLEncoding.EncodeToString(m)

	_ = bm
	t.Logf("%s", m)

	// esbuild Transform 支持输入 sourcemap
	// 不过在报错的时候产生的 Location 没有通过 sourcemap 转换，需要手动转换。
	result := api.Transform(`<><h1>h1 </h1><p>hih {a}</p></>;
//# sourceMappingURL=data:application/json;base64`+bm, api.TransformOptions{
		Loader:            api.LoaderJSX,
		Target:            api.ESNext,
		JSXMode:           api.JSXModeAutomatic,
		JSXDev:            false,
		Banner:            "",
		Platform:          api.PlatformNode,
		Format:            api.FormatIIFE,
		Sourcemap:         api.SourceMapInline,
		Sourcefile:        "root.js",
		MinifyIdentifiers: true,
	})

	if len(result.Errors) != 0 {
		var err error
		c, _ := sourcemap2.Parse("", m)
		e := result.Errors[0]
		if e.Location != nil {
			file := e.Location.File
			l := e.Location.Line
			i := e.Location.Column
			text := e.Location.LineText
			source, name, line, col, ok := c.Source(l, i)
			t.Logf("%v %v => %+v %v %v %v %v", l, i, source, name, line, col, ok)

			// 手动转换
			if ok {
				file = source
				l = line
				i = col
				text = sourceLine(c.SourceContent(source), line)
			}

			err = fmt.Errorf("\n%v: (%v:%v) \n%v\n%v^ %v\n", file, l, i, text, strings.Repeat(" ", i), e.Text)
		} else {
			err = fmt.Errorf("%v\n", e.Text)
		}
		t.Fatal(err)
	}

	t.Logf("%s", result.Code)

	code := string(result.Code)

	gx, _ := gojsx.NewJsx(gojsx.Option{})
	v, err := gx.ExecCode([]byte(code))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", v)
	// output :ReferenceError: a is not defined
	//        	at a.md:2:6
}

func TestLoad(t *testing.T) {
	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{"./test/Index.jsx"},
		Bundle:            true,
		JSXMode:           api.JSXModeAutomatic,
		JSXImportSource:   "../internal/js",
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,

		Loader: map[string]api.Loader{
			".png": api.LoaderDataURL,
			".svg": api.LoaderText,
		},
		OutExtensions: map[string]string{},
		Target:        api.ES2015,
		Write:         true,
		Sourcemap:     api.SourceMapInline,
	})

	if len(result.Errors) > 0 {
		for _, v := range result.Errors {
			t.Fatalf("%+v %+v", v.Text, v.Location)
		}
	}

	files := result.OutputFiles
	for _, f := range files {
		t.Logf("%+v %s", f.Path, f.Contents)
	}
}
