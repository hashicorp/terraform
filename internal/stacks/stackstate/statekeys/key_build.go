// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statekeys

import (
	"strings"
)

// rawKeyBuilder is a helper for building multi-field keys in the format
// that's expected by [cutKeyField].
//
// The zero value of rawKeyBuilder is ready to use.
type rawKeyBuilder struct {
	b strings.Builder
	w bool
}

// AppendField appends the given string to the key-in-progress as an additional
// field.
//
// The given string must not contain any unquoted commas, because comma is the
// field delimiter. If given an invalid field value this function will panic.
func (b *rawKeyBuilder) AppendField(s string) {
	if keyDelimiterIdx(s) != -1 {
		panic("key field contains the field delimiter")
	}
	if b.w {
		b.b.WriteByte(',')
	}
	b.w = true
	b.b.WriteString(s)
}

// Raw returns the assembled raw key string.
func (b *rawKeyBuilder) Raw() string {
	return b.b.String()
}
