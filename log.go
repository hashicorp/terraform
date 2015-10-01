package main

import (
	"io"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/logutils"
)

// These are the environmental variables that determine if we log, and if
// we log whether or not the log should go to a file.
const (
	EnvLog      = "TF_LOG"       // Set to True
	EnvLogLevel = "TF_LOG_LEVEL" // Set to a log level
	EnvLogFile  = "TF_LOG_PATH"  // Set to a file
)

var validLevels = []logutils.LogLevel{"TRACE", "DEBUG", "INFO", "WARN", "ERROR"}

// logOutput determines where we should send logs (if anywhere) and the log level.
func logOutput() (logOutput io.Writer, err error) {
	logOutput = nil
	if os.Getenv(EnvLog) == "" {
		return
	}

	logOutput = os.Stderr

	if logPath := os.Getenv(EnvLogFile); logPath != "" {
		var err error
		logOutput, err = os.Create(logPath)
		if err != nil {
			return nil, err
		}
	}

	// This was the default since the beginning
	logLevel := logutils.LogLevel("TRACE")

	if level := os.Getenv(EnvLogLevel); level != "" {
		if isValidLevel(level) {
			// allow following for better ux: info, Info or INFO
			logLevel = logutils.LogLevel(strings.ToUpper(level))
		} else {
			log.Printf("[WARN] Invalid log level: %q. Valid levels are: %+v",
				level, validLevels)
		}
	}

	logOutput = &logutils.LevelFilter{
		Levels:   validLevels,
		MinLevel: logLevel,
		Writer:   logOutput,
	}

	return
}

func isValidLevel(level string) bool {
	for _, l := range validLevels {
		if strings.ToUpper(level) == string(l) {
			return true
		}
	}

	return false
}
