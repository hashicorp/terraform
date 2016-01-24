package logging

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/logutils"
)

// These are the environmental variables that determine if we log, and if
// we log whether or not the log should go to a file.
const (
	EnvLog     = "TF_LOG"      // Set to True
	EnvLogFile = "TF_LOG_PATH" // Set to a file
)

var validLevels = []logutils.LogLevel{"TRACE", "DEBUG", "INFO", "WARN", "ERROR"}

// LogOutput determines where we should send logs (if anywhere) and the log level.
func LogOutput() (logOutput io.Writer, err error) {
	logOutput = ioutil.Discard
	envLevel := os.Getenv(EnvLog)
	if envLevel == "" {
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

	if isValidLogLevel(envLevel) {
		// allow following for better ux: info, Info or INFO
		logLevel = logutils.LogLevel(strings.ToUpper(envLevel))
	} else {
		log.Printf("[WARN] Invalid log level: %q. Defaulting to level: TRACE. Valid levels are: %+v",
			envLevel, validLevels)
	}

	logOutput = &logutils.LevelFilter{
		Levels:   validLevels,
		MinLevel: logLevel,
		Writer:   logOutput,
	}

	return
}

func isValidLogLevel(level string) bool {
	for _, l := range validLevels {
		if strings.ToUpper(level) == string(l) {
			return true
		}
	}

	return false
}
