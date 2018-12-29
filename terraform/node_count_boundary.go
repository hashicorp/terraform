package terraform

import (
	"github.com/hashicorp/terraform/configs"
)

// NodeCountBoundary fixes up any transitions between "each modes" in objects
// saved in state, such as switching from NoEach to EachInt.
type NodeCountBoundary struct {
	Config *configs.Config
}

func (n *NodeCountBoundary) Name() string {
	return "meta.count-boundary (EachMode fixup)"
}

// GraphNodeEvalable
func (n *NodeCountBoundary) EvalTree() EvalNode {
	return &EvalCountFixZeroOneBoundaryGlobal{
		Config: n.Config,
	}
}
