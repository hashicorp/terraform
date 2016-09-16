package terraform

import (
	"fmt"

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

// GraphNodeReferenceable
func (n *NodeApplyableOutput) ReferenceableName() []string {
	name := fmt.Sprintf("output.%s", n.Config.Name)
	return []string{name}
}

// GraphNodeReferencer
func (n *NodeApplyableOutput) References() []string {
	var result []string
	result = append(result, ReferencesFromConfig(n.Config.RawConfig)...)
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
