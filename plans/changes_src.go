package plans

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
	"github.com/zclconf/go-cty/cty"
)

// ResourceInstanceChangeSrc is a not-yet-decoded ResourceInstanceChange.
// Pass the associated resource type's schema type to method Decode to
// obtain a ResourceInstancChange.
type ResourceInstanceChangeSrc struct {
	// Addr is the absolute address of the resource instance that the change
	// will apply to.
	Addr addrs.AbsResourceInstance

	// DeposedKey is the identifier for a deposed object associated with the
	// given instance, or states.NotDeposed if this change applies to the
	// current object.
	//
	// A Replace change for a resource with create_before_destroy set will
	// create a new DeposedKey temporarily during replacement. In that case,
	// DeposedKey in the plan is always states.NotDeposed, representing that
	// the current object is being replaced with the deposed.
	DeposedKey states.DeposedKey

	// Provider is the address of the provider configuration that was used
	// to plan this change, and thus the configuration that must also be
	// used to apply it.
	ProviderAddr addrs.AbsProviderConfig

	// ChangeSrc is an embedded description of the not-yet-decoded change.
	ChangeSrc

	// RequiredReplace is a set of paths that caused the change action to be
	// Replace rather than Update. Always nil if the change action is not
	// Replace.
	//
	// This is retained only for UI-plan-rendering purposes and so it does not
	// currently survive a round-trip through a saved plan file.
	RequiredReplace []cty.Path

	// Private allows a provider to stash any extra data that is opaque to
	// Terraform that relates to this change. Terraform will save this
	// byte-for-byte and return it to the provider in the apply call.
	Private []byte
}

// Decode unmarshals the raw representation of the instance object being
// changed. Pass the implied type of the corresponding resource type schema
// for correct operation.
func (rcs *ResourceInstanceChangeSrc) Decode(ty cty.Type) (*ResourceInstanceChange, error) {
	change, err := rcs.ChangeSrc.Decode(ty)
	if err != nil {
		return nil, err
	}
	return &ResourceInstanceChange{
		Addr:            rcs.Addr,
		DeposedKey:      rcs.DeposedKey,
		ProviderAddr:    rcs.ProviderAddr,
		Change:          *change,
		RequiredReplace: rcs.RequiredReplace,
		Private:         rcs.Private,
	}, nil
}

// OutputChange describes a change to an output value.
type OutputChangeSrc struct {
	// ChangeSrc is an embedded description of the not-yet-decoded change.
	//
	// For output value changes, the type constraint for the DynamicValue
	// instances is always cty.DynamicPseudoType.
	ChangeSrc

	// Sensitive, if true, indicates that either the old or new value in the
	// change is sensitive and so a rendered version of the plan in the UI
	// should elide the actual values while still indicating the action of the
	// change.
	Sensitive bool
}

// Decode unmarshals the raw representation of the output value being
// changed.
func (ocs *OutputChangeSrc) Decode() (*OutputChange, error) {
	change, err := ocs.ChangeSrc.Decode(cty.DynamicPseudoType)
	if err != nil {
		return nil, err
	}
	return &OutputChange{
		Change:    *change,
		Sensitive: ocs.Sensitive,
	}, nil
}

// ChangeSrc is a not-yet-decoded Change.
type ChangeSrc struct {
	// Action defines what kind of change is being made.
	Action Action

	// Before and After correspond to the fields of the same name in Change,
	// but have not yet been decoded from the serialized value used for
	// storage.
	Before, After DynamicValue
}

// Decode unmarshals the raw representations of the before and after values
// to produce a Change object. Pass the type constraint that the result must
// conform to.
//
// Where a ChangeSrc is embedded in some other struct, it's generally better
// to call the corresponding Decode method of that struct rather than working
// directly with its embedded Change.
func (cs *ChangeSrc) Decode(ty cty.Type) (*Change, error) {
	before, err := cs.Before.Decode(ty)
	if err != nil {
		return nil, fmt.Errorf("error decoding 'before' value: %s", err)
	}
	after, err := cs.After.Decode(ty)
	if err != nil {
		return nil, fmt.Errorf("error decoding 'after' value: %s", err)
	}
	return &Change{
		Action: cs.Action,
		Before: before,
		After:  after,
	}, nil
}
