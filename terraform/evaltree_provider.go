package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// ProviderEvalTree returns the evaluation tree for initializing and
// configuring providers.
func ProviderEvalTree(n string, config *config.RawConfig) EvalNode {
	var provider ResourceProvider
	var resourceConfig *ResourceConfig

	seq := make([]EvalNode, 0, 5)
	seq = append(seq, &EvalInitProvider{Name: n})

	// Input stuff
	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkInput},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n,
					Output: &provider,
				},
				&EvalInputProvider{
					Name:     n,
					Provider: &provider,
					Config:   config,
				},
			},
		},
	})

	// Apply stuff
	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkValidate, walkRefresh, walkPlan, walkApply},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n,
					Output: &provider,
				},
				&EvalInterpolate{
					Config: config,
					Output: &resourceConfig,
				},
				&EvalValidateProvider{
					ProviderName: n,
					Provider:     &provider,
					Config:       &resourceConfig,
				},
				&EvalConfigProvider{
					Provider: n,
					Config:   &resourceConfig,
				},
			},
		},
	})

	return &EvalSequence{Nodes: seq}
}
