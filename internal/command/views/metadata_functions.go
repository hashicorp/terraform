// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonfunction"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty/function"
)

func NewMetadataFunctions(viewType arguments.ViewType, view *View) MetadataFunctions {
	// FormatVersion represents the version of the json format and will be
	// incremented for any change to this format that requires changes to a
	// consuming parser.
	const formatVersion = jsonfunction.FormatVersion

	switch viewType {
	case arguments.ViewHuman:
		return &MetadataFunctionsHuman{
			view: view,
		}
	case arguments.ViewJSON:
		return &MetadataFunctionsJSON{
			view: &JSONStaticView[MetadataFunctionsOutput]{
				view: view,
			},
			legacyView:    view,
			formatVersion: formatVersion,
		}
	default:
		panic("unsupported view type")
	}
}

type MetadataFunctions interface {
	Functions(funcs map[string]function.Function)
	Diagnostics(diags tfdiags.Diagnostics)
}

type MetadataFunctionsHuman struct {
	view *View
}

func (v *MetadataFunctionsHuman) Functions(_ map[string]function.Function) {
	panic("MetadataFunctionsHuman.Functions: the `metadata functions` command should not be producing human output for happy path")
}

func (v *MetadataFunctionsHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

type MetadataFunctionsJSON struct {
	view          *JSONStaticView[MetadataFunctionsOutput]
	legacyView    *View
	formatVersion string
}

// Functions is the top-level object returned when exporting function signatures
type MetadataFunctionsOutput struct {
	FormatVersion string                                     `json:"format_version"`
	Signatures    map[string]*jsonfunction.FunctionSignature `json:"function_signatures,omitempty"`
}

func (v *MetadataFunctionsJSON) Functions(funcs map[string]function.Function) {
	// We don't use MetadataFunctionsOutput or other data in the views package to prepare
	// output for now, as this is implemented in the jsonfunction package.
	//
	// TODO: Make views package responsible for the output format and versioning.

	output, diags := jsonfunction.Marshal(funcs)
	if diags.HasErrors() {
		// Shouldn't happen because we control the input
		panic(fmt.Sprintf("MetadataFunctionsJSON.Functions: failed to marshal functions: %s", diags.Err()))
	}

	v.view.view.streams.Print(string(output))
}

func (v *MetadataFunctionsJSON) Diagnostics(diags tfdiags.Diagnostics) {
	// Human-readable output is produced despite this being the JSON view.
	v.legacyView.Diagnostics(diags)
}
