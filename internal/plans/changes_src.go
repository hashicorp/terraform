package plans

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

// ResourceInstanceChangeSrc is a not-yet-decoded ResourceInstanceChange.
// Pass the associated resource type's schema type to method Decode to
// obtain a ResourceInstanceChange.
type ResourceInstanceChangeSrc struct {
	// Addr is the absolute address of the resource instance that the change
	// will apply to.
	Addr addrs.AbsResourceInstance

	// PrevRunAddr is the absolute address that this resource instance had at
	// the conclusion of a previous run.
	//
	// This will typically be the same as Addr, but can be different if the
	// previous resource instance was subject to a "moved" block that we
	// handled in the process of creating this plan.
	//
	// For the initial creation of a resource instance there isn't really any
	// meaningful "previous run address", but PrevRunAddr will still be set
	// equal to Addr in that case in order to simplify logic elsewhere which
	// aims to detect and react to the movement of instances between addresses.
	PrevRunAddr addrs.AbsResourceInstance

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

	// ActionReason is an optional extra indication of why we chose the
	// action recorded in Change.Action for this particular resource instance.
	//
	// This is an approximate mechanism only for the purpose of explaining the
	// plan to end-users in the UI and is not to be used for any
	// decision-making during the apply step; if apply behavior needs to vary
	// depending on the "action reason" then the information for that decision
	// must be recorded more precisely elsewhere for that purpose.
	//
	// See the field of the same name in ResourceInstanceChange for more
	// details.
	ActionReason ResourceInstanceChangeActionReason

	// RequiredReplace is a set of paths that caused the change action to be
	// Replace rather than Update. Always nil if the change action is not
	// Replace.
	RequiredReplace cty.PathSet

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
	prevRunAddr := rcs.PrevRunAddr
	if prevRunAddr.Resource.Resource.Type == "" {
		// Suggests an old caller that hasn't been properly updated to
		// populate this yet.
		prevRunAddr = rcs.Addr
	}
	return &ResourceInstanceChange{
		Addr:            rcs.Addr,
		PrevRunAddr:     prevRunAddr,
		DeposedKey:      rcs.DeposedKey,
		ProviderAddr:    rcs.ProviderAddr,
		Change:          *change,
		ActionReason:    rcs.ActionReason,
		RequiredReplace: rcs.RequiredReplace,
		Private:         rcs.Private,
	}, nil
}

// DeepCopy creates a copy of the receiver where any pointers to nested mutable
// values are also copied, thus ensuring that future mutations of the receiver
// will not affect the copy.
//
// Some types used within a resource change are immutable by convention even
// though the Go language allows them to be mutated, such as the types from
// the addrs package. These are _not_ copied by this method, under the
// assumption that callers will behave themselves.
func (rcs *ResourceInstanceChangeSrc) DeepCopy() *ResourceInstanceChangeSrc {
	if rcs == nil {
		return nil
	}
	ret := *rcs

	ret.RequiredReplace = cty.NewPathSet(ret.RequiredReplace.List()...)

	if len(ret.Private) != 0 {
		private := make([]byte, len(ret.Private))
		copy(private, ret.Private)
		ret.Private = private
	}

	ret.ChangeSrc.Before = ret.ChangeSrc.Before.Copy()
	ret.ChangeSrc.After = ret.ChangeSrc.After.Copy()

	return &ret
}

func (rcs *ResourceInstanceChangeSrc) Moved() bool {
	return !rcs.Addr.Equal(rcs.PrevRunAddr)
}

// OutputChangeSrc describes a change to an output value.
type OutputChangeSrc struct {
	// Addr is the absolute address of the output value that the change
	// will apply to.
	Addr addrs.AbsOutputValue

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
		Addr:      ocs.Addr,
		Change:    *change,
		Sensitive: ocs.Sensitive,
	}, nil
}

// DeepCopy creates a copy of the receiver where any pointers to nested mutable
// values are also copied, thus ensuring that future mutations of the receiver
// will not affect the copy.
//
// Some types used within a resource change are immutable by convention even
// though the Go language allows them to be mutated, such as the types from
// the addrs package. These are _not_ copied by this method, under the
// assumption that callers will behave themselves.
func (ocs *OutputChangeSrc) DeepCopy() *OutputChangeSrc {
	if ocs == nil {
		return nil
	}
	ret := *ocs

	ret.ChangeSrc.Before = ret.ChangeSrc.Before.Copy()
	ret.ChangeSrc.After = ret.ChangeSrc.After.Copy()

	return &ret
}

// ChangeSrc is a not-yet-decoded Change.
type ChangeSrc struct {
	// Action defines what kind of change is being made.
	Action Action

	// Before and After correspond to the fields of the same name in Change,
	// but have not yet been decoded from the serialized value used for
	// storage.
	Before, After DynamicValue

	// BeforeValMarks and AfterValMarks are stored path+mark combinations
	// that might be discovered when encoding a change. Marks are removed
	// to enable encoding (marked values cannot be marshalled), and so storing
	// the path+mark combinations allow us to re-mark the value later
	// when, for example, displaying the diff to the UI.
	BeforeValMarks, AfterValMarks []cty.PathValueMarks
}

// Decode unmarshals the raw representations of the before and after values
// to produce a Change object. Pass the type constraint that the result must
// conform to.
//
// Where a ChangeSrc is embedded in some other struct, it's generally better
// to call the corresponding Decode method of that struct rather than working
// directly with its embedded Change.
func (cs *ChangeSrc) Decode(ty cty.Type) (*Change, error) {
	var err error
	before := cty.NullVal(ty)
	after := cty.NullVal(ty)

	if len(cs.Before) > 0 {
		before, err = cs.Before.Decode(ty)
		if err != nil {
			return nil, fmt.Errorf("error decoding 'before' value: %s", err)
		}
	}
	if len(cs.After) > 0 {
		after, err = cs.After.Decode(ty)
		if err != nil {
			return nil, fmt.Errorf("error decoding 'after' value: %s", err)
		}
	}

	return &Change{
		Action: cs.Action,
		Before: before.MarkWithPaths(cs.BeforeValMarks),
		After:  after.MarkWithPaths(cs.AfterValMarks),
	}, nil
}
