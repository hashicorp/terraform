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
		if target.AddrType() == addrs.AbsActionAddrType ||
			target.AddrType() == addrs.AbsActionInstanceAddrType {
			v := nodeActionInvoke{Target: target}
			g.Add(&v)
		}
	}

	return nil
}
