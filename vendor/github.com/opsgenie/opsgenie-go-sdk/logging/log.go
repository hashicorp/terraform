/*
Copyright 2015 OpsGenie. All rights reserved.
Use of this source code is governed by a Apache Software
license that can be found in the LICENSE file.
*/

//Package logging provides log interface.
package logging

import (
	"fmt"

	"github.com/cihub/seelog"
)

// logger is the internal logger object.
var logger seelog.LoggerInterface

func init() {
	DisableLog()
}

// DisableLog disables all library log output.
func DisableLog() {
	logger = seelog.Disabled
}

// UseLogger is a wrapper for Seelog's UseLogger function. It sets the newLogger as the current logger.
func UseLogger(newLogger seelog.LoggerInterface) {
	logger = newLogger
	seelog.UseLogger(logger)
}

// Logger returns internal logger object to achieve logging.
func Logger() seelog.LoggerInterface {
	return logger
}

// ConfigureLogger configures the new logger according to the configuration and sets it as the current logger.
func ConfigureLogger(testConfig []byte) {
	loggr, err := seelog.LoggerFromConfigAsBytes([]byte(testConfig))
	if err != nil {
		fmt.Printf("error occured: %s\n", err.Error())
	}
	UseLogger(loggr)
}

// FlushLog is a wrapper for seelog's Flush function. It flushes all the messages in the logger.
func FlushLog() {
	logger.Flush()
}
