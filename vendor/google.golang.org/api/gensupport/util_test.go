// Copyright 2016 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gensupport

import "io"

// errReader reads out of a buffer until it is empty, then returns the specified error.
type errReader struct {
	buf []byte
	err error
}

func (er *errReader) Read(p []byte) (int, error) {
	if len(er.buf) == 0 {
		if er.err == nil {
			return 0, io.EOF
		}
		return 0, er.err
	}
	n := copy(p, er.buf)
	er.buf = er.buf[n:]
	return n, nil
}
