// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/states"
)

// ChangesSrc describes various actions that Terraform will attempt to take if
// the corresponding plan is applied.
//
// A Changes object can be rendered into a visual diff (by the caller, using
// code in another package) for display to the user.
type ChangesSrc struct {
	// Resources tracks planned changes to resource instance objects.
	Resources []*ResourceInstanceChangeSrc

	// Outputs tracks planned changes output values.
	//
	// Note that although an in-memory plan contains planned changes for
	// outputs throughout the configuration, a plan serialized
	// to disk retains only the root outputs because they are
	// externally-visible, while other outputs are implementation details and
	// can be easily re-calculated during the apply phase. Therefore only root
	// module outputs will survive a round-trip through a plan file.
	Outputs []*OutputChangeSrc
}

func NewChangesSrc() *ChangesSrc {
	return &ChangesSrc{}
}

func (c *ChangesSrc) Empty() bool {
	for _, res := range c.Resources {
		if res.Action != NoOp || res.Moved() {
			return false
		}

		if res.Importing != nil {
			return false
		}
	}

	for _, out := range c.Outputs {
		if out.Addr.Module.IsRoot() && out.Action != NoOp {
			return false
		}
	}

	return true
}

// ResourceInstance returns the planned change for the current object of the
// resource instance of the given address, if any. Returns nil if no change is
// planned.
func (c *ChangesSrc) ResourceInstance(addr addrs.AbsResourceInstance) *ResourceInstanceChangeSrc {
	for _, rc := range c.Resources {
		if rc.Addr.Equal(addr) && rc.DeposedKey == states.NotDeposed {
			return rc
		}
	}

	return nil
}

// ResourceInstanceDeposed returns the plan change of a deposed object of
// the resource instance of the given address, if any. Returns nil if no change
// is planned.
func (c *ChangesSrc) ResourceInstanceDeposed(addr addrs.AbsResourceInstance, key states.DeposedKey) *ResourceInstanceChangeSrc {
	for _, rc := range c.Resources {
		if rc.Addr.Equal(addr) && rc.DeposedKey == key {
			return rc
		}
	}

	return nil
}

// OutputValue returns the planned change for the output value with the
//
//	given address, if any. Returns nil if no change is planned.
func (c *ChangesSrc) OutputValue(addr addrs.AbsOutputValue) *OutputChangeSrc {
	for _, oc := range c.Outputs {
		if oc.Addr.Equal(addr) {
			return oc
		}
	}

	return nil
}

// Decode decodes all the stored resource and output changes into a new *Changes value.
func (c *ChangesSrc) Decode(schemas *schemarepo.Schemas) (*Changes, error) {
	changes := NewChanges()

	for _, rcs := range c.Resources {
		p, ok := schemas.Providers[rcs.ProviderAddr.Provider]
		if !ok {
			return nil, fmt.Errorf("ChangesSrc.Decode: missing provider %s for %s", rcs.ProviderAddr, rcs.Addr)
		}

		var schema providers.Schema
		switch rcs.Addr.Resource.Resource.Mode {
		case addrs.ManagedResourceMode:
			schema = p.ResourceTypes[rcs.Addr.Resource.Resource.Type]
		case addrs.DataResourceMode:
			schema = p.DataSources[rcs.Addr.Resource.Resource.Type]
		default:
			panic(fmt.Sprintf("unexpected resource mode %s", rcs.Addr.Resource.Resource.Mode))
		}

		if schema.Block == nil {
			return nil, fmt.Errorf("ChangesSrc.Decode: missing schema for %s", rcs.Addr)
		}

		rc, err := rcs.Decode(schema.Block.ImpliedType())
		if err != nil {
			return nil, err
		}

		rc.Before = marks.MarkPaths(rc.Before, marks.Sensitive, rcs.BeforeSensitivePaths)
		rc.After = marks.MarkPaths(rc.After, marks.Sensitive, rcs.AfterSensitivePaths)

		changes.Resources = append(changes.Resources, rc)
	}

	for _, ocs := range c.Outputs {
		oc, err := ocs.Decode()
		if err != nil {
			return nil, err
		}
		changes.Outputs = append(changes.Outputs, oc)
	}
	return changes, nil
}

// AppendResourceInstanceChange records the given resource instance change in
// the set of planned resource changes.
func (c *ChangesSrc) AppendResourceInstanceChange(change *ResourceInstanceChangeSrc) {
	if c == nil {
		panic("AppendResourceInstanceChange on nil ChangesSync")
	}

	s := change.DeepCopy()
	c.Resources = append(c.Resources, s)
}

// ResourceInstanceChangeSrc is a not-yet-decoded ResourceInstanceChange.
// Pass the associated resource type's schema type to method Decode to
// obtain a ResourceInstanceChange.
type ResourceInstanceChangeSrc struct {
	// Addr is the absolute address of the resource instance that the change
	// will apply to.
	//
	// THIS IS NOT A SUFFICIENT UNIQUE IDENTIFIER! It doesn't consider the
	// fact that multiple objects for the same resource instance might be
	// present in the same plan; use the ObjectAddr method instead if you
	// need a unique address for a particular change.
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

func (rcs *ResourceInstanceChangeSrc) ObjectAddr() addrs.AbsResourceInstanceObject {
	return addrs.AbsResourceInstanceObject{
		ResourceInstance: rcs.Addr,
		DeposedKey:       rcs.DeposedKey,
	}
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

// ImportingSrc is the part of a ChangeSrc that describes the embedded import
// action.
//
// The fields in here are subject to change, so downstream consumers should be
// prepared for backwards compatibility in case the contents changes.
type ImportingSrc struct {
	// ID is the original ID of the imported resource.
	ID string

	// Unknown is true if the ID was unknown when we tried to import it. This
	// should only be true if the overall change is embedded within a deferred
	// action.
	Unknown bool
}

// Decode unmarshals the raw representation of the importing action.
func (is *ImportingSrc) Decode() *Importing {
	if is == nil {
		return nil
	}
	if is.Unknown {
		return &Importing{
			ID: cty.UnknownVal(cty.String),
		}
	}
	return &Importing{
		ID: cty.StringVal(is.ID),
	}
}

// ChangeSrc is a not-yet-decoded Change.
type ChangeSrc struct {
	// Action defines what kind of change is being made.
	Action Action

	// Before and After correspond to the fields of the same name in Change,
	// but have not yet been decoded from the serialized value used for
	// storage.
	Before, After DynamicValue

	// BeforeSensitivePaths and AfterSensitivePaths are the paths for any
	// values in Before or After (respectively) that are considered to be
	// sensitive. The sensitive marks are removed from the in-memory values
	// to enable encoding (marked values cannot be marshalled), and so we
	// store the sensitive paths to allow re-marking later when we decode
	// the serialized change.
	BeforeSensitivePaths, AfterSensitivePaths []cty.Path

	// Importing is present if the resource is being imported as part of this
	// change.
	//
	// Use the simple presence of this field to detect if a ChangeSrc is to be
	// imported, the contents of this structure may be modified going forward.
	Importing *ImportingSrc

	// GeneratedConfig contains any HCL config generated for this resource
	// during planning, as a string. If GeneratedConfig is populated, Importing
	// should be true. However, not all Importing changes contain generated
	// config.
	GeneratedConfig string
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
		Action:          cs.Action,
		Before:          marks.MarkPaths(before, marks.Sensitive, cs.BeforeSensitivePaths),
		After:           marks.MarkPaths(after, marks.Sensitive, cs.AfterSensitivePaths),
		Importing:       cs.Importing.Decode(),
		GeneratedConfig: cs.GeneratedConfig,
	}, nil
}
