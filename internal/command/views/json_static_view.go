// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"encoding/json"

	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func NewJSONStaticView[T any](view *View) *JSONStaticView[T] {
	return &JSONStaticView[T]{
		view: view,

		schema: *new(T),
	}
}

type JSONStaticView[T any] struct {
	view *View

	schema T
}

// Print is a shared convenience method for printing the given data as indented JSON to stdout.
// Methods that intentionally do not indent their output should not use this function.
// Examples: `metadata function`, `providers schema`
func (v *JSONStaticView[T]) Print(data T) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		// Should never happen because we fully-control the input here
		panic(err)
	}

	v.view.streams.Println(string(b))
}

// Eprint is a shared convenience method for printing the given data as indented JSON to stderr.
// Methods that intentionally do not indent their output should not use this function.
// Examples: `metadata function`, `providers schema`
//
// This method is intended to be used when the data includes an error diagnostic, i.e. when a command
// is returning early due to an error and the output contains diagnostics along with zero-values.
func (v *JSONStaticView[T]) Eprint(data T) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		// Should never happen because we fully-control the input here
		panic(err)
	}

	v.view.streams.Eprintln(string(b))
}

func (v *JSONStaticView[T]) prepareDiagnostics(diags tfdiags.Diagnostics) []*viewsjson.Diagnostic {
	configSources := v.view.configSources()
	result := []*viewsjson.Diagnostic{}
	for _, diag := range diags {
		result = append(result, viewsjson.NewDiagnostic(diag, configSources))
	}
	return result
}
