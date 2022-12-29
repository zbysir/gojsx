package jsx

import "testing"

func TestTransform(t *testing.T) {
	x := NewEsBuildTransform(false)

	t.Run("json", func(t *testing.T) {
		b, err := x.Transform("1.json", []byte(`{"a":1}`), TransformerFormatCommonJS)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s", b)

	})

	t.Run("css", func(t *testing.T) {
		b, err := x.Transform("1.css", []byte(`.a{color: red}`), TransformerFormatCommonJS)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s", b)
	})

	t.Run("tsx", func(t *testing.T) {
		b, err := x.Transform("1.tsx", []byte(`import HelloJSX from './index.tsx'; module.exports = <HelloJSX></HelloJSX>`), TransformerFormatIIFE)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s", b)
	})
}
