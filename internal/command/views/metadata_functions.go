package views

import (
	"encoding/json"
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
	const formatVersion = "1.0"

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
	output := &MetadataFunctionsOutput{
		FormatVersion: v.formatVersion,
		Signatures:    map[string]*jsonfunction.FunctionSignature{},
	}

	signatures, diags := jsonfunction.MarshalToFunctionSignatures(funcs)
	if diags.HasErrors() {
		// Shouldn't happen because we control the input
		panic(fmt.Sprintf("MetadataFunctionsJSON.Functions: failed to marshal functions: %s", diags.Err()))
	}
	output.Signatures = signatures

	// The marshalled JSON should not include any newlines or whitespace.
	b, err := json.Marshal(output)
	if err != nil {
		// Should never happen because we fully-control the input here
		panic(err)
	}
	v.view.view.streams.Print(string(b))
}

func (v *MetadataFunctionsJSON) Diagnostics(diags tfdiags.Diagnostics) {
	// Human-readable output is produced despite this being the JSON view.
	v.legacyView.Diagnostics(diags)
}
