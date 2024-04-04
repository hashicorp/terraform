package plans

import "github.com/zclconf/go-cty/cty"

type DeferredReason string

const (
	// DeferredReasonInvalid is used when the reason for deferring is
	// unknown or irrelevant.
	DeferredReasonInvalid DeferredReason = "invalid"

	// DeferredReasonInstanceCountUnknown is used when the reason for deferring
	// is that the count or for_each meta-attribute was unknown.
	DeferredReasonInstanceCountUnknown DeferredReason = "instance_count_unknown"

	// DeferredReasonResourceConfigUnknown is used when the reason for deferring
	// is that the resource configuration was unknown.
	DeferredReasonResourceConfigUnknown DeferredReason = "resource_config_unknown"

	// DeferredReasonProviderConfigUnknown is used when the reason for deferring
	// is that the provider configuration was unknown.
	DeferredReasonProviderConfigUnknown DeferredReason = "provider_config_unknown"

	// DeferredReasonAbsentPrereq is used when the reason for deferring is that
	// a required prerequisite resource was absent.
	DeferredReasonAbsentPrereq DeferredReason = "absent_prereq"

	// DeferredReasonDeferredPrereq is used when the reason for deferring is
	// that a required prerequisite resource was itself deferred.
	DeferredReasonDeferredPrereq DeferredReason = "deferred_prereq"
)

// DeferredResourceInstanceChangeSrc tracks information about a resource that
// has been deferred for some reason.
type DeferredResourceInstanceChangeSrc struct {
	// DeferredReason is the reason why this resource instance was deferred.
	DeferredReason DeferredReason

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
	DeferredReason DeferredReason

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
