package tfdiags

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

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
		case NonFatalError:
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

// ErrWithWarnings is similar to Err except that it will also return a non-nil
// error if the receiver contains only warnings.
//
// In the warnings-only situation, the result is guaranteed to be of dynamic
// type NonFatalError, allowing diagnostics-aware callers to type-assert
// and unwrap it, treating it as non-fatal.
//
// This should be used only in contexts where the caller is able to recognize
// and handle NonFatalError. For normal callers that expect a lack of errors
// to be signaled by nil, use just Diagnostics.Err.
func (diags Diagnostics) ErrWithWarnings() error {
	if len(diags) == 0 {
		return nil
	}
	if diags.HasErrors() {
		return diags.Err()
	}
	return NonFatalError{diags}
}

// NonFatalErr is similar to Err except that it always returns either nil
// (if there are no diagnostics at all) or NonFatalError.
//
// This allows diagnostics to be returned over an error return channel while
// being explicit that the diagnostics should not halt processing.
//
// This should be used only in contexts where the caller is able to recognize
// and handle NonFatalError. For normal callers that expect a lack of errors
// to be signaled by nil, use just Diagnostics.Err.
func (diags Diagnostics) NonFatalErr() error {
	if len(diags) == 0 {
		return nil
	}
	return NonFatalError{diags}
}

// Sort applies an ordering to the diagnostics in the receiver in-place.
//
// The ordering is: warnings before errors, sourceless before sourced,
// short source paths before long source paths, and then ordering by
// position within each file.
//
// Diagnostics that do not differ by any of these sortable characteristics
// will remain in the same relative order after this method returns.
func (diags Diagnostics) Sort() {
	sort.Stable(sortDiagnostics(diags))
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
		desc := diags[0].Description()
		if desc.Detail == "" {
			return desc.Summary
		}
		return fmt.Sprintf("%s: %s", desc.Summary, desc.Detail)
	default:
		var ret bytes.Buffer
		fmt.Fprintf(&ret, "%d problems:\n", len(diags))
		for _, diag := range dae.Diagnostics {
			desc := diag.Description()
			if desc.Detail == "" {
				fmt.Fprintf(&ret, "\n- %s", desc.Summary)
			} else {
				fmt.Fprintf(&ret, "\n- %s: %s", desc.Summary, desc.Detail)
			}
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

// NonFatalError is a special error type, returned by
// Diagnostics.ErrWithWarnings and Diagnostics.NonFatalErr,
// that indicates that the wrapped diagnostics should be treated as non-fatal.
// Callers can conditionally type-assert an error to this type in order to
// detect the non-fatal scenario and handle it in a different way.
type NonFatalError struct {
	Diagnostics
}

func (woe NonFatalError) Error() string {
	diags := woe.Diagnostics
	switch {
	case len(diags) == 0:
		// should never happen, since we don't create this wrapper if
		// there are no diagnostics in the list.
		return "no errors or warnings"
	case len(diags) == 1:
		desc := diags[0].Description()
		if desc.Detail == "" {
			return desc.Summary
		}
		return fmt.Sprintf("%s: %s", desc.Summary, desc.Detail)
	default:
		var ret bytes.Buffer
		if diags.HasErrors() {
			fmt.Fprintf(&ret, "%d problems:\n", len(diags))
		} else {
			fmt.Fprintf(&ret, "%d warnings:\n", len(diags))
		}
		for _, diag := range woe.Diagnostics {
			desc := diag.Description()
			if desc.Detail == "" {
				fmt.Fprintf(&ret, "\n- %s", desc.Summary)
			} else {
				fmt.Fprintf(&ret, "\n- %s: %s", desc.Summary, desc.Detail)
			}
		}
		return ret.String()
	}
}

// sortDiagnostics is an implementation of sort.Interface
type sortDiagnostics []Diagnostic

var _ sort.Interface = sortDiagnostics(nil)

func (sd sortDiagnostics) Len() int {
	return len(sd)
}

func (sd sortDiagnostics) Less(i, j int) bool {
	iD, jD := sd[i], sd[j]
	iSev, jSev := iD.Severity(), jD.Severity()
	iSrc, jSrc := iD.Source(), jD.Source()

	switch {

	case iSev != jSev:
		return iSev == Warning

	case (iSrc.Subject == nil) != (jSrc.Subject == nil):
		return iSrc.Subject == nil

	case iSrc.Subject != nil && *iSrc.Subject != *jSrc.Subject:
		iSubj := iSrc.Subject
		jSubj := jSrc.Subject
		switch {
		case iSubj.Filename != jSubj.Filename:
			// Path with fewer segments goes first if they are different lengths
			sep := string(filepath.Separator)
			iCount := strings.Count(iSubj.Filename, sep)
			jCount := strings.Count(jSubj.Filename, sep)
			if iCount != jCount {
				return iCount < jCount
			}
			return iSubj.Filename < jSubj.Filename
		case iSubj.Start.Byte != jSubj.Start.Byte:
			return iSubj.Start.Byte < jSubj.Start.Byte
		case iSubj.End.Byte != jSubj.End.Byte:
			return iSubj.End.Byte < jSubj.End.Byte
		}
		fallthrough

	default:
		// The remaining properties do not have a defined ordering, so
		// we'll leave it unspecified. Since we use sort.Stable in
		// the caller of this, the ordering of remaining items will
		// be preserved.
		return false
	}
}

func (sd sortDiagnostics) Swap(i, j int) {
	sd[i], sd[j] = sd[j], sd[i]
}
