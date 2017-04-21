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
	"fmt"
)

// syncLogger performs logging in the same goroutine where 'Trace/Debug/...'
// func was called
type syncLogger struct {
	commonLogger
}

// NewSyncLogger creates a new synchronous logger
func NewSyncLogger(config *logConfig) *syncLogger {
	syncLogger := new(syncLogger)

	syncLogger.commonLogger = *newCommonLogger(config, syncLogger)

	return syncLogger
}

func (syncLogger *syncLogger) innerLog(
	level LogLevel,
	context LogContextInterface,
	message fmt.Stringer) {

	syncLogger.processLogMsg(level, message, context)
}

func (syncLogger *syncLogger) Close() {
	syncLogger.m.Lock()
	defer syncLogger.m.Unlock()

	if !syncLogger.Closed() {
		if err := syncLogger.config.RootDispatcher.Close(); err != nil {
			reportInternalError(err)
		}
		syncLogger.closedM.Lock()
		syncLogger.closed = true
		syncLogger.closedM.Unlock()
	}
}

func (syncLogger *syncLogger) Flush() {
	syncLogger.m.Lock()
	defer syncLogger.m.Unlock()

	if !syncLogger.Closed() {
		syncLogger.config.RootDispatcher.Flush()
	}
}
