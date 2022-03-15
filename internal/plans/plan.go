package plans

import (
	"sort"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

// Plan is the top-level type representing a planned set of changes.
//
// A plan is a summary of the set of changes required to move from a current
// state to a goal state derived from configuration. The described changes
// are not applied directly, but contain an approximation of the final
// result that will be completed during apply by resolving any values that
// cannot be predicted.
//
// A plan must always be accompanied by the configuration it was built from,
// since the plan does not itself include all of the information required to
// make the changes indicated.
type Plan struct {
	// Mode is the mode under which this plan was created.
	//
	// This is only recorded to allow for UI differences when presenting plans
	// to the end-user, and so it must not be used to influence apply-time
	// behavior. The actions during apply must be described entirely by
	// the Changes field, regardless of how the plan was created.
	UIMode Mode

	VariableValues    map[string]DynamicValue
	Changes           *Changes
	DriftedResources  []*ResourceInstanceChangeSrc
	TargetAddrs       []addrs.Targetable
	ForceReplaceAddrs []addrs.AbsResourceInstance
	Backend           Backend

	// PrevRunState and PriorState both describe the situation that the plan
	// was derived from:
	//
	// PrevRunState is a representation of the outcome of the previous
	// Terraform operation, without any updates from the remote system but
	// potentially including some changes that resulted from state upgrade
	// actions.
	//
	// PriorState is a representation of the current state of remote objects,
	// which will differ from PrevRunState if the "refresh" step returned
	// different data, which might reflect drift.
	//
	// PriorState is the main snapshot we use for actions during apply.
	// PrevRunState is only here so that we can diff PriorState against it in
	// order to report to the user any out-of-band changes we've detected.
	PrevRunState *states.State
	PriorState   *states.State
}

// CanApply returns true if and only if the recieving plan includes content
// that would make sense to apply. If it returns false, the plan operation
// should indicate that there's nothing to do and Terraform should exit
// without prompting the user to confirm the changes.
//
// This function represents our main business logic for making the decision
// about whether a given plan represents meaningful "changes", and so its
// exact definition may change over time; the intent is just to centralize the
// rules for that rather than duplicating different versions of it at various
// locations in the UI code.
func (p *Plan) CanApply() bool {
	switch {
	case !p.Changes.Empty():
		// "Empty" means that everything in the changes is a "NoOp", so if
		// not empty then there's at least one non-NoOp change.
		return true

	case !p.PriorState.ManagedResourcesEqual(p.PrevRunState):
		// If there are no changes planned but we detected some
		// outside-Terraform changes while refreshing then we consider
		// that applyable in isolation only if this was a refresh-only
		// plan where we expect updating the state to include these
		// changes was the intended goal.
		//
		// (We don't treat a "refresh only" plan as applyable in normal
		// planning mode because historically the refresh result wasn't
		// considered part of a plan at all, and so it would be
		// a disruptive breaking change if refreshing alone suddenly
		// became applyable in the normal case and an existing configuration
		// was relying on ignore_changes in order to be convergent in spite
		// of intentional out-of-band operations.)
		return p.UIMode == RefreshOnlyMode

	default:
		// Otherwise, there are either no changes to apply or they are changes
		// our cases above don't consider as worthy of applying in isolation.
		return false
	}
}

// ProviderAddrs returns a list of all of the provider configuration addresses
// referenced throughout the receiving plan.
//
// The result is de-duplicated so that each distinct address appears only once.
func (p *Plan) ProviderAddrs() []addrs.AbsProviderConfig {
	if p == nil || p.Changes == nil {
		return nil
	}

	m := map[string]addrs.AbsProviderConfig{}
	for _, rc := range p.Changes.Resources {
		m[rc.ProviderAddr.String()] = rc.ProviderAddr
	}
	if len(m) == 0 {
		return nil
	}

	// This is mainly just so we'll get stable results for testing purposes.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ret := make([]addrs.AbsProviderConfig, len(keys))
	for i, key := range keys {
		ret[i] = m[key]
	}

	return ret
}

// Backend represents the backend-related configuration and other data as it
// existed when a plan was created.
type Backend struct {
	// Type is the type of backend that the plan will apply against.
	Type string

	// Config is the configuration of the backend, whose schema is decided by
	// the backend Type.
	Config DynamicValue

	// Workspace is the name of the workspace that was active when the plan
	// was created. It is illegal to apply a plan created for one workspace
	// to the state of another workspace.
	// (This constraint is already enforced by the statefile lineage mechanism,
	// but storing this explicitly allows us to return a better error message
	// in the situation where the user has the wrong workspace selected.)
	Workspace string
}

func NewBackend(typeName string, config cty.Value, configSchema *configschema.Block, workspaceName string) (*Backend, error) {
	dv, err := NewDynamicValue(config, configSchema.ImpliedType())
	if err != nil {
		return nil, err
	}

	return &Backend{
		Type:      typeName,
		Config:    dv,
		Workspace: workspaceName,
	}, nil
}
