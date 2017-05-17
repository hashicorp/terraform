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
	"io"
)

// A dispatcherInterface is used to dispatch message to all underlying receivers.
// Dispatch logic depends on given context and log level. Any errors are reported using errorFunc.
// Also, as underlying receivers may have a state, dispatcher has a ShuttingDown method which performs
// an immediate cleanup of all data that is stored in the receivers
type dispatcherInterface interface {
	flusherInterface
	io.Closer
	Dispatch(message string, level LogLevel, context LogContextInterface, errorFunc func(err error))
}

type dispatcher struct {
	formatter   *formatter
	writers     []*formattedWriter
	dispatchers []dispatcherInterface
}

// Creates a dispatcher which dispatches data to a list of receivers.
// Each receiver should be either a Dispatcher or io.Writer, otherwise an error will be returned
func createDispatcher(formatter *formatter, receivers []interface{}) (*dispatcher, error) {
	if formatter == nil {
		return nil, errors.New("formatter cannot be nil")
	}
	if receivers == nil || len(receivers) == 0 {
		return nil, errors.New("receivers cannot be nil or empty")
	}

	disp := &dispatcher{formatter, make([]*formattedWriter, 0), make([]dispatcherInterface, 0)}
	for _, receiver := range receivers {
		writer, ok := receiver.(*formattedWriter)
		if ok {
			disp.writers = append(disp.writers, writer)
			continue
		}

		ioWriter, ok := receiver.(io.Writer)
		if ok {
			writer, err := NewFormattedWriter(ioWriter, disp.formatter)
			if err != nil {
				return nil, err
			}
			disp.writers = append(disp.writers, writer)
			continue
		}

		dispInterface, ok := receiver.(dispatcherInterface)
		if ok {
			disp.dispatchers = append(disp.dispatchers, dispInterface)
			continue
		}

		return nil, errors.New("method can receive either io.Writer or dispatcherInterface")
	}

	return disp, nil
}

func (disp *dispatcher) Dispatch(
	message string,
	level LogLevel,
	context LogContextInterface,
	errorFunc func(err error)) {

	for _, writer := range disp.writers {
		err := writer.Write(message, level, context)
		if err != nil {
			errorFunc(err)
		}
	}

	for _, dispInterface := range disp.dispatchers {
		dispInterface.Dispatch(message, level, context, errorFunc)
	}
}

// Flush goes through all underlying writers which implement flusherInterface interface
// and closes them. Recursively performs the same action for underlying dispatchers
func (disp *dispatcher) Flush() {
	for _, disp := range disp.Dispatchers() {
		disp.Flush()
	}

	for _, formatWriter := range disp.Writers() {
		flusher, ok := formatWriter.Writer().(flusherInterface)
		if ok {
			flusher.Flush()
		}
	}
}

// Close goes through all underlying writers which implement io.Closer interface
// and closes them. Recursively performs the same action for underlying dispatchers
// Before closing, writers are flushed to prevent loss of any buffered data, so
// a call to Flush() func before Close() is not necessary
func (disp *dispatcher) Close() error {
	for _, disp := range disp.Dispatchers() {
		disp.Flush()
		err := disp.Close()
		if err != nil {
			return err
		}
	}

	for _, formatWriter := range disp.Writers() {
		flusher, ok := formatWriter.Writer().(flusherInterface)
		if ok {
			flusher.Flush()
		}

		closer, ok := formatWriter.Writer().(io.Closer)
		if ok {
			err := closer.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (disp *dispatcher) Writers() []*formattedWriter {
	return disp.writers
}

func (disp *dispatcher) Dispatchers() []dispatcherInterface {
	return disp.dispatchers
}

func (disp *dispatcher) String() string {
	str := "formatter: " + disp.formatter.String() + "\n"

	str += "    ->Dispatchers:"

	if len(disp.dispatchers) == 0 {
		str += "none\n"
	} else {
		str += "\n"

		for _, disp := range disp.dispatchers {
			str += fmt.Sprintf("        ->%s", disp)
		}
	}

	str += "    ->Writers:"

	if len(disp.writers) == 0 {
		str += "none\n"
	} else {
		str += "\n"

		for _, writer := range disp.writers {
			str += fmt.Sprintf("        ->%s\n", writer)
		}
	}

	return str
}
