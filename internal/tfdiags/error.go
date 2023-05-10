// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfdiags

// nativeError is a Diagnostic implementation that wraps a normal Go error
type nativeError struct {
	err error
}

var _ Diagnostic = nativeError{}

func (e nativeError) Severity() Severity {
	return Error
}

func (e nativeError) Description() Description {
	return Description{
		Summary: FormatError(e.err),
	}
}

func (e nativeError) Source() Source {
	// No source information available for a native error
	return Source{}
}

func (e nativeError) FromExpr() *FromExpr {
	// Native errors are not expression-related
	return nil
}

func (e nativeError) ExtraInfo() interface{} {
	// Native errors don't carry any "extra information".
	return nil
}
