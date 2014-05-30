package main

import (
	"io"
	"os"
)

// These are the environmental variables that determine if we log, and if
// we log whether or not the log should go to a file.
const EnvLog = "TF_LOG"
const EnvLogFile = "TF_LOG_PATH"

// logOutput determines where we should send logs (if anywhere).
func logOutput() (logOutput io.Writer, err error) {
	logOutput = nil
	if os.Getenv(EnvLog) != "" {
		logOutput = os.Stderr

		if logPath := os.Getenv(EnvLogFile); logPath != "" {
			var err error
			logOutput, err = os.Create(logPath)
			if err != nil {
				return nil, err
			}
		}
	}

	return
}
