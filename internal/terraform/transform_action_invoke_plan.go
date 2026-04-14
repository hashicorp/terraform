// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

type ActionInvokePlanTransformer struct {
	Config        *configs.Config
	ActionTargets []addrs.Targetable
	Operation     walkOperation

	queryPlanMode bool
}

func (t *ActionInvokePlanTransformer) Transform(g *Graph) error {
	if t.Operation != walkPlan || t.queryPlanMode || len(t.ActionTargets) == 0 {
		return nil
	}

	for _, v := range g.Vertices() {
		actionNode, ok := v.(*NodeActionConfig)
		if !ok {
			continue
		}

		for _, target := range t.ActionTargets {
			// we need to create the invoke node in the correct module scope for each target
			var targetAddr addrs.ConfigAction
			var instAddr addrs.AbsActionInstance

			switch target := target.(type) {
			case addrs.AbsActionInstance:
				targetAddr = target.ConfigAction()
				instAddr = target
			case addrs.AbsAction:
				targetAddr = target.Config()
				instAddr = target.Instance(addrs.NoKey)
			default:
				panic(fmt.Sprintf("invalid action addr: %#v", target))
			}

			if !actionNode.Addr.Equal(targetAddr) {
				continue
			}

			g.Add(&nodeActionInvokeExpand{
				Target:       target,
				Module:       targetAddr.Module,
				Addr:         instAddr,
				ActionConfig: actionNode,
			})

		}
	}

	return nil

}
