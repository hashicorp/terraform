package trace

//go:generate counterfeiter . Printer

type Printer interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
	WritesToConsole() bool
}
