package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
)

// ProviderEvalTree returns the evaluation tree for initializing and
// configuring providers.
func ProviderEvalTree(n *NodeApplyableProvider, config *configs.Provider) EvalNode {
	var provider ResourceProvider

	addr := n.Addr
	relAddr := addr.ProviderConfig

	seq := make([]EvalNode, 0, 5)
	seq = append(seq, &EvalInitProvider{
		TypeName: relAddr.Type,
		Addr:     addr.ProviderConfig,
	})

	// Input stuff
	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkImport},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Addr:   addr,
					Output: &provider,
				},
			},
		},
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
					Addr:     relAddr,
					Provider: &provider,
					Config:   config,
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
					Addr:   addr,
					Output: &provider,
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
					Addr:     relAddr,
					Provider: &provider,
					Config:   config,
				},
			},
		},
	})

	return &EvalSequence{Nodes: seq}
}

// CloseProviderEvalTree returns the evaluation tree for closing
// provider connections that aren't needed anymore.
func CloseProviderEvalTree(addr addrs.AbsProviderConfig) EvalNode {
	return &EvalCloseProvider{Addr: addr.ProviderConfig}
}
