package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// GraphNodeConfigVariable represents a Variable in the config.
type GraphNodeConfigVariable struct {
	Variable *config.Variable
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
