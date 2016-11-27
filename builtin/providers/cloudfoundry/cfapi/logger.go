package cfapi

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/kr/pretty"

	"code.cloudfoundry.org/cli/cf/terminal"
	"code.cloudfoundry.org/cli/cf/trace"
)

// Logger -
type Logger struct {
	TracePrinter trace.Printer
	UI           terminal.UI
	isDebug      bool
}

// NewLogger -
func NewLogger(debug bool, tracePath string) *Logger {

	l := &Logger{}

	if _, err := os.Stat(tracePath); os.IsExist(err) {
		l.TracePrinter = trace.NewLogger(os.Stdout, true, tracePath, "")
	} else {
		l.TracePrinter = trace.NewLogger(os.Stdout, strings.ToLower(tracePath) == "true", "", "")
	}

	l.UI = terminal.NewUI(os.Stdin, os.Stdout, terminal.NewTeePrinter(os.Stdout), l.TracePrinter)
	l.isDebug = debug

	return l
}

// LogMessage -
func (l *Logger) LogMessage(format string, v ...interface{}) {
	l.TracePrinter.Printf(format, v)
}

// DebugMessage -
func (l *Logger) DebugMessage(format string, v ...interface{}) {
	if l.isDebug {
		vv := []interface{}{}
		for _, o := range v {
			k := reflect.ValueOf(o).Kind()
			if k == reflect.Struct ||
				k == reflect.Interface ||
				k == reflect.Ptr ||
				k == reflect.Slice ||
				k == reflect.Map {
				vv = append(vv, pretty.Formatter(o))
			} else {
				vv = append(vv, o)
			}
		}
		hdr := terminal.HeaderColor(fmt.Sprintf("[%s] DEBUG:", time.Now().Format(time.RFC3339)))
		l.UI.Say(fmt.Sprintf("%s %s", hdr, format), vv...)
	}
}
