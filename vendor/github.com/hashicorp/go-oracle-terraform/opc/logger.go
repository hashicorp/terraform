package opc

import (
	"io"
	"io/ioutil"
	"log"
	"os"
)

const (
	LogOff   LogLevelType = 0
	LogDebug LogLevelType = 1
)

type LogLevelType uint

// Logger interface. Should be satisfied by Terraform's logger as well as the Default logger
type Logger interface {
	Log(...interface{})
}

type LoggerFunc func(...interface{})

func (f LoggerFunc) Log(args ...interface{}) {
	f(args...)
}

// Returns a default logger if one isn't specified during configuration
func NewDefaultLogger() Logger {
	logWriter, err := LogOutput()
	if err != nil {
		log.Fatalf("Error setting up log writer: %s", err)
	}
	return &defaultLogger{
		logger: log.New(logWriter, "", log.LstdFlags),
	}
}

// Default logger to satisfy the logger interface
type defaultLogger struct {
	logger *log.Logger
}

func (l defaultLogger) Log(args ...interface{}) {
	l.logger.Println(args...)
}

func LogOutput() (logOutput io.Writer, err error) {
	// Default to nil
	logOutput = ioutil.Discard

	logLevel := LogLevel()
	if logLevel == LogOff {
		return
	}

	// Logging is on, set output to STDERR
	logOutput = os.Stderr
	return
}

// Gets current Log Level from the ORACLE_LOG env var
func LogLevel() LogLevelType {
	envLevel := os.Getenv("ORACLE_LOG")
	if envLevel == "" {
		return LogOff
	} else {
		return LogDebug
	}
}
