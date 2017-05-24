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

type formattedWriter struct {
	writer    io.Writer
	formatter *formatter
}

func NewFormattedWriter(writer io.Writer, formatter *formatter) (*formattedWriter, error) {
	if formatter == nil {
		return nil, errors.New("formatter can not be nil")
	}

	return &formattedWriter{writer, formatter}, nil
}

func (formattedWriter *formattedWriter) Write(message string, level LogLevel, context LogContextInterface) error {
	str := formattedWriter.formatter.Format(message, level, context)
	_, err := formattedWriter.writer.Write([]byte(str))
	return err
}

func (formattedWriter *formattedWriter) String() string {
	return fmt.Sprintf("writer: %s, format: %s", formattedWriter.writer, formattedWriter.formatter)
}

func (formattedWriter *formattedWriter) Writer() io.Writer {
	return formattedWriter.writer
}

func (formattedWriter *formattedWriter) Format() *formatter {
	return formattedWriter.formatter
}
