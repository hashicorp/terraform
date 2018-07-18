package hcl

import (
	"fmt"
)

// DiagnosticSeverity represents the severity of a diagnostic.
type DiagnosticSeverity int

const (
	// DiagInvalid is the invalid zero value of DiagnosticSeverity
	DiagInvalid DiagnosticSeverity = iota

	// DiagError indicates that the problem reported by a diagnostic prevents
	// further progress in parsing and/or evaluating the subject.
	DiagError

	// DiagWarning indicates that the problem reported by a diagnostic warrants
	// user attention but does not prevent further progress. It is most
	// commonly used for showing deprecation notices.
	DiagWarning
)

// Diagnostic represents information to be presented to a user about an
// error or anomoly in parsing or evaluating configuration.
type Diagnostic struct {
	Severity DiagnosticSeverity

	// Summary and detail contain the English-language description of the
	// problem. Summary is a terse description of the general problem and
	// detail is a more elaborate, often-multi-sentence description of
	// the probem and what might be done to solve it.
	Summary string
	Detail  string
	Subject *Range
	Context *Range
}

// Diagnostics is a list of Diagnostic instances.
type Diagnostics []*Diagnostic

// error implementation, so that diagnostics can be returned via APIs
// that normally deal in vanilla Go errors.
//
// This presents only minimal context about the error, for compatibility
// with usual expectations about how errors will present as strings.
func (d *Diagnostic) Error() string {
	return fmt.Sprintf("%s: %s; %s", d.Subject, d.Summary, d.Detail)
}

// error implementation, so that sets of diagnostics can be returned via
// APIs that normally deal in vanilla Go errors.
func (d Diagnostics) Error() string {
	count := len(d)
	switch {
	case count == 0:
		return "no diagnostics"
	case count == 1:
		return d[0].Error()
	default:
		return fmt.Sprintf("%s, and %d other diagnostic(s)", d[0].Error(), count-1)
	}
}

// Append appends a new error to a Diagnostics and return the whole Diagnostics.
//
// This is provided as a convenience for returning from a function that
// collects and then returns a set of diagnostics:
//
//     return nil, diags.Append(&hcl.Diagnostic{ ... })
//
// Note that this modifies the array underlying the diagnostics slice, so
// must be used carefully within a single codepath. It is incorrect (and rude)
// to extend a diagnostics created by a different subsystem.
func (d Diagnostics) Append(diag *Diagnostic) Diagnostics {
	return append(d, diag)
}

// Extend concatenates the given Diagnostics with the receiver and returns
// the whole new Diagnostics.
//
// This is similar to Append but accepts multiple diagnostics to add. It has
// all the same caveats and constraints.
func (d Diagnostics) Extend(diags Diagnostics) Diagnostics {
	return append(d, diags...)
}

// HasErrors returns true if the receiver contains any diagnostics of
// severity DiagError.
func (d Diagnostics) HasErrors() bool {
	for _, diag := range d {
		if diag.Severity == DiagError {
			return true
		}
	}
	return false
}

func (d Diagnostics) Errs() []error {
	var errs []error
	for _, diag := range d {
		if diag.Severity == DiagError {
			errs = append(errs, diag)
		}
	}

	return errs
}

// A DiagnosticWriter emits diagnostics somehow.
type DiagnosticWriter interface {
	WriteDiagnostic(*Diagnostic) error
	WriteDiagnostics(Diagnostics) error
}
