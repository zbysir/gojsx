package html2jsx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/tdewolff/parse/v2"
	"github.com/zbysir/gojsx/pkg/htmlparser"
	"io"
	"log"
)

// Convert
//
// enableJsx: 如果不开启，则会将 {} 处理成 html 编码
func Convert(reader io.Reader, writer io.Writer, enableJsx bool) error {
	c := ctx{}
	return c.Covert(reader, writer, enableJsx)
}

type ctx struct {
	currStartTag                 []byte
	dangerouslySetInnerHTML      bytes.Buffer
	dangerouslyHTMLStartTag      []byte
	startDangerouslySetInnerHTML bool
	debug                        bool
}

func (c *ctx) Covert(src io.Reader, writer io.Writer, enableJsx bool) error {
	l := htmlparser.NewLexer(parse.NewInput(src))
	//pos := 0

	for l.Err() == nil {
		//start := pos
		tt, bs := l.Next()

		//pos += len(bs)

		if c.debug {
			log.Printf("debug token: %s %s", tt, bs)
		}

		writer.Write(c.toJsxToken(tt, bs, enableJsx))
	}
	if l.Err() != io.EOF {
		return l.Err()
	}

	return nil
}

func encodeJsxInsecure(s []byte) []byte {
	s = bytes.ReplaceAll(s, []byte("{"), []byte("&#123;"))
	s = bytes.ReplaceAll(s, []byte("}"), []byte("&#125;"))
	s = bytes.ReplaceAll(s, []byte("<"), []byte("&lt;"))
	s = bytes.ReplaceAll(s, []byte(">"), []byte("&gt;"))
	return s
}

func toStringCode(s []byte) []byte {
	var bf bytes.Buffer
	je := json.NewEncoder(&bf)
	je.SetEscapeHTML(false)
	_ = je.Encode(string(s))
	return bytes.TrimSuffix(bf.Bytes(), []byte{'\n'})
}

// - script|pre|style|textarea 的子节点需要处理成纯文本
func (c *ctx) toJsxToken(tt htmlparser.TokenType, src []byte, enableJsx bool) []byte {
	if c.startDangerouslySetInnerHTML {
		switch tt {
		case htmlparser.EndTagToken:
			var tag = src[2 : len(src)-1]
			if bytes.Equal(c.dangerouslyHTMLStartTag, tag) {
				inner := toStringCode(c.dangerouslySetInnerHTML.Bytes())
				bs := []byte(fmt.Sprintf(` dangerouslySetInnerHTML={{ __html: %s }}>%s`, inner, src))
				c.dangerouslySetInnerHTML.Reset()
				c.dangerouslyHTMLStartTag = nil
				c.startDangerouslySetInnerHTML = false
				return bs
			}
		}
		c.dangerouslySetInnerHTML.Write(src)
		return nil
	}
	switch tt {
	case htmlparser.StartTagToken:
		var tag = src[1:]
		if in(tag, []string{"script", "pre", "style", "textarea"}) {
			c.dangerouslyHTMLStartTag = tag
		} else if len(tag) == 0 {
			// for <
			src = encodeJsxInsecure(src)
		} else {
			// for <Component
			src = bytes.ToLower(src)
		}
		c.currStartTag = tag
	case htmlparser.StartTagVoidToken:
		if bytes.Equal(c.dangerouslyHTMLStartTag, c.currStartTag) {
			c.startDangerouslySetInnerHTML = false
			c.dangerouslyHTMLStartTag = nil
		}
	case htmlparser.StartTagCloseToken:
		if c.dangerouslyHTMLStartTag != nil {
			c.startDangerouslySetInnerHTML = true
			return nil
		}
		// for >
		if len(c.currStartTag) == 0 {
			src = encodeJsxInsecure(src)
		}
	case htmlparser.EndTagToken:
		// for </>
		if len(c.currStartTag) == 0 {
			src = encodeJsxInsecure(src)
		} else {
			src = bytes.ToLower(src)
		}
	case htmlparser.TextToken:
		if !enableJsx {
			src = encodeJsxInsecure(src)
		}
	case htmlparser.CommentToken:
		src = encodeJsxInsecure(src)
	case htmlparser.AttributeToken:
		kvs := bytes.Split(src, []byte("="))
		if len(kvs) == 2 {
			v := bytes.Trim(kvs[1], " ")
			if !bytes.HasSuffix(v, []byte(`"`)) || !bytes.HasPrefix(v, []byte(`"`)) {
				v, _ = json.Marshal(string(v))
			}
			src = []byte(fmt.Sprintf("%s=%s", kvs[0], v))
		}
	}

	return src
}

func in(s []byte, i []string) bool {
	for _, item := range i {
		if item == string(s) {
			return true
		}
	}

	return false
}
