// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package json

// Importing contains metadata about a resource change that includes an import
// action.
//
// Every field in here should be treated as optional as future versions do not
// make a guarantee that they will retain the format of this change.
//
// Consumers should be capable of rendering/parsing the Importing struct even
// if it does not have the ID field set.
type Importing struct {
	ID string `json:"id,omitempty"`
}
