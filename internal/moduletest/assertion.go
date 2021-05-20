package moduletest

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Assertion is the description of a single test assertion, whether
// successful or unsuccessful.
type Assertion struct {
	Outcome Status

	// Description is a user-provided, human-readable description of what
	// this assertion represents.
	Description string

	// Message is typically relevant only for TestFailed or TestError
	// assertions, giving a human-readable description of the problem,
	// formatted in the way our format package expects to receive paragraphs
	// for terminal word wrapping.
	Message string

	// Diagnostics includes diagnostics specific to the current test assertion,
	// if available.
	Diagnostics tfdiags.Diagnostics
}

// Component represents a component being tested, each of which can have
// several associated test assertions.
type Component struct {
	Assertions map[string]*Assertion
}

// Status is an enumeration of possible outcomes of a test assertion.
type Status rune

//go:generate go run golang.org/x/tools/cmd/stringer -type=Status assertion.go

const (
	// Pending indicates that the test was registered (during planning)
	// but didn't register an outcome during apply, perhaps due to being
	// blocked by some other upstream failure.
	Pending Status = '?'

	// Passed indicates that the test condition succeeded.
	Passed Status = 'P'

	// Failed indicates that the test condition was valid but did not
	// succeed.
	Failed Status = 'F'

	// Error indicates that the test condition was invalid or that the
	// test report failed in some other way.
	Error Status = 'E'
)

// SuiteCanPass returns true if a suite containing an assertion with this
// status could possibly succeed. The suite as a whole succeeds only if all
// of its assertions have statuses where SuiteCanPass returns true.
func (s Status) SuiteCanPass() bool {
	switch s {
	case Failed, Error:
		return false
	default:
		return true
	}
}
