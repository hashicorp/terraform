package trace

import (
	"io"
	"os"
	"path/filepath"
	"strconv"

	. "code.cloudfoundry.org/cli/cf/i18n"
)

func NewLogger(writer io.Writer, verbose bool, cfTrace, configTrace string) Printer {
	LoggingToStdout = verbose

	var printers []Printer

	stdoutLogger := NewWriterPrinter(writer, true)

	for _, path := range []string{cfTrace, configTrace} {
		b, err := strconv.ParseBool(path)
		LoggingToStdout = LoggingToStdout || b

		if path != "" && err != nil {
			var file *os.File
			err = os.MkdirAll(filepath.Dir(path), os.ModeDir|os.ModePerm)
			if err == nil {
				file, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
			}

			if err == nil {
				printers = append(printers, NewWriterPrinter(file, false))
			} else {
				stdoutLogger.Printf(T("CF_TRACE ERROR CREATING LOG FILE {{.Path}}:\n{{.Err}}",
					map[string]interface{}{"Path": path, "Err": err}))

				LoggingToStdout = true
			}
		}
	}

	if LoggingToStdout {
		printers = append(printers, stdoutLogger)
	}

	return CombinePrinters(printers)
}
