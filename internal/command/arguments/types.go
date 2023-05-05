// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arguments

// ViewType represents which view layer to use for a given command. Not all
// commands will support all view types, and validation that the type is
// supported should happen in the view constructor.
type ViewType rune

const (
	ViewNone  ViewType = 0
	ViewHuman ViewType = 'H'
	ViewJSON  ViewType = 'J'
	ViewRaw   ViewType = 'R'
)

func (vt ViewType) String() string {
	switch vt {
	case ViewNone:
		return "none"
	case ViewHuman:
		return "human"
	case ViewJSON:
		return "json"
	case ViewRaw:
		return "raw"
	default:
		return "unknown"
	}
}
