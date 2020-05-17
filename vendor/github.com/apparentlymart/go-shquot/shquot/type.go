package shquot

import (
	"unicode/utf8"
)

// Q is the signature of all command line quoting functions in this package.
// This may be useful for calling applications that select dynamically which
// quoting mechanism to use and store a reference to the appropriate function
// to call later.
//
// cmdline is a slice of string arguments where the first element is
// conventionally the command itself and any remaining elements are arguments
// to that command. This mimics the way command lines are passed to the execve
// function on a Unix (POSIX) system.
//
// The strings in cmdline are assumed to be UTF-8 encoded. If not, the results
// of some functions may be incorrect.
type Q func(cmdline []string) string

// QS is a variant of Q that returns separate arguments for the unquoted
// command name (first element of cmdline, usually verbatim) and quoted
// remaining arguments, for use with intermediaries that require the command
// name to be provided out-of-band.
type QS func(cmdline []string) (cmd, args string)

// QV is the signature of a function that checks if a given cmdline is valid.
// It returns true if the command line meets some validation constraint and
// false otherwise.
//
// Most quoting functions can accept any command line consisting of valid UTF-8
// strings, but some impose other constraints that may cause their result to
// be lossy, as described in each function's own documentation. In such cases,
// a separate function with a "Valid" suffix added allows a caller to check
// whether the given command line meets the constraints.
type QV func(cmdline []string) bool

// AlwaysValid is a placeholder implementation of QV that always returns true.
// This should be used only in callers that generalize over all Q and QV
// implementations to represent situations where the Q function has no
// unusual constraints.
func AlwaysValid(cmdline []string) bool {
	return true
}

// ValidUTF8 checks whether the elements of the given command line are all
// valid UTF-8 strings, returning false if not.
//
// This is just a convenience wrapper for applying utf8.ValidString to each
// string in the slice.
func ValidUTF8(cmdline []string) bool {
	for _, a := range cmdline {
		if !utf8.ValidString(a) {
			return false
		}
	}
	return true
}

// AllValid combines multiple validation functions together to produce a single
// function that returns true only if all of the given checks return true.
// This is useful only for callers that generalize over all Q and QV
// implementations and that need to compose multiple checks in some cases.
func AllValid(checks ...QV) QV {
	return func(cmdline []string) bool {
		for _, check := range checks {
			if !check(cmdline) {
				return false
			}
		}
		return true
	}
}
