package plans

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
)

// Changes describes various actions that Terraform will attempt to take if
// the corresponding plan is applied.
//
// A Changes object can be rendered into a visual diff (by the caller, using
// code in another package) for display to the user.
type Changes struct {
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

// NewChanges returns a valid Changes object that describes no changes.
func NewChanges() *Changes {
	return &Changes{}
}

func (c *Changes) Empty() bool {
	for _, res := range c.Resources {
		if res.Action != NoOp || res.Moved() {
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
func (c *Changes) ResourceInstance(addr addrs.AbsResourceInstance) *ResourceInstanceChangeSrc {
	for _, rc := range c.Resources {
		if rc.Addr.Equal(addr) && rc.DeposedKey == states.NotDeposed {
			return rc
		}
	}

	return nil

}

// InstancesForAbsResource returns the planned change for the current objects
// of the resource instances of the given address, if any. Returns nil if no
// changes are planned.
func (c *Changes) InstancesForAbsResource(addr addrs.AbsResource) []*ResourceInstanceChangeSrc {
	var changes []*ResourceInstanceChangeSrc
	for _, rc := range c.Resources {
		resAddr := rc.Addr.ContainingResource()
		if resAddr.Equal(addr) && rc.DeposedKey == states.NotDeposed {
			changes = append(changes, rc)
		}
	}

	return changes
}

// InstancesForConfigResource returns the planned change for the current objects
// of the resource instances of the given address, if any. Returns nil if no
// changes are planned.
func (c *Changes) InstancesForConfigResource(addr addrs.ConfigResource) []*ResourceInstanceChangeSrc {
	var changes []*ResourceInstanceChangeSrc
	for _, rc := range c.Resources {
		resAddr := rc.Addr.ContainingResource().Config()
		if resAddr.Equal(addr) && rc.DeposedKey == states.NotDeposed {
			changes = append(changes, rc)
		}
	}

	return changes
}

// ResourceInstanceDeposed returns the plan change of a deposed object of
// the resource instance of the given address, if any. Returns nil if no change
// is planned.
func (c *Changes) ResourceInstanceDeposed(addr addrs.AbsResourceInstance, key states.DeposedKey) *ResourceInstanceChangeSrc {
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
func (c *Changes) OutputValue(addr addrs.AbsOutputValue) *OutputChangeSrc {
	for _, oc := range c.Outputs {
		if oc.Addr.Equal(addr) {
			return oc
		}
	}

	return nil
}

// RootOutputValues returns planned changes for all outputs of the root module.
func (c *Changes) RootOutputValues() []*OutputChangeSrc {
	var res []*OutputChangeSrc

	for _, oc := range c.Outputs {
		// we can't evaluate root module outputs
		if !oc.Addr.Module.Equal(addrs.RootModuleInstance) {
			continue
		}

		res = append(res, oc)

	}

	return res
}

// OutputValues returns planned changes for all outputs for all module
// instances that reside in the parent path.  Returns nil if no changes are
// planned.
func (c *Changes) OutputValues(parent addrs.ModuleInstance, module addrs.ModuleCall) []*OutputChangeSrc {
	var res []*OutputChangeSrc

	for _, oc := range c.Outputs {
		// we can't evaluate root module outputs
		if oc.Addr.Module.Equal(addrs.RootModuleInstance) {
			continue
		}

		changeMod, changeCall := oc.Addr.Module.Call()
		// this does not reside on our parent instance path
		if !changeMod.Equal(parent) {
			continue
		}

		// this is not the module you're looking for
		if changeCall.Name != module.Name {
			continue
		}

		res = append(res, oc)

	}

	return res
}

// SyncWrapper returns a wrapper object around the receiver that can be used
// to make certain changes to the receiver in a concurrency-safe way, as long
// as all callers share the same wrapper object.
func (c *Changes) SyncWrapper() *ChangesSync {
	return &ChangesSync{
		changes: c,
	}
}

// ResourceInstanceChange describes a change to a particular resource instance
// object.
type ResourceInstanceChange struct {
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

	// Change is an embedded description of the change.
	Change

	// ActionReason is an optional extra indication of why we chose the
	// action recorded in Change.Action for this particular resource instance.
	//
	// This is an approximate mechanism only for the purpose of explaining the
	// plan to end-users in the UI and is not to be used for any
	// decision-making during the apply step; if apply behavior needs to vary
	// depending on the "action reason" then the information for that decision
	// must be recorded more precisely elsewhere for that purpose.
	//
	// Sometimes there might be more than one reason for choosing a particular
	// action. In that case, it's up to the codepath making that decision to
	// decide which value would provide the most relevant explanation to the
	// end-user and return that. It's not a goal of this field to represent
	// fine details about the planning process.
	ActionReason ResourceInstanceChangeActionReason

	// RequiredReplace is a set of paths that caused the change action to be
	// Replace rather than Update. Always nil if the change action is not
	// Replace.
	//
	// This is retained only for UI-plan-rendering purposes and so it does not
	// currently survive a round-trip through a saved plan file.
	RequiredReplace cty.PathSet

	// Private allows a provider to stash any extra data that is opaque to
	// Terraform that relates to this change. Terraform will save this
	// byte-for-byte and return it to the provider in the apply call.
	Private []byte
}

// Encode produces a variant of the reciever that has its change values
// serialized so it can be written to a plan file. Pass the implied type of the
// corresponding resource type schema for correct operation.
func (rc *ResourceInstanceChange) Encode(ty cty.Type) (*ResourceInstanceChangeSrc, error) {
	cs, err := rc.Change.Encode(ty)
	if err != nil {
		return nil, err
	}
	prevRunAddr := rc.PrevRunAddr
	if prevRunAddr.Resource.Resource.Type == "" {
		// Suggests an old caller that hasn't been properly updated to
		// populate this yet.
		prevRunAddr = rc.Addr
	}
	return &ResourceInstanceChangeSrc{
		Addr:            rc.Addr,
		PrevRunAddr:     prevRunAddr,
		DeposedKey:      rc.DeposedKey,
		ProviderAddr:    rc.ProviderAddr,
		ChangeSrc:       *cs,
		ActionReason:    rc.ActionReason,
		RequiredReplace: rc.RequiredReplace,
		Private:         rc.Private,
	}, err
}

func (rc *ResourceInstanceChange) Moved() bool {
	return !rc.Addr.Equal(rc.PrevRunAddr)
}

// Simplify will, where possible, produce a change with a simpler action than
// the receiever given a flag indicating whether the caller is dealing with
// a normal apply or a destroy. This flag deals with the fact that Terraform
// Core uses a specialized graph node type for destroying; only that
// specialized node should set "destroying" to true.
//
// The following table shows the simplification behavior:
//
//	Action    Destroying?   New Action
//	--------+-------------+-----------
//	Create    true          NoOp
//	Delete    false         NoOp
//	Replace   true          Delete
//	Replace   false         Create
//
// For any combination not in the above table, the Simplify just returns the
// receiver as-is.
func (rc *ResourceInstanceChange) Simplify(destroying bool) *ResourceInstanceChange {
	if destroying {
		switch rc.Action {
		case Delete:
			// We'll fall out and just return rc verbatim, then.
		case CreateThenDelete, DeleteThenCreate:
			return &ResourceInstanceChange{
				Addr:         rc.Addr,
				DeposedKey:   rc.DeposedKey,
				Private:      rc.Private,
				ProviderAddr: rc.ProviderAddr,
				Change: Change{
					Action: Delete,
					Before: rc.Before,
					After:  cty.NullVal(rc.Before.Type()),
				},
			}
		default:
			return &ResourceInstanceChange{
				Addr:         rc.Addr,
				DeposedKey:   rc.DeposedKey,
				Private:      rc.Private,
				ProviderAddr: rc.ProviderAddr,
				Change: Change{
					Action: NoOp,
					Before: rc.Before,
					After:  rc.Before,
				},
			}
		}
	} else {
		switch rc.Action {
		case Delete:
			return &ResourceInstanceChange{
				Addr:         rc.Addr,
				DeposedKey:   rc.DeposedKey,
				Private:      rc.Private,
				ProviderAddr: rc.ProviderAddr,
				Change: Change{
					Action: NoOp,
					Before: rc.Before,
					After:  rc.Before,
				},
			}
		case CreateThenDelete, DeleteThenCreate:
			return &ResourceInstanceChange{
				Addr:         rc.Addr,
				DeposedKey:   rc.DeposedKey,
				Private:      rc.Private,
				ProviderAddr: rc.ProviderAddr,
				Change: Change{
					Action: Create,
					Before: cty.NullVal(rc.After.Type()),
					After:  rc.After,
				},
			}
		}
	}

	// If we fall out here then our change is already simple enough.
	return rc
}

// ResourceInstanceChangeActionReason allows for some extra user-facing
// reasoning for why a particular change action was chosen for a particular
// resource instance.
//
// This only represents sufficient detail to give a suitable explanation to
// an end-user, and mustn't be used for any real decision-making during the
// apply step.
type ResourceInstanceChangeActionReason rune

//go:generate go run golang.org/x/tools/cmd/stringer -type=ResourceInstanceChangeActionReason changes.go

const (
	// In most cases there's no special reason for choosing a particular
	// action, which is represented by ResourceInstanceChangeNoReason.
	ResourceInstanceChangeNoReason ResourceInstanceChangeActionReason = 0

	// ResourceInstanceReplaceBecauseTainted indicates that the resource
	// instance must be replaced because its existing current object is
	// marked as "tainted".
	ResourceInstanceReplaceBecauseTainted ResourceInstanceChangeActionReason = 'T'

	// ResourceInstanceReplaceByRequest indicates that the resource instance
	// is planned to be replaced because a caller specifically asked for it
	// to be using ReplaceAddrs. (On the command line, the -replace=...
	// planning option.)
	ResourceInstanceReplaceByRequest ResourceInstanceChangeActionReason = 'R'

	// ResourceInstanceReplaceByTriggers indicates that the resource instance
	// is planned to be replaced because of a corresponding change in a
	// replace_triggered_by reference.
	ResourceInstanceReplaceByTriggers ResourceInstanceChangeActionReason = 'D'

	// ResourceInstanceReplaceBecauseCannotUpdate indicates that the resource
	// instance is planned to be replaced because the provider has indicated
	// that a requested change cannot be applied as an update.
	//
	// In this case, the RequiredReplace field will typically be populated on
	// the ResourceInstanceChange object to give information about specifically
	// which arguments changed in a non-updatable way.
	ResourceInstanceReplaceBecauseCannotUpdate ResourceInstanceChangeActionReason = 'F'

	// ResourceInstanceDeleteBecauseNoResourceConfig indicates that the
	// resource instance is planned to be deleted because there's no
	// corresponding resource configuration block in the configuration.
	ResourceInstanceDeleteBecauseNoResourceConfig ResourceInstanceChangeActionReason = 'N'

	// ResourceInstanceDeleteBecauseWrongRepetition indicates that the
	// resource instance is planned to be deleted because the instance key
	// type isn't consistent with the repetition mode selected in the
	// resource configuration.
	ResourceInstanceDeleteBecauseWrongRepetition ResourceInstanceChangeActionReason = 'W'

	// ResourceInstanceDeleteBecauseCountIndex indicates that the resource
	// instance is planned to be deleted because its integer instance key
	// is out of range for the current configured resource "count" value.
	ResourceInstanceDeleteBecauseCountIndex ResourceInstanceChangeActionReason = 'C'

	// ResourceInstanceDeleteBecauseEachKey indicates that the resource
	// instance is planned to be deleted because its string instance key
	// isn't one of the keys included in the current configured resource
	// "for_each" value.
	ResourceInstanceDeleteBecauseEachKey ResourceInstanceChangeActionReason = 'E'

	// ResourceInstanceDeleteBecauseNoModule indicates that the resource
	// instance is planned to be deleted because it belongs to a module
	// instance that's no longer declared in the configuration.
	//
	// This is less specific than the reasons we return for the various ways
	// a resource instance itself can be no longer declared, including both
	// the total removal of a module block and changes to its count/for_each
	// arguments. This difference in detail is out of pragmatism, because
	// potentially multiple nested modules could all contribute conflicting
	// specific reasons for a particular instance to no longer be declared.
	ResourceInstanceDeleteBecauseNoModule ResourceInstanceChangeActionReason = 'M'

	// ResourceInstanceDeleteBecauseNoMoveTarget indicates that the resource
	// address appears as the target ("to") in a moved block, but no
	// configuration exists for that resource. According to our move rules,
	// this combination evaluates to a deletion of the "new" resource.
	ResourceInstanceDeleteBecauseNoMoveTarget ResourceInstanceChangeActionReason = 'A'

	// ResourceInstanceReadBecauseConfigUnknown indicates that the resource
	// must be read during apply (rather than during planning) because its
	// configuration contains unknown values. This reason applies only to
	// data resources.
	ResourceInstanceReadBecauseConfigUnknown ResourceInstanceChangeActionReason = '?'

	// ResourceInstanceReadBecauseDependencyPending indicates that the resource
	// must be read during apply (rather than during planning) because it
	// depends on a managed resource instance which has its own changes
	// pending.
	ResourceInstanceReadBecauseDependencyPending ResourceInstanceChangeActionReason = '!'

	// ResourceInstanceReadBecauseSmokeTest indicates that the resource
	// must be read during apply (rather than during planning) because it
	// is part of the description of a smoke test.
	ResourceInstanceReadBecauseSmokeTest ResourceInstanceChangeActionReason = 'S'
)

// OutputChange describes a change to an output value.
type OutputChange struct {
	// Addr is the absolute address of the output value that the change
	// will apply to.
	Addr addrs.AbsOutputValue

	// Change is an embedded description of the change.
	//
	// For output value changes, the type constraint for the DynamicValue
	// instances is always cty.DynamicPseudoType.
	Change

	// Sensitive, if true, indicates that either the old or new value in the
	// change is sensitive and so a rendered version of the plan in the UI
	// should elide the actual values while still indicating the action of the
	// change.
	Sensitive bool
}

// Encode produces a variant of the reciever that has its change values
// serialized so it can be written to a plan file.
func (oc *OutputChange) Encode() (*OutputChangeSrc, error) {
	cs, err := oc.Change.Encode(cty.DynamicPseudoType)
	if err != nil {
		return nil, err
	}
	return &OutputChangeSrc{
		Addr:      oc.Addr,
		ChangeSrc: *cs,
		Sensitive: oc.Sensitive,
	}, err
}

// Change describes a single change with a given action.
type Change struct {
	// Action defines what kind of change is being made.
	Action Action

	// Interpretation of Before and After depend on Action:
	//
	//     NoOp     Before and After are the same, unchanged value
	//     Create   Before is nil, and After is the expected value after create.
	//     Read     Before is any prior value (nil if no prior), and After is the
	//              value that was or will be read.
	//     Update   Before is the value prior to update, and After is the expected
	//              value after update.
	//     Replace  As with Update.
	//     Delete   Before is the value prior to delete, and After is always nil.
	//
	// Unknown values may appear anywhere within the Before and After values,
	// either as the values themselves or as nested elements within known
	// collections/structures.
	Before, After cty.Value
}

// Encode produces a variant of the reciever that has its change values
// serialized so it can be written to a plan file. Pass the type constraint
// that the values are expected to conform to; to properly decode the values
// later an identical type constraint must be provided at that time.
//
// Where a Change is embedded in some other struct, it's generally better
// to call the corresponding Encode method of that struct rather than working
// directly with its embedded Change.
func (c *Change) Encode(ty cty.Type) (*ChangeSrc, error) {
	// Storing unmarked values so that we can encode unmarked values
	// and save the PathValueMarks for re-marking the values later
	var beforeVM, afterVM []cty.PathValueMarks
	unmarkedBefore := c.Before
	unmarkedAfter := c.After

	if c.Before.ContainsMarked() {
		unmarkedBefore, beforeVM = c.Before.UnmarkDeepWithPaths()
	}
	beforeDV, err := NewDynamicValue(unmarkedBefore, ty)
	if err != nil {
		return nil, err
	}

	if c.After.ContainsMarked() {
		unmarkedAfter, afterVM = c.After.UnmarkDeepWithPaths()
	}
	afterDV, err := NewDynamicValue(unmarkedAfter, ty)
	if err != nil {
		return nil, err
	}

	return &ChangeSrc{
		Action:         c.Action,
		Before:         beforeDV,
		After:          afterDV,
		BeforeValMarks: beforeVM,
		AfterValMarks:  afterVM,
	}, nil
}
