package tfconfig

import (
	"fmt"

	legacyhclparser "github.com/hashicorp/hcl/hcl/parser"
	"github.com/hashicorp/hcl/v2"
)

// Diagnostic describes a problem (error or warning) encountered during
// configuration loading.
type Diagnostic struct {
	Severity DiagSeverity `json:"severity"`
	Summary  string       `json:"summary"`
	Detail   string       `json:"detail,omitempty"`

	// Pos is not populated for all diagnostics, but when populated should
	// indicate a particular line that the described problem relates to.
	Pos *SourcePos `json:"pos,omitempty"`
}

// Diagnostics represents a sequence of diagnostics. This is the type that
// should be returned from a function that might generate diagnostics.
type Diagnostics []Diagnostic

// HasErrors returns true if there is at least one Diagnostic of severity
// DiagError in the receiever.
//
// If a function returns a Diagnostics without errors then the result can
// be assumed to be complete within the "best effort" constraints of this
// library. If errors are present then the caller may wish to employ more
// caution in relying on the result.
func (diags Diagnostics) HasErrors() bool {
	for _, diag := range diags {
		if diag.Severity == DiagError {
			return true
		}
	}
	return false
}

func (diags Diagnostics) Error() string {
	switch len(diags) {
	case 0:
		return "no problems"
	case 1:
		return fmt.Sprintf("%s: %s", diags[0].Summary, diags[0].Detail)
	default:
		return fmt.Sprintf("%s: %s (and %d other messages)", diags[0].Summary, diags[0].Detail, len(diags)-1)
	}
}

// Err returns an error representing the receiver if the receiver HasErrors, or
// nil otherwise.
//
// The returned error can be type-asserted back to a Diagnostics if needed.
func (diags Diagnostics) Err() error {
	if diags.HasErrors() {
		return diags
	}
	return nil
}

// DiagSeverity describes the severity of a Diagnostic.
type DiagSeverity rune

// DiagError indicates a problem that prevented proper processing of the
// configuration. In the precense of DiagError diagnostics the result is
// likely to be incomplete.
const DiagError DiagSeverity = 'E'

// DiagWarning indicates a problem that the user may wish to consider but
// that did not prevent proper processing of the configuration.
const DiagWarning DiagSeverity = 'W'

// MarshalJSON is an implementation of encoding/json.Marshaler
func (s DiagSeverity) MarshalJSON() ([]byte, error) {
	switch s {
	case DiagError:
		return []byte(`"error"`), nil
	case DiagWarning:
		return []byte(`"warning"`), nil
	default:
		return []byte(`"invalid"`), nil
	}
}

func diagnosticsHCL(diags hcl.Diagnostics) Diagnostics {
	if len(diags) == 0 {
		return nil
	}
	ret := make(Diagnostics, len(diags))
	for i, diag := range diags {
		ret[i] = Diagnostic{
			Summary: diag.Summary,
			Detail:  diag.Detail,
		}
		switch diag.Severity {
		case hcl.DiagError:
			ret[i].Severity = DiagError
		case hcl.DiagWarning:
			ret[i].Severity = DiagWarning
		}
		if diag.Subject != nil {
			pos := sourcePosHCL(*diag.Subject)
			ret[i].Pos = &pos
		}
	}
	return ret
}

func diagnosticsError(err error) Diagnostics {
	if err == nil {
		return nil
	}

	if posErr, ok := err.(*legacyhclparser.PosError); ok {
		pos := sourcePosLegacyHCL(posErr.Pos, "")
		return Diagnostics{
			Diagnostic{
				Severity: DiagError,
				Summary:  posErr.Err.Error(),
				Pos:      &pos,
			},
		}
	}

	return Diagnostics{
		Diagnostic{
			Severity: DiagError,
			Summary:  err.Error(),
		},
	}
}

func diagnosticsErrorf(format string, args ...interface{}) Diagnostics {
	return diagnosticsError(fmt.Errorf(format, args...))
}
