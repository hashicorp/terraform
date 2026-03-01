// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

// ViewType represents which view layer to use for a given command. Not all
// commands will support all view types, and validation that the type is
// supported should happen in the view constructor.
type ViewType rune

const (
	ViewNone          ViewType = 0
	ViewHuman         ViewType = 'H'
	ViewHumanRedacted ViewType = 'I'
	ViewJSON          ViewType = 'J'
	ViewJSONRedacted  ViewType = 'K'
	ViewRaw           ViewType = 'R'
)

func (vt ViewType) String() string {
	switch vt {
	case ViewNone:
		return "none"
	case ViewHuman:
		return "human"
	case ViewHumanRedacted:
		return "humanredacted"
	case ViewJSON:
		return "json"
	case ViewJSONRedacted:
		return "jsonredacted"
	case ViewRaw:
		return "raw"
	default:
		return "unknown"
	}
}
