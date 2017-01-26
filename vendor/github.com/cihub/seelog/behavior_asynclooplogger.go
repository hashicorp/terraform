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

// asyncLoopLogger represents asynchronous logger which processes the log queue in
// a 'for' loop
type asyncLoopLogger struct {
	asyncLogger
}

// NewAsyncLoopLogger creates a new asynchronous loop logger
func NewAsyncLoopLogger(config *logConfig) *asyncLoopLogger {

	asnLoopLogger := new(asyncLoopLogger)

	asnLoopLogger.asyncLogger = *newAsyncLogger(config)

	go asnLoopLogger.processQueue()

	return asnLoopLogger
}

func (asnLoopLogger *asyncLoopLogger) processItem() (closed bool) {
	asnLoopLogger.queueHasElements.L.Lock()
	defer asnLoopLogger.queueHasElements.L.Unlock()

	for asnLoopLogger.msgQueue.Len() == 0 && !asnLoopLogger.Closed() {
		asnLoopLogger.queueHasElements.Wait()
	}

	if asnLoopLogger.Closed() {
		return true
	}

	asnLoopLogger.processQueueElement()
	return false
}

func (asnLoopLogger *asyncLoopLogger) processQueue() {
	for !asnLoopLogger.Closed() {
		closed := asnLoopLogger.processItem()

		if closed {
			break
		}
	}
}
