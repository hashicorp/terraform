// Copyright (c) 2012 - Cloud Instruments Co., Ltd.
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package seelog

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	staticFuncCallDepth = 3 // See 'commonLogger.log' method comments
	loggerFuncCallDepth = 3
)

// Current is the logger used in all package level convenience funcs like 'Trace', 'Debug', 'Flush', etc.
var Current LoggerInterface

// Default logger that is created from an empty config: "<seelog/>". It is not closed by a ReplaceLogger call.
var Default LoggerInterface

// Disabled logger that doesn't produce any output in any circumstances. It is neither closed nor flushed by a ReplaceLogger call.
var Disabled LoggerInterface

var pkgOperationsMutex *sync.Mutex

func init() {
	pkgOperationsMutex = new(sync.Mutex)
	var err error

	if Default == nil {
		Default, err = LoggerFromConfigAsBytes([]byte("<seelog />"))
	}

	if Disabled == nil {
		Disabled, err = LoggerFromConfigAsBytes([]byte("<seelog levels=\"off\"/>"))
	}

	if err != nil {
		panic(fmt.Sprintf("Seelog couldn't start. Error: %s", err.Error()))
	}

	Current = Default
}

func createLoggerFromFullConfig(config *configForParsing) (LoggerInterface, error) {
	if config.LogType == syncloggerTypeFromString {
		return NewSyncLogger(&config.logConfig), nil
	} else if config.LogType == asyncLooploggerTypeFromString {
		return NewAsyncLoopLogger(&config.logConfig), nil
	} else if config.LogType == asyncTimerloggerTypeFromString {
		logData := config.LoggerData
		if logData == nil {
			return nil, errors.New("async timer data not set")
		}

		asyncInt, ok := logData.(asyncTimerLoggerData)
		if !ok {
			return nil, errors.New("invalid async timer data")
		}

		logger, err := NewAsyncTimerLogger(&config.logConfig, time.Duration(asyncInt.AsyncInterval))
		if !ok {
			return nil, err
		}

		return logger, nil
	} else if config.LogType == adaptiveLoggerTypeFromString {
		logData := config.LoggerData
		if logData == nil {
			return nil, errors.New("adaptive logger parameters not set")
		}

		adaptData, ok := logData.(adaptiveLoggerData)
		if !ok {
			return nil, errors.New("invalid adaptive logger parameters")
		}

		logger, err := NewAsyncAdaptiveLogger(
			&config.logConfig,
			time.Duration(adaptData.MinInterval),
			time.Duration(adaptData.MaxInterval),
			adaptData.CriticalMsgCount,
		)
		if err != nil {
			return nil, err
		}

		return logger, nil
	}
	return nil, errors.New("invalid config log type/data")
}

// UseLogger sets the 'Current' package level logger variable to the specified value.
// This variable is used in all Trace/Debug/... package level convenience funcs.
//
// Example:
//
// after calling
//     seelog.UseLogger(somelogger)
// the following:
//     seelog.Debug("abc")
// will be equal to
//     somelogger.Debug("abc")
//
// IMPORTANT: UseLogger do NOT close the previous logger (only flushes it). So if
// you constantly use it to replace loggers and don't close them in other code, you'll
// end up having memory leaks.
//
// To safely replace loggers, use ReplaceLogger.
func UseLogger(logger LoggerInterface) error {
	if logger == nil {
		return errors.New("logger can not be nil")
	}

	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()

	oldLogger := Current
	Current = logger

	if oldLogger != nil {
		oldLogger.Flush()
	}

	return nil
}

// ReplaceLogger acts as UseLogger but the logger that was previously
// used is disposed (except Default and Disabled loggers).
//
// Example:
//     import log "github.com/cihub/seelog"
//
//     func main() {
//         logger, err := log.LoggerFromConfigAsFile("seelog.xml")
//
//         if err != nil {
//             panic(err)
//         }
//
//         log.ReplaceLogger(logger)
//         defer log.Flush()
//
//         log.Trace("test")
//         log.Debugf("var = %s", "abc")
//     }
func ReplaceLogger(logger LoggerInterface) error {
	if logger == nil {
		return errors.New("logger can not be nil")
	}

	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()

	defer func() {
		if err := recover(); err != nil {
			reportInternalError(fmt.Errorf("recovered from panic during ReplaceLogger: %s", err))
		}
	}()

	if Current == Default {
		Current.Flush()
	} else if Current != nil && !Current.Closed() && Current != Disabled {
		Current.Flush()
		Current.Close()
	}

	Current = logger

	return nil
}

// Tracef formats message according to format specifier
// and writes to default logger with log level = Trace.
func Tracef(format string, params ...interface{}) {
	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()
	Current.traceWithCallDepth(staticFuncCallDepth, newLogFormattedMessage(format, params))
}

// Debugf formats message according to format specifier
// and writes to default logger with log level = Debug.
func Debugf(format string, params ...interface{}) {
	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()
	Current.debugWithCallDepth(staticFuncCallDepth, newLogFormattedMessage(format, params))
}

// Infof formats message according to format specifier
// and writes to default logger with log level = Info.
func Infof(format string, params ...interface{}) {
	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()
	Current.infoWithCallDepth(staticFuncCallDepth, newLogFormattedMessage(format, params))
}

// Warnf formats message according to format specifier and writes to default logger with log level = Warn
func Warnf(format string, params ...interface{}) error {
	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()
	message := newLogFormattedMessage(format, params)
	Current.warnWithCallDepth(staticFuncCallDepth, message)
	return errors.New(message.String())
}

// Errorf formats message according to format specifier and writes to default logger with log level = Error
func Errorf(format string, params ...interface{}) error {
	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()
	message := newLogFormattedMessage(format, params)
	Current.errorWithCallDepth(staticFuncCallDepth, message)
	return errors.New(message.String())
}

// Criticalf formats message according to format specifier and writes to default logger with log level = Critical
func Criticalf(format string, params ...interface{}) error {
	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()
	message := newLogFormattedMessage(format, params)
	Current.criticalWithCallDepth(staticFuncCallDepth, message)
	return errors.New(message.String())
}

// Trace formats message using the default formats for its operands and writes to default logger with log level = Trace
func Trace(v ...interface{}) {
	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()
	Current.traceWithCallDepth(staticFuncCallDepth, newLogMessage(v))
}

// Debug formats message using the default formats for its operands and writes to default logger with log level = Debug
func Debug(v ...interface{}) {
	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()
	Current.debugWithCallDepth(staticFuncCallDepth, newLogMessage(v))
}

// Info formats message using the default formats for its operands and writes to default logger with log level = Info
func Info(v ...interface{}) {
	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()
	Current.infoWithCallDepth(staticFuncCallDepth, newLogMessage(v))
}

// Warn formats message using the default formats for its operands and writes to default logger with log level = Warn
func Warn(v ...interface{}) error {
	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()
	message := newLogMessage(v)
	Current.warnWithCallDepth(staticFuncCallDepth, message)
	return errors.New(message.String())
}

// Error formats message using the default formats for its operands and writes to default logger with log level = Error
func Error(v ...interface{}) error {
	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()
	message := newLogMessage(v)
	Current.errorWithCallDepth(staticFuncCallDepth, message)
	return errors.New(message.String())
}

// Critical formats message using the default formats for its operands and writes to default logger with log level = Critical
func Critical(v ...interface{}) error {
	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()
	message := newLogMessage(v)
	Current.criticalWithCallDepth(staticFuncCallDepth, message)
	return errors.New(message.String())
}

// Flush immediately processes all currently queued messages and all currently buffered messages.
// It is a blocking call which returns only after the queue is empty and all the buffers are empty.
//
// If Flush is called for a synchronous logger (type='sync'), it only flushes buffers (e.g. '<buffered>' receivers)
// , because there is no queue.
//
// Call this method when your app is going to shut down not to lose any log messages.
func Flush() {
	pkgOperationsMutex.Lock()
	defer pkgOperationsMutex.Unlock()
	Current.Flush()
}
