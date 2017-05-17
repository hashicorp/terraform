package terraform

import (
	"fmt"
)

// NodeDisabledProvider represents a provider that is disabled. A disabled
// provider does nothing. It exists to properly set inheritance information
// for child providers.
type NodeDisabledProvider struct {
	*NodeAbstractProvider
}

func (n *NodeDisabledProvider) Name() string {
	return fmt.Sprintf("%s (disabled)", n.NodeAbstractProvider.Name())
}

// GraphNodeEvalable
func (n *NodeDisabledProvider) EvalTree() EvalNode {
	var resourceConfig *ResourceConfig
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalInterpolate{
				Config: n.ProviderConfig(),
				Output: &resourceConfig,
			},
			&EvalBuildProviderConfig{
				Provider: n.ProviderName(),
				Config:   &resourceConfig,
				Output:   &resourceConfig,
			},
			&EvalSetProviderConfig{
				Provider: n.ProviderName(),
				Config:   &resourceConfig,
			},
		},
	}
}
