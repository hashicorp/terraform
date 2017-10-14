package logging

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/hashicorp/logutils"
)

// These are the environmental variables that determine if we log, and if
// we log whether or not the log should go to a file.
const (
	EnvLog     = "TF_LOG"      // Set to True
	EnvLogFile = "TF_LOG_PATH" // Set to a file
)

var ValidLevels = []logutils.LogLevel{"TRACE", "DEBUG", "INFO", "WARN", "ERROR"}

// LogOutput determines where we should send logs (if anywhere) and the log level.
func LogOutput() (logOutput io.Writer, err error) {
	logOutput = ioutil.Discard

	logLevel := LogLevel()
	if logLevel == "" {
		return
	}

	logOutput = os.Stderr
	if logPath := os.Getenv(EnvLogFile); logPath != "" {
		var err error
		logOutput, err = os.OpenFile(logPath, syscall.O_CREAT|syscall.O_RDWR|syscall.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
	}

	// This was the default since the beginning
	logOutput = &logutils.LevelFilter{
		Levels:   ValidLevels,
		MinLevel: logutils.LogLevel(logLevel),
		Writer:   logOutput,
	}

	return
}

// SetOutput checks for a log destination with LogOutput, and calls
// log.SetOutput with the result. If LogOutput returns nil, SetOutput uses
// ioutil.Discard. Any error from LogOutout is fatal.
func SetOutput() {
	out, err := LogOutput()
	if err != nil {
		log.Fatal(err)
	}

	if out == nil {
		out = ioutil.Discard
	}

	log.SetOutput(out)
}

// LogLevel returns the current log level string based the environment vars
func LogLevel() string {
	envLevel := os.Getenv(EnvLog)
	if envLevel == "" {
		return ""
	}

	logLevel := "TRACE"
	if isValidLogLevel(envLevel) {
		// allow following for better ux: info, Info or INFO
		logLevel = strings.ToUpper(envLevel)
	} else {
		log.Printf("[WARN] Invalid log level: %q. Defaulting to level: TRACE. Valid levels are: %+v",
			envLevel, ValidLevels)
	}

	return logLevel
}

// IsDebugOrHigher returns whether or not the current log level is debug or trace
func IsDebugOrHigher() bool {
	level := string(LogLevel())
	return level == "DEBUG" || level == "TRACE"
}

func isValidLogLevel(level string) bool {
	for _, l := range ValidLevels {
		if strings.ToUpper(level) == string(l) {
			return true
		}
	}

	return false
}
