package terraform

import (
	"fmt"
)

// NodeDestroyableModule represents a module destruction.
type NodeDestroyableModuleVariable struct {
	PathValue []string
}

func (n *NodeDestroyableModuleVariable) Name() string {
	result := "plan-destroy"
	if len(n.PathValue) > 1 {
		result = fmt.Sprintf("%s.%s", modulePrefixStr(n.PathValue), result)
	}

	return result
}

// GraphNodeSubPath
func (n *NodeDestroyableModuleVariable) Path() []string {
	return n.PathValue
}

// GraphNodeEvalable
func (n *NodeDestroyableModuleVariable) EvalTree() EvalNode {
	return &EvalDiffDestroyModule{Path: n.PathValue}
}
