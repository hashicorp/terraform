package tfdiags

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/errwrap"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl2/hcl"
)

// Diagnostics is a list of diagnostics. Diagnostics is intended to be used
// where a Go "error" might normally be used, allowing richer information
// to be conveyed (more context, support for warnings).
//
// A nil Diagnostics is a valid, empty diagnostics list, thus allowing
// heap allocation to be avoided in the common case where there are no
// diagnostics to report at all.
type Diagnostics []Diagnostic

// Append is the main interface for constructing Diagnostics lists, taking
// an existing list (which may be nil) and appending the new objects to it
// after normalizing them to be implementations of Diagnostic.
//
// The usual pattern for a function that natively "speaks" diagnostics is:
//
//     // Create a nil Diagnostics at the start of the function
//     var diags diag.Diagnostics
//
//     // At later points, build on it if errors / warnings occur:
//     foo, err := DoSomethingRisky()
//     if err != nil {
//         diags = diags.Append(err)
//     }
//
//     // Eventually return the result and diagnostics in place of error
//     return result, diags
//
// Append accepts a variety of different diagnostic-like types, including
// native Go errors and HCL diagnostics. It also knows how to unwrap
// a multierror.Error into separate error diagnostics. It can be passed
// another Diagnostics to concatenate the two lists. If given something
// it cannot handle, this function will panic.
func (diags Diagnostics) Append(new ...interface{}) Diagnostics {
	for _, item := range new {
		if item == nil {
			continue
		}

		switch ti := item.(type) {
		case Diagnostic:
			diags = append(diags, ti)
		case Diagnostics:
			diags = append(diags, ti...) // flatten
		case diagnosticsAsError:
			diags = diags.Append(ti.Diagnostics) // unwrap
		case hcl.Diagnostics:
			for _, hclDiag := range ti {
				diags = append(diags, hclDiagnostic{hclDiag})
			}
		case *hcl.Diagnostic:
			diags = append(diags, hclDiagnostic{ti})
		case *multierror.Error:
			for _, err := range ti.Errors {
				diags = append(diags, nativeError{err})
			}
		case error:
			switch {
			case errwrap.ContainsType(ti, Diagnostics(nil)):
				// If we have an errwrap wrapper with a Diagnostics hiding
				// inside then we'll unpick it here to get access to the
				// individual diagnostics.
				diags = diags.Append(errwrap.GetType(ti, Diagnostics(nil)))
			case errwrap.ContainsType(ti, hcl.Diagnostics(nil)):
				// Likewise, if we have HCL diagnostics we'll unpick that too.
				diags = diags.Append(errwrap.GetType(ti, hcl.Diagnostics(nil)))
			default:
				diags = append(diags, nativeError{ti})
			}
		default:
			panic(fmt.Errorf("can't construct diagnostic(s) from %T", item))
		}
	}

	// Given the above, we should never end up with a non-nil empty slice
	// here, but we'll make sure of that so callers can rely on empty == nil
	if len(diags) == 0 {
		return nil
	}

	return diags
}

// HasErrors returns true if any of the diagnostics in the list have
// a severity of Error.
func (diags Diagnostics) HasErrors() bool {
	for _, diag := range diags {
		if diag.Severity() == Error {
			return true
		}
	}
	return false
}

// ForRPC returns a version of the receiver that has been simplified so that
// it is friendly to RPC protocols.
//
// Currently this means that it can be serialized with encoding/gob and
// subsequently re-inflated. It may later grow to include other serialization
// formats.
//
// Note that this loses information about the original objects used to
// construct the diagnostics, so e.g. the errwrap API will not work as
// expected on an error-wrapped Diagnostics that came from ForRPC.
func (diags Diagnostics) ForRPC() Diagnostics {
	ret := make(Diagnostics, len(diags))
	for i := range diags {
		ret[i] = makeRPCFriendlyDiag(diags[i])
	}
	return ret
}

// Err flattens a diagnostics list into a single Go error, or to nil
// if the diagnostics list does not include any error-level diagnostics.
//
// This can be used to smuggle diagnostics through an API that deals in
// native errors, but unfortunately it will lose naked warnings (warnings
// that aren't accompanied by at least one error) since such APIs have no
// mechanism through which to report these.
//
//     return result, diags.Error()
func (diags Diagnostics) Err() error {
	if !diags.HasErrors() {
		return nil
	}
	return diagnosticsAsError{diags}
}

type diagnosticsAsError struct {
	Diagnostics
}

func (dae diagnosticsAsError) Error() string {
	diags := dae.Diagnostics
	switch {
	case len(diags) == 0:
		// should never happen, since we don't create this wrapper if
		// there are no diagnostics in the list.
		return "no errors"
	case len(diags) == 1:
		return diags[0].Description().Summary
	default:
		var ret bytes.Buffer
		fmt.Fprintf(&ret, "%d problems:\n", len(diags))
		for _, diag := range dae.Diagnostics {
			fmt.Fprintf(&ret, "\n- %s", diag.Description().Summary)
		}
		return ret.String()
	}
}

// WrappedErrors is an implementation of errwrap.Wrapper so that an error-wrapped
// diagnostics object can be picked apart by errwrap-aware code.
func (dae diagnosticsAsError) WrappedErrors() []error {
	var errs []error
	for _, diag := range dae.Diagnostics {
		if wrapper, isErr := diag.(nativeError); isErr {
			errs = append(errs, wrapper.err)
		}
	}
	return errs
}
