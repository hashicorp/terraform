// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/repl"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The Output view renders either one or all outputs, depending on whether or
// not the name argument is empty.
type Output interface {
	Output(name string, outputs map[string]*states.OutputValue) tfdiags.Diagnostics
	Diagnostics(diags tfdiags.Diagnostics)
}

// NewOutput returns an initialized Output implementation for the given ViewType.
func NewOutput(vt arguments.ViewType, view *View) Output {
	switch vt {
	case arguments.ViewJSON:
		return &OutputJSON{view: view}
	case arguments.ViewRaw:
		return &OutputRaw{view: view}
	case arguments.ViewHuman:
		return &OutputHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The OutputHuman implementation renders outputs in a format equivalent to HCL
// source. This uses the same formatting logic as in the console REPL.
type OutputHuman struct {
	view *View
}

var _ Output = (*OutputHuman)(nil)

func (v *OutputHuman) Output(name string, outputs map[string]*states.OutputValue) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if len(outputs) == 0 {
		diags = diags.Append(noOutputsWarning())
		return diags
	}

	if name != "" {
		output, ok := outputs[name]
		if !ok {
			diags = diags.Append(missingOutputError(name))
			return diags
		}
		if output.Ephemeral {
			diags = diags.Append(ephemeralOutputError(name))
			return diags
		}
		result := repl.FormatValue(output.Value, 0)
		v.view.streams.Println(result)
		return nil
	}

	outputBuf := new(bytes.Buffer)
	if len(outputs) > 0 {
		// Output the outputs in alphabetical order
		keyLen := 0
		ks := make([]string, 0, len(outputs))
		for key := range outputs {
			ks = append(ks, key)
			if len(key) > keyLen {
				keyLen = len(key)
			}
		}
		sort.Strings(ks)

		for _, k := range ks {
			v := outputs[k]
			if v.Ephemeral && v.Sensitive {
				outputBuf.WriteString(fmt.Sprintf("%s = <ephemeral, sensitive>\n", k))
				continue
			}
			if v.Ephemeral {
				outputBuf.WriteString(fmt.Sprintf("%s = <ephemeral>\n", k))
				continue
			}
			if v.Sensitive {
				outputBuf.WriteString(fmt.Sprintf("%s = <sensitive>\n", k))
				continue
			}

			result := repl.FormatValue(v.Value, 0)
			outputBuf.WriteString(fmt.Sprintf("%s = %s\n", k, result))
		}
	}

	v.view.streams.Println(strings.TrimSpace(outputBuf.String()))

	return nil
}

func (v *OutputHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// The OutputRaw implementation renders single string, number, or boolean
// output values directly and without quotes or other formatting. This is
// intended for use in shell scripting or other environments where the exact
// type of an output value is not important.
type OutputRaw struct {
	view *View
}

var _ Output = (*OutputRaw)(nil)

func (v *OutputRaw) Output(name string, outputs map[string]*states.OutputValue) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if len(outputs) == 0 {
		diags = diags.Append(noOutputsWarning())
		return diags
	}

	if name == "" {
		diags = diags.Append(fmt.Errorf("Raw output format is only supported for single outputs"))
		return diags
	}

	output, ok := outputs[name]
	if !ok {
		diags = diags.Append(missingOutputError(name))
		return diags
	}

	if output.Ephemeral {
		diags = diags.Append(ephemeralOutputError(name))
		return diags
	}

	strV, err := convert.Convert(output.Value, cty.String)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported value for raw output",
			fmt.Sprintf(
				"The -raw option only supports strings, numbers, and boolean values, but output value %q is %s.\n\nUse the -json option for machine-readable representations of output values that have complex types.",
				name, output.Value.Type().FriendlyName(),
			),
		))
		return diags
	}
	if strV.IsNull() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported value for raw output",
			fmt.Sprintf(
				"The value for output value %q is null, so -raw mode cannot print it.",
				name,
			),
		))
		return diags
	}
	if !strV.IsKnown() {
		// Since we're working with values from the state it would be very
		// odd to end up in here, but we'll handle it anyway to avoid a
		// panic in case our rules somehow change in future.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported value for raw output",
			fmt.Sprintf(
				"The value for output value %q won't be known until after a successful terraform apply, so -raw mode cannot print it.",
				name,
			),
		))
		return diags
	}
	// If we get out here then we should have a valid string to print.
	// We're writing it using Print here so that a shell caller will get
	// exactly the value and no extra whitespace (including trailing newline).
	v.view.streams.Print(strV.AsString())
	return nil
}

func (v *OutputRaw) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// The OutputJSON implementation renders outputs as JSON values. When rendering
// a single output, only the value is displayed. When rendering all outputs,
// the result is a JSON object with keys matching the output names and object
// values including type and sensitivity metadata.
type OutputJSON struct {
	view *View
}

var _ Output = (*OutputJSON)(nil)

func (v *OutputJSON) Output(name string, outputs map[string]*states.OutputValue) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if name != "" {
		output, ok := outputs[name]
		if !ok {
			diags = diags.Append(missingOutputError(name))
			return diags
		}
		if output.Ephemeral {
			diags = diags.Append(ephemeralOutputError(name))
			return diags
		}
		value := output.Value

		jsonOutput, err := ctyjson.Marshal(value, value.Type())
		if err != nil {
			diags = diags.Append(err)
			return diags
		}

		v.view.streams.Println(string(jsonOutput))

		return nil
	}

	// Due to a historical accident, the switch from state version 2 to
	// 3 caused our JSON output here to be the full metadata about the
	// outputs rather than just the output values themselves as we'd
	// show in the single value case. We must now maintain that behavior
	// for compatibility, so this is an emulation of the JSON
	// serialization of outputs used in state format version 3.
	//
	// Note that when running the output command, the value of an ephemeral
	// output is always nil and its type is always cty.DynamicPseudoType.
	type OutputMeta struct {
		Ephemeral bool            `json:"ephemeral"`
		Sensitive bool            `json:"sensitive"`
		Type      json.RawMessage `json:"type"`
		Value     json.RawMessage `json:"value"`
	}
	outputMetas := map[string]OutputMeta{}

	for n, os := range outputs {
		jsonVal, err := ctyjson.Marshal(os.Value, os.Value.Type())
		if err != nil {
			diags = diags.Append(err)
			return diags
		}
		jsonType, err := ctyjson.MarshalType(os.Value.Type())
		if err != nil {
			diags = diags.Append(err)
			return diags
		}
		outputMetas[n] = OutputMeta{
			Ephemeral: os.Ephemeral,
			Sensitive: os.Sensitive,
			Type:      json.RawMessage(jsonType),
			Value:     json.RawMessage(jsonVal),
		}
	}

	jsonOutputs, err := json.MarshalIndent(outputMetas, "", "  ")
	if err != nil {
		diags = diags.Append(err)
		return diags
	}

	v.view.streams.Println(string(jsonOutputs))

	return nil
}

func (v *OutputJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// For text and raw output modes, an empty map of outputs is considered a
// separate and higher priority failure mode than an output not being present
// in a non-empty map. This warning diagnostic explains how this might have
// happened.
func noOutputsWarning() tfdiags.Diagnostic {
	return tfdiags.Sourceless(
		tfdiags.Warning,
		"No outputs found",
		"The state file either has no outputs defined, or all the defined "+
			"outputs are empty. Please define an output in your configuration "+
			"with the `output` keyword and run `terraform refresh` for it to "+
			"become available. If you are using interpolation, please verify "+
			"the interpolated value is not empty. You can use the "+
			"`terraform console` command to assist.",
	)
}

// Attempting to display a missing output results in this failure, which
// includes suggestions on how to rectify the problem.
func missingOutputError(name string) tfdiags.Diagnostic {
	return tfdiags.Sourceless(
		tfdiags.Error,
		fmt.Sprintf("Output %q not found", name),
		"The output variable requested could not be found in the state "+
			"file. If you recently added this to your configuration, be "+
			"sure to run `terraform apply`, since the state won't be updated "+
			"with new output variables until that command is run.",
	)
}

func ephemeralOutputError(name string) tfdiags.Diagnostic {
	return tfdiags.Sourceless(
		tfdiags.Error,
		fmt.Sprintf("Output %q is ephemeral", name),
		"The output requested is not available. It is marked as ephemeral "+
			"and therefore not persisted to state.",
	)
}
