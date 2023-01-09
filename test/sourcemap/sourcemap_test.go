package sourcemap

import (
	"bytes"
	sourcemap2 "github.com/go-sourcemap/sourcemap"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSourceMap(t *testing.T) {
	m := Map{
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
	//
	//
	// <><h1>h1 </h1><p>hih {a}</p></>
	//       ^6         ^17  ^22
	//

	m.AddMapping(&Mapping{
		GeneratedLine:   0,
		GeneratedColumn: 6,
		OriginalFile:    "a.md",
		OriginalLine:    0,
		OriginalColumn:  2,
		OriginalName:    "",
	})
	m.AddMapping(&Mapping{
		GeneratedLine:   0,
		GeneratedColumn: 17,
		OriginalFile:    "a.md",
		OriginalLine:    1,
		OriginalColumn:  0,
		OriginalName:    "",
	})
	m.AddMapping(&Mapping{
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
		t.Fatal(err)
	}

	c, err := sourcemap2.Parse("./", sm.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	source, name, line, column, ok := c.Source(1, 6)
	assert.Equal(t, true, ok)
	assert.Equal(t, line, 1)
	assert.Equal(t, column, 2)
	assert.Equal(t, name, "")
	assert.Equal(t, source, "a.md")
}
