package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/providers"
)

// ProviderEvalTree returns the evaluation tree for initializing and
// configuring providers.
func ProviderEvalTree(n *NodeApplyableProvider, config *configs.Provider) EvalNode {
	var provider providers.Interface

	addr := n.Addr

	seq := make([]EvalNode, 0, 5)
	seq = append(seq, &EvalInitProvider{
		Addr: addr,
	})

	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkValidate},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Addr:   addr,
					Output: &provider,
				},
				&EvalValidateProvider{
					Addr:     addr,
					Provider: &provider,
					Config:   config,
				},
			},
		},
	})

	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh, walkPlan, walkApply, walkDestroy, walkImport},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Addr:   addr,
					Output: &provider,
				},
			},
		},
	})

	// We configure on everything but validate, since validate may
	// not have access to all the variables.
	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh, walkPlan, walkApply, walkDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalConfigProvider{
					Addr:     addr,
					Provider: &provider,
					Config:   config,
				},
			},
		},
	})
	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkImport},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalConfigProvider{
					Addr:                addr,
					Provider:            &provider,
					Config:              config,
					VerifyConfigIsKnown: true,
				},
			},
		},
	})

	return &EvalSequence{Nodes: seq}
}

// CloseProviderEvalTree returns the evaluation tree for closing
// provider connections that aren't needed anymore.
func CloseProviderEvalTree(addr addrs.AbsProviderConfig) EvalNode {
	return &EvalCloseProvider{Addr: addr}
}
