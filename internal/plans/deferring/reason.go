package deferring

import (
	"github.com/zclconf/go-cty/cty"
)

// Reason is an enumeration of possible reasons for deferring.
type Reason rune

//go:generate go run golang.org/x/tools/cmd/stringer -type=Reason

const (
	// BecauseUpstream means that an action was deferred because it depends
	// on the result of some other action that was deferred.
	//
	// In this case the proposed action should typically be complete in itself
	// but must be deferred anyway to preserve the correct order of operations
	// in relation to some other deferred action.
	BecauseUpstream = '↰'

	// BecauseExpansionUnknown means that an action was deferred because
	// Terraform can't yet calculate the full set of instances of the object
	// whose action is being described.
	BecauseExpansionUnknown = '#'

	// BecauseProviderConfigUnknown means that a resource instance action was
	// deferred because the configuration for the provider instance that would
	// be performing the action is not yet sufficiently known to produce a
	// complete plan.
	BecauseProviderConfigUnknown = 'P'

	// BecauseResourceInstanceConfigUnknown means that a resource instance
	// action was deferred because the configuration for that resource instance
	// is not yet sufficiently known to produce a complete plan.
	BecauseResourceInstanceConfigUnknown = 'R'
)

// Explanation combines a deferral reason with other reason-specific
// information.
type Explanation struct {
	// Reason describes both the deferral reason and how to interpret the
	// other fields of this type.
	Reason Reason

	// ArgPath describes an attribute path to an argument in an object that
	// specifically caused the deferral.
	//
	// - For [BecauseProviderConfigUnknown] this is a path into the provider
	//   configuration block that the relevant resource is associated with.
	// - For [BecauseResourceInstanceConfigUnknown] this is a path into the
	//   configuration of the relevant resource instance.
	// - For all other reasons this is always nil.
	ArgPath cty.Path
}
