package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/hashicorp/go-hclog"
)

// These are the environmental variables that determine if we log, and if
// we log whether or not the log should go to a file.
const (
	EnvLog     = "TF_LOG"      // Set to True
	EnvLogFile = "TF_LOG_PATH" // Set to a file
)

// ValidLevels are the log level names that Terraform recognizes.
var ValidLevels = []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR"}

// logger is the global hclog logger
var logger hclog.Logger

// logWriter is a global writer for logs, to be used with the std log package
var logWriter io.Writer

func init() {
	logger = NewHCLogger("")
	logWriter = logger.StandardWriter(&hclog.StandardLoggerOptions{InferLevels: true})

	// setup the default std library logger to use our output
	log.SetFlags(0)
	log.SetPrefix("")
	log.SetOutput(logWriter)
}

// SetupTempLog adds a new log sink which writes all logs to the given file.
func RegisterSink(f *os.File) {
	l, ok := logger.(hclog.InterceptLogger)
	if !ok {
		panic("global logger is not an InterceptLogger")
	}

	if f == nil {
		return
	}

	l.RegisterSink(hclog.NewSinkAdapter(&hclog.LoggerOptions{
		Level:  hclog.Trace,
		Output: f,
	}))
}

// LogOutput return the default global log io.Writer
func LogOutput() io.Writer {
	return logWriter
}

// HCLogger returns the default global hclog logger
func HCLogger() hclog.Logger {
	return logger
}

// NewHCLogger returns a new hclog.Logger instance with the given name
func NewHCLogger(name string) hclog.Logger {
	logOutput := io.Writer(os.Stderr)
	logLevel := CurrentLogLevel()
	if logLevel == "" {
		logOutput = ioutil.Discard
	}

	if logPath := os.Getenv(EnvLogFile); logPath != "" {
		f, err := os.OpenFile(logPath, syscall.O_CREAT|syscall.O_RDWR|syscall.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
		} else {
			logOutput = f
		}
	}

	return hclog.NewInterceptLogger(&hclog.LoggerOptions{
		Name:   name,
		Level:  hclog.LevelFromString(logLevel),
		Output: logOutput,
	})
}

func NewProviderLogger() hclog.Logger {
	return logger.Named("plugin")
}

// CurrentLogLevel returns the current log level string based the environment vars
func CurrentLogLevel() string {
	envLevel := strings.ToUpper(os.Getenv(EnvLog))
	if envLevel == "" {
		return ""
	}

	logLevel := "TRACE"
	if isValidLogLevel(envLevel) {
		logLevel = envLevel
	} else {
		logger.Warn(fmt.Sprintf("Invalid log level: %q. Defaulting to level: TRACE. Valid levels are: %+v",
			envLevel, ValidLevels))
	}

	return logLevel
}

// IsDebugOrHigher returns whether or not the current log level is debug or trace
func IsDebugOrHigher() bool {
	level := string(CurrentLogLevel())
	return level == "DEBUG" || level == "TRACE"
}

func isValidLogLevel(level string) bool {
	for _, l := range ValidLevels {
		if level == string(l) {
			return true
		}
	}

	return false
}
