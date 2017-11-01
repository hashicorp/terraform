package terraform

import (
	"strings"

	"github.com/hashicorp/terraform/config"
)

// ProviderEvalTree returns the evaluation tree for initializing and
// configuring providers.
func ProviderEvalTree(n *NodeApplyableProvider, config *config.ProviderConfig) EvalNode {
	var provider ResourceProvider
	var resourceConfig *ResourceConfig

	typeName := strings.SplitN(n.NameValue, ".", 2)[0]

	seq := make([]EvalNode, 0, 5)
	seq = append(seq, &EvalInitProvider{
		TypeName: typeName,
		Name:     n.Name(),
	})

	// Input stuff
	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkInput, walkImport},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n.Name(),
					Output: &provider,
				},
				&EvalInterpolateProvider{
					Config: config,
					Output: &resourceConfig,
				},
				&EvalBuildProviderConfig{
					Provider: n.NameValue,
					Config:   &resourceConfig,
					Output:   &resourceConfig,
				},
				&EvalInputProvider{
					Name:     n.NameValue,
					Provider: &provider,
					Config:   &resourceConfig,
				},
			},
		},
	})

	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkValidate},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n.Name(),
					Output: &provider,
				},
				&EvalInterpolateProvider{
					Config: config,
					Output: &resourceConfig,
				},
				&EvalBuildProviderConfig{
					Provider: n.NameValue,
					Config:   &resourceConfig,
					Output:   &resourceConfig,
				},
				&EvalValidateProvider{
					Provider: &provider,
					Config:   &resourceConfig,
				},
			},
		},
	})

	// Apply stuff
	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh, walkPlan, walkApply, walkDestroy, walkImport},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n.Name(),
					Output: &provider,
				},
				&EvalInterpolateProvider{
					Config: config,
					Output: &resourceConfig,
				},
				&EvalBuildProviderConfig{
					Provider: n.NameValue,
					Config:   &resourceConfig,
					Output:   &resourceConfig,
				},
			},
		},
	})

	// We configure on everything but validate, since validate may
	// not have access to all the variables.
	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh, walkPlan, walkApply, walkDestroy, walkImport},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalConfigProvider{
					Provider: n.Name(),
					Config:   &resourceConfig,
				},
			},
		},
	})

	return &EvalSequence{Nodes: seq}
}

// CloseProviderEvalTree returns the evaluation tree for closing
// provider connections that aren't needed anymore.
func CloseProviderEvalTree(n string) EvalNode {
	return &EvalCloseProvider{Name: n}
}
