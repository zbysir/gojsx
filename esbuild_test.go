package gojsx

import (
	"github.com/evanw/esbuild/pkg/api"
	"testing"
)

func TestEsbuild(t *testing.T) {
	result := api.Transform("export default function A (){return <div> 123</div>}", api.TransformOptions{
		Loader:  api.LoaderJSX,
		Target:  api.ES2015,
		JSXMode: api.JSXModeAutomatic,
		JSXDev:  true,
		Banner:  "",
		Format:  api.FormatCommonJS,
	})

	if len(result.Errors) != 0 {
		t.Fatalf("%+v", result.Errors)
	}
	t.Logf("%s", result.Code)
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
