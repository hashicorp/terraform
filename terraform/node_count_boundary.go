package terraform

// NodeCountBoundary fixes any "count boundaries" in the state: resources
// that are named "foo.0" when they should be named "foo"
type NodeCountBoundary struct{}

func (n *NodeCountBoundary) Name() string {
	return "meta.count-boundary (count boundary fixup)"
}

// GraphNodeEvaluable
func (n *NodeCountBoundary) EvalTree() EvalNode {
	return &EvalCountFixZeroOneBoundaryGlobal{}
}
