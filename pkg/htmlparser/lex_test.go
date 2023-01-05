package htmlparser

import (
	"bytes"
	"github.com/tdewolff/parse/v2"
	"io"
	"testing"
)

var JsxDom = []byte(`
<a href={"#"+id}>{title}</a>

<Toc item={item} enable={true} style={{margeTop: '1px'}} className="toc" disabled></Toc>
<Toc item={item}/>

<></>
`)

func TestName(t *testing.T) {
	l := NewLexer(parse.NewInput(bytes.NewBuffer(JsxDom)))
	for {
		err := l.Err()
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}

		tt, bs := l.Next()
		t.Logf("%+v %s", tt, bs)
	}
}
