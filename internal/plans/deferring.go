// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
)

// DeferredResourceInstanceChangeSrc tracks information about a resource that
// has been deferred for some reason.
type DeferredResourceInstanceChangeSrc struct {
	// DeferredReason is the reason why this resource instance was deferred.
	DeferredReason providers.DeferredReason

	// ChangeSrc contains any information we have about the deferred change.
	// This could be incomplete so must be parsed with care.
	ChangeSrc *ResourceInstanceChangeSrc
}

func (rcs *DeferredResourceInstanceChangeSrc) Decode(ty cty.Type) (*DeferredResourceInstanceChange, error) {
	change, err := rcs.ChangeSrc.Decode(ty)
	if err != nil {
		return nil, err
	}

	return &DeferredResourceInstanceChange{
		DeferredReason: rcs.DeferredReason,
		Change:         change,
	}, nil
}

// DeferredResourceInstanceChange tracks information about a resource that
// has been deferred for some reason.
type DeferredResourceInstanceChange struct {
	// DeferredReason is the reason why this resource instance was deferred.
	DeferredReason providers.DeferredReason

	// Change contains any information we have about the deferred change. This
	// could be incomplete so must be parsed with care.
	Change *ResourceInstanceChange
}

func (rcs *DeferredResourceInstanceChange) Encode(ty cty.Type) (*DeferredResourceInstanceChangeSrc, error) {
	change, err := rcs.Change.Encode(ty)
	if err != nil {
		return nil, err
	}

	return &DeferredResourceInstanceChangeSrc{
		DeferredReason: rcs.DeferredReason,
		ChangeSrc:      change,
	}, nil
}
