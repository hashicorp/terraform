package terraform

// NodeCountBoundary fixes any "count boundarie" in the state: resources
// that are named "foo.0" when they should be named "foo"
type NodeCountBoundary struct{}

func (n *NodeCountBoundary) Name() string {
	return "meta.count-boundary (count boundary fixup)"
}

// GraphNodeEvalable
func (n *NodeCountBoundary) EvalTree() EvalNode {
	return &EvalCountFixZeroOneBoundaryGlobal{}
}
