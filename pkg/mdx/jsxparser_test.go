package mdx

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseToClose(t *testing.T) {
	cases := []struct {
		Name string
		In   string
		Out  string
	}{
		{
			Name: "Base",
			In: `

<Name name={'bysir'}></Name>

133

`,
			Out: "<Name name={'bysir'}></Name>",
		},
		{
			Name: "SelfClose",
			In: `

<Name name={'bysir'} r/>333

`,
			Out: "<Name name={'bysir'} r/>",
		},
		{
			Name: "Lines",
			In: `

<Name name={'bysir'}>333

</Name>
`,
			Out: "<Name name={'bysir'}>333\n\n</Name>",
		},
		{
			Name: "Nesting",
			In: `

<Name name={'bysir'}>333
<Name></Name>
<a></a>
</Name>
`,
			Out: "<Name name={'bysir'}>333\n<Name></Name>\n<a></a>\n</Name>",
		},
		{
			Name: "Pure",
			In: `
<searchbtn></searchbtn>`,
			Out: "<searchbtn></searchbtn>",
		},
		{
			Name: "OneLetter",
			In:   `<A></A>`,
			Out:  "<A></A>",
		},
		{
			Name: "fragment",
			In: `<>
  <p> {1} </p>
</>`,
			Out: "<>\n  <p> {1} </p>\n</>",
		},
		{
			Name: "blankLine",
			In: `

<>
  <Footer></Footer>
</>`,
			Out: "<>\n  <Footer></Footer>\n</>",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			buf := bytes.NewBuffer([]byte(c.In))
			s, e, ok, err := parseTagToClose(buf)
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				panic("not ok")
			}

			assert.Equal(t, c.Out, string([]byte(c.In)[s:e]))
		})
	}

}
