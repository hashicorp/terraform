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
	"math"
	"time"
)

var (
	adaptiveLoggerMaxInterval         = time.Minute
	adaptiveLoggerMaxCriticalMsgCount = uint32(1000)
)

// asyncAdaptiveLogger represents asynchronous adaptive logger which acts like
// an async timer logger, but its interval depends on the current message count
// in the queue.
//
// Interval = I, minInterval = m, maxInterval = M, criticalMsgCount = C, msgCount = c:
// I = m + (C - Min(c, C)) / C * (M - m)
type asyncAdaptiveLogger struct {
	asyncLogger
	minInterval      time.Duration
	criticalMsgCount uint32
	maxInterval      time.Duration
}

// NewAsyncLoopLogger creates a new asynchronous adaptive logger
func NewAsyncAdaptiveLogger(
	config *logConfig,
	minInterval time.Duration,
	maxInterval time.Duration,
	criticalMsgCount uint32) (*asyncAdaptiveLogger, error) {

	if minInterval <= 0 {
		return nil, errors.New("async adaptive logger min interval should be > 0")
	}

	if maxInterval > adaptiveLoggerMaxInterval {
		return nil, fmt.Errorf("async adaptive logger max interval should be <= %s",
			adaptiveLoggerMaxInterval)
	}

	if criticalMsgCount <= 0 {
		return nil, errors.New("async adaptive logger critical msg count should be > 0")
	}

	if criticalMsgCount > adaptiveLoggerMaxCriticalMsgCount {
		return nil, fmt.Errorf("async adaptive logger critical msg count should be <= %s",
			adaptiveLoggerMaxInterval)
	}

	asnAdaptiveLogger := new(asyncAdaptiveLogger)

	asnAdaptiveLogger.asyncLogger = *newAsyncLogger(config)
	asnAdaptiveLogger.minInterval = minInterval
	asnAdaptiveLogger.maxInterval = maxInterval
	asnAdaptiveLogger.criticalMsgCount = criticalMsgCount

	go asnAdaptiveLogger.processQueue()

	return asnAdaptiveLogger, nil
}

func (asnAdaptiveLogger *asyncAdaptiveLogger) processItem() (closed bool, itemCount int) {
	asnAdaptiveLogger.queueHasElements.L.Lock()
	defer asnAdaptiveLogger.queueHasElements.L.Unlock()

	for asnAdaptiveLogger.msgQueue.Len() == 0 && !asnAdaptiveLogger.Closed() {
		asnAdaptiveLogger.queueHasElements.Wait()
	}

	if asnAdaptiveLogger.Closed() {
		return true, asnAdaptiveLogger.msgQueue.Len()
	}

	asnAdaptiveLogger.processQueueElement()
	return false, asnAdaptiveLogger.msgQueue.Len() - 1
}

// I = m + (C - Min(c, C)) / C * (M - m) =>
// I = m + cDiff * mDiff,
// 		cDiff = (C - Min(c, C)) / C)
//		mDiff = (M - m)
func (asnAdaptiveLogger *asyncAdaptiveLogger) calcAdaptiveInterval(msgCount int) time.Duration {
	critCountF := float64(asnAdaptiveLogger.criticalMsgCount)
	cDiff := (critCountF - math.Min(float64(msgCount), critCountF)) / critCountF
	mDiff := float64(asnAdaptiveLogger.maxInterval - asnAdaptiveLogger.minInterval)

	return asnAdaptiveLogger.minInterval + time.Duration(cDiff*mDiff)
}

func (asnAdaptiveLogger *asyncAdaptiveLogger) processQueue() {
	for !asnAdaptiveLogger.Closed() {
		closed, itemCount := asnAdaptiveLogger.processItem()

		if closed {
			break
		}

		interval := asnAdaptiveLogger.calcAdaptiveInterval(itemCount)

		<-time.After(interval)
	}
}
