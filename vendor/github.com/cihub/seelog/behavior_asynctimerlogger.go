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
	"time"
)

// asyncTimerLogger represents asynchronous logger which processes the log queue each
// 'duration' nanoseconds
type asyncTimerLogger struct {
	asyncLogger
	interval time.Duration
}

// NewAsyncLoopLogger creates a new asynchronous loop logger
func NewAsyncTimerLogger(config *logConfig, interval time.Duration) (*asyncTimerLogger, error) {

	if interval <= 0 {
		return nil, errors.New("async logger interval should be > 0")
	}

	asnTimerLogger := new(asyncTimerLogger)

	asnTimerLogger.asyncLogger = *newAsyncLogger(config)
	asnTimerLogger.interval = interval

	go asnTimerLogger.processQueue()

	return asnTimerLogger, nil
}

func (asnTimerLogger *asyncTimerLogger) processItem() (closed bool) {
	asnTimerLogger.queueHasElements.L.Lock()
	defer asnTimerLogger.queueHasElements.L.Unlock()

	for asnTimerLogger.msgQueue.Len() == 0 && !asnTimerLogger.Closed() {
		asnTimerLogger.queueHasElements.Wait()
	}

	if asnTimerLogger.Closed() {
		return true
	}

	asnTimerLogger.processQueueElement()
	return false
}

func (asnTimerLogger *asyncTimerLogger) processQueue() {
	for !asnTimerLogger.Closed() {
		closed := asnTimerLogger.processItem()

		if closed {
			break
		}

		<-time.After(asnTimerLogger.interval)
	}
}
