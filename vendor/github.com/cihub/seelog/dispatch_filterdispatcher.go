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

// A filterDispatcher writes the given message to underlying receivers only if message log level
// is in the allowed list.
type filterDispatcher struct {
	*dispatcher
	allowList map[LogLevel]bool
}

// NewFilterDispatcher creates a new filterDispatcher using a list of allowed levels.
func NewFilterDispatcher(formatter *formatter, receivers []interface{}, allowList ...LogLevel) (*filterDispatcher, error) {
	disp, err := createDispatcher(formatter, receivers)
	if err != nil {
		return nil, err
	}

	allows := make(map[LogLevel]bool)
	for _, allowLevel := range allowList {
		allows[allowLevel] = true
	}

	return &filterDispatcher{disp, allows}, nil
}

func (filter *filterDispatcher) Dispatch(
	message string,
	level LogLevel,
	context LogContextInterface,
	errorFunc func(err error)) {
	isAllowed, ok := filter.allowList[level]
	if ok && isAllowed {
		filter.dispatcher.Dispatch(message, level, context, errorFunc)
	}
}

func (filter *filterDispatcher) String() string {
	return fmt.Sprintf("filterDispatcher ->\n%s", filter.dispatcher)
}
