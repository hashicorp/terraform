// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
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
		return NewValidateJSON(view)
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
	view *JSONStaticView[ValidateOutput]

	// Legacy view for logging warnings in human-readable format
	// The `version` command's JSON output was implemented in a way that still produces
	// human-readable output when diagnostics are logged. This is preserved, as updating
	// it is a breaking change, but this should be amended in a future major version.
	legacyView *View

	formatVersion string
}

var _ Validate = (*ValidateJSON)(nil)

func NewValidateJSON(view *View) *ValidateJSON {
	// FormatVersion represents the version of the json format and will be
	// incremented for any change to this format that requires changes to a
	// consuming parser.
	const formatVersion = "1.0"

	return &ValidateJSON{
		view:          NewJSONStaticView[ValidateOutput](view),
		legacyView:    view,
		formatVersion: formatVersion,
	}
}

type ValidateOutput struct {
	FormatVersion string `json:"format_version"`

	// We include some summary information that is actually redundant
	// with the detailed diagnostics, but avoids the need for callers
	// to re-implement our logic for deciding these.
	Valid        bool                    `json:"valid"`
	ErrorCount   int                     `json:"error_count"`
	WarningCount int                     `json:"warning_count"`
	Diagnostics  []*viewsjson.Diagnostic `json:"diagnostics"`
}

func (v *ValidateJSON) Results(diags tfdiags.Diagnostics) int {
	output := ValidateOutput{
		FormatVersion: v.formatVersion,
		Valid:         true, // until proven otherwise

		// Make sure this always appears as an array in our output, since
		// this is easier to consume for dynamically-typed languages.
		Diagnostics: []*viewsjson.Diagnostic{},
	}

	// Collect counts
	for _, diag := range diags {
		switch diag.Severity() {
		case tfdiags.Error:
			output.ErrorCount++
			output.Valid = false
		case tfdiags.Warning:
			output.WarningCount++
		}
	}

	// Convert diagnostics to JSON-friendly format
	output.Diagnostics = v.view.prepareDiagnostics(diags)

	v.view.Print(output)

	if diags.HasErrors() {
		return 1
	}
	return 0
}

// Diagnostics should only be called if the validation walk cannot be executed.
// In this case, we choose to render human-readable diagnostic output,
// primarily for backwards compatibility.
func (v *ValidateJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.legacyView.Diagnostics(diags)
}
