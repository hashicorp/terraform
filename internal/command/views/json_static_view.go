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
