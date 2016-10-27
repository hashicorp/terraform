package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelNone LogLevel = 99
)

var levels = map[string]LogLevel{
	"DEBUG": LevelDebug,
	"INFO":  LevelInfo,
	"WARN":  LevelWarn,
	"ERROR": LevelError,
	"NONE":  LevelNone,
}
var levelKeys = []string{"DEBUG", "INFO", "WARN", "ERROR", "NONE"}

func Levelify(levelString string) (LogLevel, error) {
	upperLevelString := strings.ToUpper(levelString)
	level, ok := levels[upperLevelString]
	if !ok {
		expected := strings.Join(levelKeys, ", ")
		return level, fmt.Errorf("Unknown LogLevel string '%s', expected one of [%s]", levelString, expected)
	}
	return level, nil
}

type Logger interface {
	Debug(tag, msg string, args ...interface{})
	DebugWithDetails(tag, msg string, args ...interface{})
	Info(tag, msg string, args ...interface{})
	Warn(tag, msg string, args ...interface{})
	Error(tag, msg string, args ...interface{})
	ErrorWithDetails(tag, msg string, args ...interface{})
	HandlePanic(tag string)
	ToggleForcedDebug()
	Flush() error
	FlushTimeout(time.Duration) error
}

type logger struct {
	level       LogLevel
	out         *log.Logger
	err         *log.Logger
	forcedDebug bool
	outMu       sync.Mutex
	errMu       sync.Mutex
}

func New(level LogLevel, out, err *log.Logger) Logger {
	return &logger{
		level: level,
		out:   out,
		err:   err,
	}
}

func NewLogger(level LogLevel) Logger {
	return NewWriterLogger(level, os.Stdout, os.Stderr)
}

func NewWriterLogger(level LogLevel, out, err io.Writer) Logger {
	return New(
		level,
		log.New(out, "", log.LstdFlags),
		log.New(err, "", log.LstdFlags),
	)
}

func (l *logger) Flush() error                       { return nil }
func (l *logger) FlushTimeout(_ time.Duration) error { return nil }

func (l *logger) Debug(tag, msg string, args ...interface{}) {
	if l.level > LevelDebug && !l.forcedDebug {
		return
	}

	msg = "DEBUG - " + msg
	l.outPrintf(tag, msg, args...)
}

// DebugWithDetails will automatically change the format of the message
// to insert a block of text after the log
func (l *logger) DebugWithDetails(tag, msg string, args ...interface{}) {
	msg = msg + "\n********************\n%s\n********************"
	l.Debug(tag, msg, args...)
}

func (l *logger) Info(tag, msg string, args ...interface{}) {
	if l.level > LevelInfo && !l.forcedDebug {
		return
	}

	msg = "INFO - " + msg
	l.outPrintf(tag, msg, args...)
}

func (l *logger) Warn(tag, msg string, args ...interface{}) {
	if l.level > LevelWarn && !l.forcedDebug {
		return
	}

	msg = "WARN - " + msg
	l.errPrintf(tag, msg, args...)
}

func (l *logger) Error(tag, msg string, args ...interface{}) {
	if l.level > LevelError && !l.forcedDebug {
		return
	}

	msg = "ERROR - " + msg
	l.errPrintf(tag, msg, args...)
}

// ErrorWithDetails will automatically change the format of the message
// to insert a block of text after the log
func (l *logger) ErrorWithDetails(tag, msg string, args ...interface{}) {
	msg = msg + "\n********************\n%s\n********************"
	l.Error(tag, msg, args...)
}

func (l *logger) recoverPanic(tag string) (didPanic bool) {
	if e := recover(); e != nil {
		var msg string
		switch obj := e.(type) {
		case string:
			msg = obj
		case fmt.Stringer:
			msg = obj.String()
		case error:
			msg = obj.Error()
		default:
			msg = fmt.Sprintf("%#v", obj)
		}
		l.ErrorWithDetails(tag, "Panic: %s", msg, debug.Stack())
		return true
	}
	return false
}

func (l *logger) HandlePanic(tag string) {
	if l.recoverPanic(tag) {
		os.Exit(2)
	}
}

func (l *logger) ToggleForcedDebug() {
	l.forcedDebug = !l.forcedDebug
}

func (l *logger) errPrintf(tag, msg string, args ...interface{}) {
	s := fmt.Sprintf(msg, args...)
	l.errMu.Lock()
	l.err.SetPrefix("[" + tag + "] ")
	l.err.Output(2, s)
	l.errMu.Unlock()
}

func (l *logger) outPrintf(tag, msg string, args ...interface{}) {
	s := fmt.Sprintf(msg, args...)
	l.outMu.Lock()
	l.out.SetPrefix("[" + tag + "] ")
	l.out.Output(2, s)
	l.outMu.Unlock()
}
