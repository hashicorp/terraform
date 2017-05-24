package terraform

import (
	"fmt"

	"github.com/r3labs/terraform/config"
)

// NodeRootVariable represents a root variable input.
type NodeRootVariable struct {
	Config *config.Variable
}

func (n *NodeRootVariable) Name() string {
	result := fmt.Sprintf("var.%s", n.Config.Name)
	return result
}

// GraphNodeReferenceable
func (n *NodeRootVariable) ReferenceableName() []string {
	return []string{n.Name()}
}
