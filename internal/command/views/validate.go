package views

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The Validate is used for the validate command.
type Validate interface {
	// Results renders the diagnostics returned from a validation walk, and
	// returns a CLI exit code: 0 if there are no errors, 1 otherwise
	Results(diags tfdiags.Diagnostics) int

	// Diagnostics renders early diagnostics, resulting from argument parsing.
	Diagnostics(diags tfdiags.Diagnostics)
}

// NewValidate returns an initialized Validate implementation for the given ViewType.
func NewValidate(vt arguments.ViewType, view *View) Validate {
	switch vt {
	case arguments.ViewJSON:
		return &ValidateJSON{view: view}
	case arguments.ViewHuman:
		return &ValidateHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The ValidateHuman implementation renders diagnostics in a human-readable form,
// along with a success/failure message if Terraform is able to execute the
// validation walk.
type ValidateHuman struct {
	view *View
}

var _ Validate = (*ValidateHuman)(nil)

func (v *ValidateHuman) Results(diags tfdiags.Diagnostics) int {
	columns := v.view.outputColumns()

	if len(diags) == 0 {
		v.view.streams.Println(format.WordWrap(v.view.colorize.Color(validateSuccess), columns))
	} else {
		v.Diagnostics(diags)

		if !diags.HasErrors() {
			v.view.streams.Println(format.WordWrap(v.view.colorize.Color(validateWarnings), columns))
		}
	}

	if diags.HasErrors() {
		return 1
	}
	return 0
}

const validateSuccess = "[green][bold]Success![reset] The configuration is valid.\n"

const validateWarnings = "[green][bold]Success![reset] The configuration is valid, but there were some validation warnings as shown above.\n"

func (v *ValidateHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// The ValidateJSON implementation renders validation results as a JSON object.
// This object includes top-level fields summarizing the result, and an array
// of JSON diagnostic objects.
type ValidateJSON struct {
	view *View
}

var _ Validate = (*ValidateJSON)(nil)

func (v *ValidateJSON) Results(diags tfdiags.Diagnostics) int {
	// FormatVersion represents the version of the json format and will be
	// incremented for any change to this format that requires changes to a
	// consuming parser.
	const FormatVersion = "0.1"

	type Output struct {
		FormatVersion string `json:"format_version"`

		// We include some summary information that is actually redundant
		// with the detailed diagnostics, but avoids the need for callers
		// to re-implement our logic for deciding these.
		Valid        bool                    `json:"valid"`
		ErrorCount   int                     `json:"error_count"`
		WarningCount int                     `json:"warning_count"`
		Diagnostics  []*viewsjson.Diagnostic `json:"diagnostics"`
	}

	output := Output{
		FormatVersion: FormatVersion,
		Valid:         true, // until proven otherwise
	}
	configSources := v.view.configSources()
	for _, diag := range diags {
		output.Diagnostics = append(output.Diagnostics, viewsjson.NewDiagnostic(diag, configSources))

		switch diag.Severity() {
		case tfdiags.Error:
			output.ErrorCount++
			output.Valid = false
		case tfdiags.Warning:
			output.WarningCount++
		}
	}
	if output.Diagnostics == nil {
		// Make sure this always appears as an array in our output, since
		// this is easier to consume for dynamically-typed languages.
		output.Diagnostics = []*viewsjson.Diagnostic{}
	}

	j, err := json.MarshalIndent(&output, "", "  ")
	if err != nil {
		// Should never happen because we fully-control the input here
		panic(err)
	}
	v.view.streams.Println(string(j))

	if diags.HasErrors() {
		return 1
	}
	return 0
}

// Diagnostics should only be called if the validation walk cannot be executed.
// In this case, we choose to render human-readable diagnostic output,
// primarily for backwards compatibility.
func (v *ValidateJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
