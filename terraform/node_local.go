package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
)

// NodeLocal represents a named local value in a particular module.
//
// Local value nodes only have one operation, common to all walk types:
// evaluate the result and place it in state.
type NodeLocal struct {
	PathValue []string
	Config    *config.Local
}

func (n *NodeLocal) Name() string {
	result := fmt.Sprintf("local.%s", n.Config.Name)
	if len(n.PathValue) > 1 {
		result = fmt.Sprintf("%s.%s", modulePrefixStr(n.PathValue), result)
	}

	return result
}

// GraphNodeSubPath
func (n *NodeLocal) Path() []string {
	return n.PathValue
}

// RemovableIfNotTargeted
func (n *NodeLocal) RemoveIfNotTargeted() bool {
	return true
}

// GraphNodeReferenceable
func (n *NodeLocal) ReferenceableName() []string {
	name := fmt.Sprintf("local.%s", n.Config.Name)
	return []string{name}
}

// GraphNodeReferencer
func (n *NodeLocal) References() []string {
	var result []string
	result = append(result, ReferencesFromConfig(n.Config.RawConfig)...)
	for _, v := range result {
		split := strings.Split(v, "/")
		for i, s := range split {
			split[i] = s + ".destroy"
		}

		result = append(result, strings.Join(split, "/"))
	}

	return result
}

// GraphNodeEvalable
func (n *NodeLocal) EvalTree() EvalNode {
	return &EvalLocal{
		Name:  n.Config.Name,
		Value: n.Config.RawConfig,
	}
}
