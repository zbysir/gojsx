package jsx

import (
	"github.com/dop251/goja"
	"strings"
)

type exception struct {
	error string
	stack []string
}

func (e *exception) Error() string {
	var b strings.Builder
	b.WriteString(e.error)
	for _, s := range e.stack {
		// skip golang require function stack
		if strings.HasSuffix(s, "(native)") {
			continue
		}
		if strings.HasSuffix(s, "))") {
			i := strings.LastIndex(s, "(")
			s = s[:i] + s[len(s)-1:]
		} else if strings.HasSuffix(s, ")") {
			i := strings.LastIndex(s, "(")
			s = s[:i]
		}

		b.WriteByte('\n')
		b.WriteString(strings.TrimSpace(s))
	}

	return b.String()
}

func parseException(s string) *exception {
	ss := strings.Split(s, "\n")
	if len(ss) == 0 {
		return nil
	}
	errMsg := ss[0]
	stack := make([]string, len(ss)-1)
	for i, s := range ss[1:] {
		stack[i] = s
	}
	return &exception{
		error: errMsg,
		stack: stack,
	}
}

func prettifyException(err error) error {
	if ex, ok := err.(*goja.Exception); ok {
		return parseException(ex.String())
	}

	return err
}
