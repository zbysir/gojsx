package jsx

import "testing"

func TestTransform(t *testing.T) {
	x := NewEsBuildTransform(false)

	t.Run("json", func(t *testing.T) {
		b, err := x.Transform("1.json", []byte(`{"a":1}`))
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s", b)

	})

	t.Run("css", func(t *testing.T) {
		b, err := x.Transform("1.css", []byte(`.a{color: red}`))
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s", b)
	})
}
