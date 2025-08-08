// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

type ActionInvokeTransformer struct {
	Targets []addrs.Targetable
}

func (t *ActionInvokeTransformer) Transform(g *Graph) error {
	for _, target := range t.Targets {
		if target.AddrType() == addrs.AbsActionAddrType {
			aaiTarget := target.(addrs.AbsAction).Instance(addrs.NoKey)
			v := nodeActionInvoke{Target: aaiTarget}
			g.Add(&v)
		} else if target.AddrType() == addrs.AbsActionInstanceAddrType {
			aaiTarget := target.(addrs.AbsActionInstance)
			v := nodeActionInvoke{Target: aaiTarget}
			g.Add(&v)
		}
	}

	return nil
}
