// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfdiags

// This "Extra" idea is something we've inherited from HCL's diagnostic model,
// and so it's primarily to expose that functionality from wrapped HCL
// diagnostics but other diagnostic types could potentially implement this
// protocol too, if needed.

// ExtraInfo tries to retrieve extra information of interface type T from
// the given diagnostic.
//
// "Extra information" is situation-specific additional contextual data which
// might allow for some special tailored reporting of particular
// diagnostics in the UI. Conventionally the extra information is provided
// as a hidden type that implements one or more interfaces which a caller
// can pass as type parameter T to retrieve a value of that type when the
// diagnostic has such an implementation.
//
// If the given diagnostic's extra value has an implementation of interface T
// then ExtraInfo returns a non-nil interface value. If there is no such
// implementation, ExtraInfo returns a nil T.
//
// Although the signature of this function does not constrain T to be an
// interface type, our convention is to only use interface types to access
// extra info in order to allow for alternative or wrapping implementations
// of the interface.
func ExtraInfo[T any](diag Diagnostic) T {
	extra := diag.ExtraInfo()
	if ret, ok := extra.(T); ok {
		return ret
	}

	// If "extra" doesn't implement T directly then we'll delegate to
	// our ExtraInfoNext helper to try iteratively unwrapping it.
	return ExtraInfoNext[T](extra)
}

// ExtraInfoNext takes a value previously returned by ExtraInfo and attempts
// to find an implementation of interface T wrapped inside of it. The return
// value meaning is the same as for ExtraInfo.
//
// This is to help with the less common situation where a particular "extra"
// value might be wrapping another value implementing the same interface,
// and so callers can peel away one layer at a time until there are no more
// nested layers.
//
// Because this function is intended for searching for _nested_ implementations
// of T, ExtraInfoNext does not consider whether value "previous" directly
// implements interface T, on the assumption that the previous call to ExtraInfo
// with the same T caused "previous" to already be that result.
func ExtraInfoNext[T any](previous interface{}) T {
	// As long as T is an interface type as documented, zero will always be
	// a nil interface value for us to return in the non-matching case.
	var zero T

	unwrapper, ok := previous.(DiagnosticExtraUnwrapper)
	// If the given value isn't unwrappable then it can't possibly have
	// any other info nested inside of it.
	if !ok {
		return zero
	}

	extra := unwrapper.UnwrapDiagnosticExtra()

	// We'll keep unwrapping until we either find the interface we're
	// looking for or we run out of layers of unwrapper.
	for {
		if ret, ok := extra.(T); ok {
			return ret
		}

		if unwrapper, ok := extra.(DiagnosticExtraUnwrapper); ok {
			extra = unwrapper.UnwrapDiagnosticExtra()
		} else {
			return zero
		}
	}
}

// DiagnosticExtraUnwrapper is an interface implemented by values in the
// Extra field of Diagnostic when they are wrapping another "Extra" value that
// was generated downstream.
//
// Diagnostic recipients which want to examine "Extra" values to sniff for
// particular types of extra data can either type-assert this interface
// directly and repeatedly unwrap until they recieve nil, or can use the
// helper function DiagnosticExtra.
//
// This interface intentionally matches hcl.DiagnosticExtraUnwrapper, so that
// wrapping extra values implemented using HCL's API will also work with the
// tfdiags API, but that non-HCL uses of this will not need to implement HCL
// just to get this interface.
type DiagnosticExtraUnwrapper interface {
	// If the reciever is wrapping another "diagnostic extra" value, returns
	// that value. Otherwise returns nil to indicate dynamically that nothing
	// is wrapped.
	//
	// The "nothing is wrapped" condition can be signalled either by this
	// method returning nil or by a type not implementing this interface at all.
	//
	// Implementers should never create unwrap "cycles" where a nested extra
	// value returns a value that was also wrapping it.
	UnwrapDiagnosticExtra() interface{}
}

// DiagnosticExtraWrapper is an interface implemented by values that can be
// dynamically updated to wrap other extra info.
type DiagnosticExtraWrapper interface {
	// WrapDiagnosticExtra accepts an ExtraInfo that it should add within the
	// current ExtraInfo.
	WrapDiagnosticExtra(inner interface{})
}

// DiagnosticExtraBecauseUnknown is an interface implemented by values in
// the Extra field of Diagnostic when the diagnostic is potentially caused by
// the presence of unknown values in an expression evaluation.
//
// Just implementing this interface is not sufficient signal, though. Callers
// must also call the DiagnosticCausedByUnknown method in order to confirm
// the result, or use the package-level function DiagnosticCausedByUnknown
// as a convenient wrapper.
type DiagnosticExtraBecauseUnknown interface {
	// DiagnosticCausedByUnknown returns true if the associated diagnostic
	// was caused by the presence of unknown values during an expression
	// evaluation, or false otherwise.
	//
	// Callers might use this to tailor what contextual information they show
	// alongside an error report in the UI, to avoid potential confusion
	// caused by talking about the presence of unknown values if that was
	// immaterial to the error.
	DiagnosticCausedByUnknown() bool
}

// DiagnosticCausedByUnknown returns true if the given diagnostic has an
// indication that it was caused by the presence of unknown values during
// an expression evaluation.
//
// This is a wrapper around checking if the diagnostic's extra info implements
// interface DiagnosticExtraBecauseUnknown and then calling its method if so.
func DiagnosticCausedByUnknown(diag Diagnostic) bool {
	maybe := ExtraInfo[DiagnosticExtraBecauseUnknown](diag)
	if maybe == nil {
		return false
	}
	return maybe.DiagnosticCausedByUnknown()
}

// DiagnosticExtraBecauseSensitive is an interface implemented by values in
// the Extra field of Diagnostic when the diagnostic is potentially caused by
// the presence of sensitive values in an expression evaluation.
//
// Just implementing this interface is not sufficient signal, though. Callers
// must also call the DiagnosticCausedBySensitive method in order to confirm
// the result, or use the package-level function DiagnosticCausedBySensitive
// as a convenient wrapper.
type DiagnosticExtraBecauseSensitive interface {
	// DiagnosticCausedBySensitive returns true if the associated diagnostic
	// was caused by the presence of sensitive values during an expression
	// evaluation, or false otherwise.
	//
	// Callers might use this to tailor what contextual information they show
	// alongside an error report in the UI, to avoid potential confusion
	// caused by talking about the presence of sensitive values if that was
	// immaterial to the error.
	DiagnosticCausedBySensitive() bool
}

// DiagnosticCausedBySensitive returns true if the given diagnostic has an
// indication that it was caused by the presence of sensitive values during
// an expression evaluation.
//
// This is a wrapper around checking if the diagnostic's extra info implements
// interface DiagnosticExtraBecauseSensitive and then calling its method if so.
func DiagnosticCausedBySensitive(diag Diagnostic) bool {
	maybe := ExtraInfo[DiagnosticExtraBecauseSensitive](diag)
	if maybe == nil {
		return false
	}
	return maybe.DiagnosticCausedBySensitive()
}

// DiagnosticExtraDoNotConsolidate tells the Diagnostics.ConsolidateWarnings
// function not to consolidate this diagnostic if it otherwise would.
type DiagnosticExtraDoNotConsolidate interface {
	// DoNotConsolidateDiagnostic returns true if the associated diagnostic
	// should not be consolidated by the Diagnostics.ConsolidateWarnings
	// function.
	DoNotConsolidateDiagnostic() bool
}

// DoNotConsolidateDiagnostic returns true if the given diagnostic should not
// be consolidated by the Diagnostics.ConsolidateWarnings function.
func DoNotConsolidateDiagnostic(diag Diagnostic) bool {
	maybe := ExtraInfo[DiagnosticExtraDoNotConsolidate](diag)
	if maybe == nil {
		return false
	}
	return maybe.DoNotConsolidateDiagnostic()
}
