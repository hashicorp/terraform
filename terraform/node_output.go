package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
)

// NodeApplyableOutput represents an output that is "applyable":
// it is ready to be applied.
type NodeApplyableOutput struct {
	PathValue []string
	Config    *config.Output // Config is the output in the config
}

func (n *NodeApplyableOutput) Name() string {
	result := fmt.Sprintf("output.%s", n.Config.Name)
	if len(n.PathValue) > 1 {
		result = fmt.Sprintf("%s.%s", modulePrefixStr(n.PathValue), result)
	}

	return result
}

// GraphNodeSubPath
func (n *NodeApplyableOutput) Path() []string {
	return n.PathValue
}

// RemovableIfNotTargeted
func (n *NodeApplyableOutput) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// GraphNodeReferenceable
func (n *NodeApplyableOutput) ReferenceableName() []string {
	name := fmt.Sprintf("output.%s", n.Config.Name)
	return []string{name}
}

// GraphNodeReferencer
func (n *NodeApplyableOutput) References() []string {
	var result []string
	result = append(result, n.Config.DependsOn...)
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
func (n *NodeApplyableOutput) EvalTree() EvalNode {
	return &EvalOpFilter{
		Ops: []walkOperation{walkRefresh, walkPlan, walkApply,
			walkDestroy, walkInput, walkValidate},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalWriteOutput{
					Name:      n.Config.Name,
					Sensitive: n.Config.Sensitive,
					Value:     n.Config.RawConfig,
				},
			},
		},
	}
}
