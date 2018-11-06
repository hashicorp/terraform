package jsonplan

import (
	"encoding/json"

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
	FormatVersion   string            `json:"format_version"`
	PriorState      json.RawMessage   `json:"prior_state,omitempty"`
	Config          config            `json:"configuration"`
	PlannedValues   values            `json:"planned_values"`
	ProposedUnknown values            `json:"proposed_unknown"`
	ResourceChanges []resourceChange  `json:"resource_changes"`
	OutputChanges   map[string]change `json:"output_changes"`
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
	Actions []string

	// Before and After are representations of the object value both before and
	// after the action. For ["create"] and ["delete"] actions, either "before"
	// or "after" is unset (respectively). For ["no-op"], the before and after
	// values are identical. The "after" value will be incomplete if there are
	// values within it that won't be known until after apply.
	Before json.RawMessage
	After  json.RawMessage
}

// Values is the common representation of resolved values for both the prior
// state (which is always complete) and the planned new state.
type values struct {
	Outputs    map[string]output
	RootModule module
}

type output struct {
	Sensitive bool
	Value     json.RawMessage
}

type source struct {
	FileName string `json:"filename"`
	Start    string `json:"start"`
	End      string `json:"end"`
}

// Marshall returns the json encoding of a terraform plan.
func Marshall(c *configload.Snapshot, p *plans.Plan, s *states.State) ([]byte, error) {
	return nil, nil
}
