package mdx

import (
	"bytes"
	"github.com/tdewolff/parse/v2"
	"github.com/zbysir/gojsx/pkg/htmlparser"
	"io"
)

func parseTagToClose(buf *bytes.Buffer) (start, end int, ok bool, err error) {
	input := parse.NewInput(buf)

	l := htmlparser.NewLexer(input)

	nesting := 0
	var currTag []byte
	var matchTag []byte
	pos := 0

	for end == 0 {
		err := l.Err()
		if err != nil {
			if err == io.EOF {
				break
			}

			return 0, 0, false, err
		}

		tp, bs := l.Next()

		//log.Printf("parseTagToClose: %s %s", tp, bs)

		begin := pos
		pos += len(bs)
		switch tp {
		case htmlparser.StartTagToken:
			currTag = bs[1:]
			if matchTag == nil {
				matchTag = bs[1:]
				nesting += 1
				start = begin
			} else if bytes.Equal(matchTag, bs[1:]) {
				nesting += 1
			}
		case htmlparser.StartTagVoidToken:
			if bytes.Equal(matchTag, currTag) {
				nesting -= 1
				if nesting == 0 {
					end = pos
					break
				}
			}
		case htmlparser.EndTagToken:
			if bytes.Equal(matchTag, bs[2:len(bs)-1]) {
				nesting -= 1
				if nesting == 0 {
					end = pos
					break
				}
			}
		}
	}
	if end != 0 {
		return start, end, true, nil
	}

	return
}
