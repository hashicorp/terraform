// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package structured

import (
	"github.com/hashicorp/terraform/jsonformat/computed"
)

type ProcessUnknown func(current Change) computed.Diff
type ProcessUnknownWithBefore func(current Change, before Change) computed.Diff

func (change Change) IsUnknown() bool {
	if unknown, ok := change.Unknown.(bool); ok {
		return unknown
	}
	return false
}

// CheckForUnknown is a helper function that handles all common functionality
// for processing an unknown value.
//
// It returns the computed unknown diff and true if this value was unknown and
// needs to be rendered as such, otherwise it returns the second return value as
// false and the first return value should be discarded.
//
// The actual processing of unknown values happens in the ProcessUnknown and
// ProcessUnknownWithBefore functions. If a value is unknown and is being
// created, the ProcessUnknown function is called and the caller should decide
// how to create the unknown value. If a value is being updated the
// ProcessUnknownWithBefore function is called and the function provides the
// before value as if it is being deleted for the caller to handle. Note that
// values being deleted will never be marked as unknown so this case isn't
// handled.
//
// The childUnknown argument is meant to allow callers with extra information
// about the type being processed to provide a list of known children that might
// not be present in the before or after values. These values will be propagated
// as the unknown values in the before value should it be needed.
func (change Change) CheckForUnknown(childUnknown interface{}, process ProcessUnknown, processBefore ProcessUnknownWithBefore) (computed.Diff, bool) {
	unknown := change.IsUnknown()

	if !unknown {
		return computed.Diff{}, false
	}

	// No matter what we do here, we want to treat the after value as explicit.
	// This is because it is going to be null in the value, and we don't want
	// the functions in this package to assume this means it has been deleted.
	change.AfterExplicit = true

	if change.Before == nil {
		return process(change), true
	}

	// If we get here, then we have a before value. We're going to model a
	// delete operation and our renderer later can render the overall change
	// accurately.
	before := change.AsDelete()

	// We also let our callers override the unknown values in any before, this
	// is the renderers can display them as being computed instead of deleted.
	before.Unknown = childUnknown
	return processBefore(change, before), true
}
