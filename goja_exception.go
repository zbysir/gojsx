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
	var errMsg strings.Builder
	stack := make([]string, 0)
	for _, s := range ss {
		if s == "" {
			continue
		} else if strings.HasPrefix(strings.TrimSpace(s), "at") {
			stack = append(stack, s)
		} else {
			if errMsg.Len() != 0 {
				errMsg.WriteByte('\n')
			}
			errMsg.WriteString(s)
		}
	}
	return &Exception{
		Text:   errMsg.String(),
		Stacks: stack,
	}
}

func PrettifyException(err error) error {
	// return err
	if ex, ok := err.(*goja.Exception); ok {
		return parseException(ex.String())
	}

	return err
}
