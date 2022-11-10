package jsx

import (
	"github.com/dop251/goja"
	"strings"
)

type Exception struct {
	Text   string
	Stacks []string
}

func (e *Exception) Error() string {
	var b strings.Builder
	b.WriteString(e.Text)
	for _, s := range e.Stacks {
		// skip golang require function Stack
		if strings.HasSuffix(s, "(native)") {
			continue
		}

		// rm pt
		if strings.HasSuffix(s, "))") {
			i := strings.LastIndex(s, "(")
			s = s[:i] + s[len(s)-1:]
		} else if strings.HasSuffix(s, ")") {
			i := strings.LastIndex(s, "(")
			s = s[:i]
		}

		b.WriteString("\n\t")
		b.WriteString(strings.TrimSpace(s))
	}

	return b.String()
}

func parseException(s string) error {
	ss := strings.Split(s, "\n")
	if len(ss) == 0 {
		return nil
	}
	errMsg := ss[0]
	stack := make([]string, 0)
	for _, s := range ss[1:] {
		if len(s) != 0 {
			stack = append(stack, s)
		}
	}
	return &Exception{
		Text:   errMsg,
		Stacks: stack,
	}
}

func prettifyException(err error) error {
	if ex, ok := err.(*goja.Exception); ok {
		return parseException(ex.String())
	}

	return err
}
