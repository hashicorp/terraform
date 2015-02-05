package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// ProviderEvalTree returns the evaluation tree for initializing and
// configuring providers.
func ProviderEvalTree(n string, config *config.RawConfig) EvalNode {
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalInitProvider{Name: n},
			&EvalValidateProvider{
				Provider: &EvalGetProvider{Name: n},
				Config:   &EvalInterpolate{Config: config},
			},
			&EvalConfigProvider{
				Provider: &EvalGetProvider{Name: n},
				Config:   &EvalInterpolate{Config: config},
			},
		},
	}
}
