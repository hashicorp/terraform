package trace

import (
	"io"
	"log"
)

type LoggerPrinter struct {
	logger          *log.Logger
	writesToConsole bool
}

func NewWriterPrinter(writer io.Writer, writesToConsole bool) Printer {
	return &LoggerPrinter{
		logger:          log.New(writer, "", 0),
		writesToConsole: writesToConsole,
	}
}

func (p *LoggerPrinter) Print(v ...interface{}) {
	p.logger.Print(v...)
}

func (p *LoggerPrinter) Printf(format string, v ...interface{}) {
	p.logger.Printf(format, v...)
}

func (p *LoggerPrinter) Println(v ...interface{}) {
	p.logger.Println(v...)
}

func (p *LoggerPrinter) WritesToConsole() bool {
	return p.writesToConsole
}
