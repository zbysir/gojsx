package console

import (
	"log"

	"github.com/dop251/goja"
)

type Printer interface {
	Log(string)
	Warn(string)
	Error(string)
}

type PrinterFunc func(s string)

func (p PrinterFunc) Log(s string) { p(s) }

func (p PrinterFunc) Warn(s string) { p(s) }

func (p PrinterFunc) Error(s string) { p(s) }

var defaultPrinter Printer = PrinterFunc(func(s string) { log.Print(s) })

func logx(rt *goja.Runtime, p func(string)) func(goja.FunctionCall) goja.Value {
	ut := Util{rt}
	return func(call goja.FunctionCall) goja.Value {
		p(ut.Js_format(call).String())
		return nil
	}
}

func Enable(runtime *goja.Runtime, printer Printer) {
	if printer == nil {
		printer = defaultPrinter
	}
	runtime.Set("console", map[string]interface{}{
		"log":   logx(runtime, printer.Log),
		"error": logx(runtime, printer.Error),
		"warn":  logx(runtime, printer.Warn),
	})
}
