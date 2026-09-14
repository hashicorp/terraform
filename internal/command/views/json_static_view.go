// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"encoding/json"
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

// Print is a shared convenience method for printing the given data as indented JSON.
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

// Diagnostics is a shared convenience method for printing diagnostics.
// When static JSON output is produced from a command, we expect error diagnostics
// to be rendered in a JSON object with the same schema as the command's output.
//
// This is implemented mostly to assert that the diagnostics are printed in the
// same format as the command's output. Calling code should use this method, instead
// of accessing the Diagnostics method of the underlying view. If that's done then
// the output will be in human-readable format regardless the view type in use.
func (v *JSONStaticView[T]) Diagnostics(data T) {
	v.Print(data)
}
