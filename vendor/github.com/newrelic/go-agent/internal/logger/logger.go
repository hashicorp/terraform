package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// Logger matches newrelic.Logger to allow implementations to be passed to
// internal packages.
type Logger interface {
	Error(msg string, context map[string]interface{})
	Warn(msg string, context map[string]interface{})
	Info(msg string, context map[string]interface{})
	Debug(msg string, context map[string]interface{})
	DebugEnabled() bool
}

// ShimLogger implements Logger and does nothing.
type ShimLogger struct{}

// Error allows ShimLogger to implement Logger.
func (s ShimLogger) Error(string, map[string]interface{}) {}

// Warn allows ShimLogger to implement Logger.
func (s ShimLogger) Warn(string, map[string]interface{}) {}

// Info allows ShimLogger to implement Logger.
func (s ShimLogger) Info(string, map[string]interface{}) {}

// Debug allows ShimLogger to implement Logger.
func (s ShimLogger) Debug(string, map[string]interface{}) {}

// DebugEnabled allows ShimLogger to implement Logger.
func (s ShimLogger) DebugEnabled() bool { return false }

type logFile struct {
	l       *log.Logger
	doDebug bool
}

// New creates a basic Logger.
func New(w io.Writer, doDebug bool) Logger {
	return &logFile{
		l:       log.New(w, logPid, logFlags),
		doDebug: doDebug,
	}
}

const logFlags = log.Ldate | log.Ltime | log.Lmicroseconds

var (
	logPid = fmt.Sprintf("(%d) ", os.Getpid())
)

func (f *logFile) fire(level, msg string, ctx map[string]interface{}) {
	js, err := json.Marshal(struct {
		Level   string                 `json:"level"`
		Event   string                 `json:"msg"`
		Context map[string]interface{} `json:"context"`
	}{
		level,
		msg,
		ctx,
	})
	if nil == err {
		f.l.Printf(string(js))
	} else {
		f.l.Printf("unable to marshal log entry: %v", err)
	}
}

func (f *logFile) Error(msg string, ctx map[string]interface{}) {
	f.fire("error", msg, ctx)
}
func (f *logFile) Warn(msg string, ctx map[string]interface{}) {
	f.fire("warn", msg, ctx)
}
func (f *logFile) Info(msg string, ctx map[string]interface{}) {
	f.fire("info", msg, ctx)
}
func (f *logFile) Debug(msg string, ctx map[string]interface{}) {
	if f.doDebug {
		f.fire("debug", msg, ctx)
	}
}
func (f *logFile) DebugEnabled() bool { return f.doDebug }
