package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// GraphNodeVariable is the interface that must be implemented by anything
// that is a variable.
type GraphNodeVariable interface {
	VariableName() string
	SetVariableValue(*config.RawConfig)
}

// GraphNodeConfigVariable represents a Variable in the config.
type GraphNodeConfigVariable struct {
	Variable *config.Variable

	// Value, if non-nil, will be used to set the value of the variable
	// during evaluation. If this is nil, evaluation will do nothing.
	Value *config.RawConfig
}

func (n *GraphNodeConfigVariable) Name() string {
	return fmt.Sprintf("var.%s", n.Variable.Name)
}

func (n *GraphNodeConfigVariable) ConfigType() GraphNodeConfigType {
	return GraphNodeConfigTypeVariable
}

func (n *GraphNodeConfigVariable) DependableName() []string {
	return []string{n.Name()}
}

func (n *GraphNodeConfigVariable) DependentOn() []string {
	return nil
}

// GraphNodeVariable impl.
func (n *GraphNodeConfigVariable) VariableName() string {
	return n.Variable.Name
}

// GraphNodeVariable impl.
func (n *GraphNodeConfigVariable) SetVariableValue(v *config.RawConfig) {
	n.Value = v
}
