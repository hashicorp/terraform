package plans

import (
	"sort"

	"github.com/hashicorp/terraform/addrs"
)

// Plan is the top-level type representing a planned set of changes.
//
// A plan is a summary of the set of changes required to move from a current
// state to a goal state derived from configuration. The described changes
// are not applied directly, but contain an approximation of the final
// result that will be completed during apply by resolving any values that
// cannot be predicted.
//
// A plan must always be accompanied by the state and configuration it was
// built from, since the plan does not itself include all of the information
// required to make the changes indicated.
type Plan struct {
	VariableValues  map[string]DynamicValue
	Changes         *Changes
	ProviderSHA256s map[string][]byte
	Backend         Backend
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
