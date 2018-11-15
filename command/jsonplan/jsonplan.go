package jsonplan

import (
	"encoding/json"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
)

// FormatVersion represents the version of the json format and will be
// incremented for any change to this format that requires changes to a
// consuming parser.
const FormatVersion = "0.1"

// Plan is the top-level representation of the json format of a plan. It includes
// the complete config and current state.
type plan struct {
	FormatVersion   string            `json:"format_version,omitempty"`
	PriorState      json.RawMessage   `json:"prior_state,omitempty"`
	Config          config            `json:"configuration,omitempty"`
	PlannedValues   values            `json:"planned_values,omitempty"`
	ProposedUnknown values            `json:"proposed_unknown,omitempty"`
	ResourceChanges []resourceChange  `json:"resource_changes,omitempty"`
	OutputChanges   map[string]change `json:"output_changes,omitempty"`
}

func newPlan() *plan {
	return &plan{
		FormatVersion: FormatVersion,
	}
}

// Change is the representation of a proposed change for an object.
type change struct {
	// Actions are the actions that will be taken on the object selected by the
	// properties below. Valid actions values are:
	//    ["no-op"]
	//    ["create"]
	//    ["read"]
	//    ["update"]
	//    ["delete", "create"]
	//    ["create", "delete"]
	//    ["delete"]
	// The two "replace" actions are represented in this way to allow callers to
	// e.g. just scan the list for "delete" to recognize all three situations
	// where the object will be deleted, allowing for any new deletion
	// combinations that might be added in future.
	Actions []string `json:"actions,omitempty"`

	// Before and After are representations of the object value both before and
	// after the action. For ["create"] and ["delete"] actions, either "before"
	// or "after" is unset (respectively). For ["no-op"], the before and after
	// values are identical. The "after" value will be incomplete if there are
	// values within it that won't be known until after apply.
	Before json.RawMessage `json:"before,omitempty"`
	After  json.RawMessage `json:"after,omitempty"`
}

// Values is the common representation of resolved values for both the prior
// state (which is always complete) and the planned new state.
type values struct {
	Outputs    map[string]output `json:"outputs,omitempty"`
	RootModule module            `json:"root_module,omitempty"`
}

type output struct {
	Sensitive bool            `json:"sensitive,omitempty"`
	Value     json.RawMessage `json:"value,omitempty"`
}

type source struct {
	FileName string `json:"filename,omitempty"`
	Start    string `json:"start,omitempty"`
	End      string `json:"end,omitempty"`
}

// Marshall returns the json encoding of a terraform plan.
func Marshall(c *configload.Snapshot, p *plans.Plan, s *states.State) ([]byte, error) {
	output := newPlan()

	// output.Config = config{
	// 	ProviderConfigs: []providerConfig{},
	// 	RootModule:      configRootModule{},
	// }
	// output.OutputChanges =
	output.marshallOutputChanges(p.Changes)
	// output.PlannedValues
	// output.PriorState
	// output.ProposedUnknown
	// output.ResourceChanges = marshalResourceChanges(p)
	output.marshallResourceChanges(p.Changes)
	ret, err := json.MarshalIndent(output, "", "  ")
	return ret, err
}

func (p *plan) marshallResourceChanges(changes *plans.Changes) {
	if changes == nil {
		// Nothing to do!
		return
	}
	for _, rc := range changes.Resources {
		var r resourceChange
		addr := rc.Addr
		dataSource := addr.Resource.Resource.Mode == addrs.DataResourceMode

		// We create "delete" actions for data resources so we can clean
		// up their entries in state, but this is an implementation detail
		// that users shouldn't see.
		if dataSource && rc.Action == plans.Delete {
			continue
		}

		r.Address = addr.String()

		r.Change = change{
			Actions: []string{rc.Action.String()},
			Before:  json.RawMessage(rc.Before),
			// After:   json.RawMessage(rc.After),
		}
		r.Deposed = rc.DeposedKey == states.NotDeposed

		key := addr.Resource.Key
		if key != nil {
			r.Index = key.String()
		}
		r.Mode = addr.Resource.Resource.Mode.String()
		r.ModuleAddress = addr.Module.String()
		r.Name = addr.Resource.Resource.Name
		r.Type = addr.Resource.Resource.Type

		p.ResourceChanges = append(p.ResourceChanges, r)

	}
}

func (p *plan) marshallOutputChanges(changes *plans.Changes) {
	if changes == nil {
		// Nothing to do!
		return
	}

	var c change
	for _, oc := range changes.Outputs {
		c.Actions = []string{oc.Action.String()}
		c.Before = json.RawMessage(oc.Before)
		c.After = json.RawMessage(oc.After)
		p.OutputChanges[oc.Addr.String()] = c
	}

}
