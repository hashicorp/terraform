package jsonstate

import (
	"encoding/json"

	"github.com/hashicorp/terraform/states"
)

// FormatVersion represents the version of the json format and will be
// incremented for any change to this format that requires changes to a
// consuming parser.
const FormatVersion = "0.1"

// state is the top-level representation of the json format of a terraform
// state.
type state struct {
	FormatVersion string      `json:"format_version"`
	Values        stateValues `json:"values"`
}

// stateValues is the common representation of resolved values for both the prior
// state (which is always complete) and the planned new state.
type stateValues struct {
	Outputs    map[string]output
	RootModule module
}

type output struct {
	Sensitive bool
	Value     json.RawMessage
}

// module is the representation of a module in state. This can be the root module
// or a child module
type module struct {
	Resources []resource

	// Address is the absolute module address, omitted for the root module
	Address string `json:"address,omitempty"`

	// Each module object can optionally have its own nested "child_modules",
	// recursively describing the full module tree.
	ChildModules []module `json:"child_modules,omitempty"`
}

type moduleCall struct {
	ResolvedSource    string                 `json:"resolved_source"`
	Expressions       map[string]interface{} `json:"expressions,omitempty"`
	CountExpression   expression             `json:"count_expression"`
	ForEachExpression expression             `json:"for_each_expression"`
	Module            module                 `json:"module"`
}

// Resource is the representation of a resource in the state.
type resource struct {
	// Address is the absolute resource address
	Address string `json:"address"`

	// Mode can be "managed" or "data"
	Mode string `json:"mode"`

	Type string `json:"type"`
	Name string `json:"name"`

	// Index is omitted for a resource not using `count` or `for_each`.
	Index int `json:"index,omitempty"`

	// ProviderName allows the property "type" to be interpreted unambiguously
	// in the unusual situation where a provider offers a resource type whose
	// name does not start with its own name, such as the "googlebeta" provider
	// offering "google_compute_instance".
	ProviderName string `json:"provider_name"`

	// SchemaVersion indicates which version of the resource type schema the
	// "values" property conforms to.
	SchemaVersion int `json:"schema_version"`

	// Values is the JSON representation of the attribute values of the
	// resource, whose structure depends on the resource type schema. Any
	// unknown values are omitted or set to null, making them indistinguishable
	// from absent values.
	Values json.RawMessage `json:"values"`
}

type source struct {
	FileName string `json:"filename"`
	Start    string `json:"start"`
	End      string `json:"end"`
}

// newState() returns a minimally-initialized state
func newState() *state {
	return &state{
		FormatVersion: FormatVersion,
	}
}

// Marshal returns the json encoding of a terraform plan.
func Marshal(s *states.State) ([]byte, error) {
	if s.Empty() {
		return nil, nil
	}

	output := newState()

	ret, err := json.Marshal(output)
	return ret, err
}
