package jsonplan

import (
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
)

// stateValues is the common representation of resolved values for both the
// prior state (which is always complete) and the planned new state.
type stateValues struct {
	Outputs    map[string]output `json:"outputs,omitempty"`
	RootModule module            `json:"root_module,omitempty"`
}

type attributeValues map[string]interface{}

func (p *plan) marshalPlannedValues(changes *plans.Changes, s *states.State) error {

	return nil
}

func marshalAttributeValues() attributeValues {
	return attributeValues{}
}
