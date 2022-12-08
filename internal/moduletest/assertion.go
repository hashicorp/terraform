package moduletest

import (
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Assertion is the description of a single test assertion, whether
// successful or unsuccessful.
//
// Deprecated: Will transition to using the check state models directly in
// the future.
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
//
// Deprecated: Will transition to using the check state models directly in
// the future.
type Component struct {
	Assertions map[string]*Assertion
}

// Status is an enumeration of possible outcomes of a test assertion.
//
// Deprecated: This is now just an alias for [checks.Status] and will be
// removed in the future.
type Status = checks.Status

//go:generate go run golang.org/x/tools/cmd/stringer -type=Status assertion.go

const (
	// Pending indicates that the test was registered (during planning)
	// but didn't register an outcome during apply, perhaps due to being
	// blocked by some other upstream failure.
	Pending Status = checks.StatusUnknown

	// Passed indicates that the test condition succeeded.
	Passed Status = checks.StatusPass

	// Failed indicates that the test condition was valid but did not
	// succeed.
	Failed Status = checks.StatusFail

	// Error indicates that the test condition was invalid or that the
	// test report failed in some other way.
	Error Status = checks.StatusError
)
