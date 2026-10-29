// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProvidersSchema is the view used for the `providers schema` command.
type ProvidersSchema interface {
	Schemas(schemas *terraform.Schemas) tfdiags.Diagnostics

	Diagnostics(diags tfdiags.Diagnostics)
}

func NewProvidersSchema(viewType arguments.ViewType, view *View) ProvidersSchema {
	// formatVersion represents the version of the json format and will be
	// incremented for any change to this format that requires changes to a
	// consuming parser.
	formatVersion := "1.0"

	switch viewType {
	case arguments.ViewHuman:
		return &ProvidersSchemaHuman{view: view}
	case arguments.ViewJSON:
		return &ProvidersSchemaJSON{
			formatVersion: formatVersion,
			view:          NewJSONStaticView[ProvidersSchemaOutput](view),
			legacyView:    view,
		}
	default:
		panic("unsupported view type: " + string(viewType))
	}
}

// The human view is only rendered to enable rendering argument-parsing errors
// when the user has failed to supply the -json flag. This is due to Terraform's
// design of human-output as the default experience.
type ProvidersSchemaHuman struct {
	view *View
}

var _ ProvidersSchema = (*ProvidersSchemaHuman)(nil)

// Argument-parsing errors should cause an early exit before the command invokes this.
func (v *ProvidersSchemaHuman) Schemas(schemas *terraform.Schemas) tfdiags.Diagnostics {
	panic("not implemented")
}

func (v *ProvidersSchemaHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// CURRENTLY UNUSED: This view implementation isn't in control of the JSON output format.
// See jsonview package's `newProviders` function. This sets a format_version value
// that is at the top-level of `provider schema`'s output. Therefore it's both describing the
// format of the schema object and the overall command's output structure.
//
// Due to this we cannot add diagnostics to the JSON output, as we cannot increment the
// format_version value easily (i.e. do we include nested format_versions for JSON schemas shared
// between commands? This decision impacts what we do here.).
//
// For an example of current output structure, see:
// internal/command/testdata/providers-schema/basic/output.json
type ProvidersSchemaOutput struct {
	// unused due to issue described above; diagnostics cannot be displayed in JSON format.
}

type ProvidersSchemaJSON struct {
	view *JSONStaticView[ProvidersSchemaOutput]

	legacyView *View

	formatVersion string
}

var _ ProvidersSchema = (*ProvidersSchemaJSON)(nil)

func (v *ProvidersSchemaJSON) Schemas(schemas *terraform.Schemas) tfdiags.Diagnostics {
	if schemas == nil {
		schemas = &terraform.Schemas{}
	}

	jsonSchemas, err := jsonprovider.Marshal(schemas)
	if err != nil {
		diags := tfdiags.Diagnostics{
			tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to render schemas as JSON",
				fmt.Sprintf("Failed to marshal provider schemas to json: %s", err),
			),
		}
		return diags
	}

	v.view.view.streams.Println(string(jsonSchemas))
	return nil
}

// Human-readable diagnostics output for the JSON view for backwards compatibility and also
// because we cannot easily change the JSON output format to include diagnostics.
func (v *ProvidersSchemaJSON) Diagnostics(diags tfdiags.Diagnostics) {
	if len(diags) == 0 {
		return
	}

	v.legacyView.Diagnostics(diags)
}
