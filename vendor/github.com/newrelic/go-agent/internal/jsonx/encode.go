// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package jsonx extends the encoding/json package to encode JSON
// incrementally and without requiring reflection.
package jsonx

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"unicode/utf8"
)

var hex = "0123456789abcdef"

// AppendString escapes s appends it to buf.
func AppendString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if 0x20 <= b && b != '\\' && b != '"' && b != '<' && b != '>' && b != '&' {
				i++
				continue
			}
			if start < i {
				buf.WriteString(s[start:i])
			}
			switch b {
			case '\\', '"':
				buf.WriteByte('\\')
				buf.WriteByte(b)
			case '\n':
				buf.WriteByte('\\')
				buf.WriteByte('n')
			case '\r':
				buf.WriteByte('\\')
				buf.WriteByte('r')
			case '\t':
				buf.WriteByte('\\')
				buf.WriteByte('t')
			default:
				// This encodes bytes < 0x20 except for \n and \r,
				// as well as <, > and &. The latter are escaped because they
				// can lead to security holes when user-controlled strings
				// are rendered into JSON and served to some browsers.
				buf.WriteString(`\u00`)
				buf.WriteByte(hex[b>>4])
				buf.WriteByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteString(`\ufffd`)
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteString(`\u202`)
			buf.WriteByte(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		buf.WriteString(s[start:])
	}
	buf.WriteByte('"')
}

// AppendStringArray appends an array of string literals to buf.
func AppendStringArray(buf *bytes.Buffer, a ...string) {
	buf.WriteByte('[')
	for i, s := range a {
		if i > 0 {
			buf.WriteByte(',')
		}
		AppendString(buf, s)
	}
	buf.WriteByte(']')
}

// AppendFloat appends a numeric literal representing the value to buf.
func AppendFloat(buf *bytes.Buffer, x float64) error {
	var scratch [64]byte

	if math.IsInf(x, 0) || math.IsNaN(x) {
		return &json.UnsupportedValueError{
			Value: reflect.ValueOf(x),
			Str:   strconv.FormatFloat(x, 'g', -1, 64),
		}
	}

	buf.Write(strconv.AppendFloat(scratch[:0], x, 'g', -1, 64))
	return nil
}

// AppendFloatArray appends an array of numeric literals to buf.
func AppendFloatArray(buf *bytes.Buffer, a ...float64) error {
	buf.WriteByte('[')
	for i, x := range a {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := AppendFloat(buf, x); err != nil {
			return err
		}
	}
	buf.WriteByte(']')
	return nil
}

// AppendInt appends a numeric literal representing the value to buf.
func AppendInt(buf *bytes.Buffer, x int64) {
	var scratch [64]byte
	buf.Write(strconv.AppendInt(scratch[:0], x, 10))
}

// AppendIntArray appends an array of numeric literals to buf.
func AppendIntArray(buf *bytes.Buffer, a ...int64) {
	var scratch [64]byte

	buf.WriteByte('[')
	for i, x := range a {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(strconv.AppendInt(scratch[:0], x, 10))
	}
	buf.WriteByte(']')
}

// AppendUint appends a numeric literal representing the value to buf.
func AppendUint(buf *bytes.Buffer, x uint64) {
	var scratch [64]byte
	buf.Write(strconv.AppendUint(scratch[:0], x, 10))
}

// AppendUintArray appends an array of numeric literals to buf.
func AppendUintArray(buf *bytes.Buffer, a ...uint64) {
	var scratch [64]byte

	buf.WriteByte('[')
	for i, x := range a {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(strconv.AppendUint(scratch[:0], x, 10))
	}
	buf.WriteByte(']')
}
