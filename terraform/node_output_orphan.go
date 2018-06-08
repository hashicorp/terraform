package terraform

import (
	"fmt"
)

// NodeOutputOrphan represents an output that is an orphan.
type NodeOutputOrphan struct {
	OutputName string
	PathValue  []string
}

func (n *NodeOutputOrphan) Name() string {
	result := fmt.Sprintf("output.%s (orphan)", n.OutputName)
	if len(n.PathValue) > 1 {
		result = fmt.Sprintf("%s.%s", modulePrefixStr(n.PathValue), result)
	}

	return result
}

// GraphNodeReferenceable
func (n *NodeOutputOrphan) ReferenceableName() []string {
	return []string{"output." + n.OutputName}
}

// GraphNodeSubPath
func (n *NodeOutputOrphan) Path() []string {
	return n.PathValue
}

// GraphNodeEvalable
func (n *NodeOutputOrphan) EvalTree() EvalNode {
	return &EvalOpFilter{
		Ops: []walkOperation{walkRefresh, walkApply, walkDestroy},
		Node: &EvalDeleteOutput{
			Name: n.OutputName,
		},
	}
}
