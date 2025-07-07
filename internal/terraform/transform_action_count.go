// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/providers"
)

// ActionCountTransformer is a GraphTransformer that expands the count
// out for a specific action.
//
// This assumes that the count is already interpolated.
type ActionCountTransformer struct {
	Schema *providers.ActionSchema
	Config configs.Action

	Addr             addrs.ConfigAction
	InstanceAddrs    []addrs.AbsActionInstance
	ResolvedProvider addrs.AbsProviderConfig
}

func (t *ActionCountTransformer) Transform(g *Graph) error {
	for _, addr := range t.InstanceAddrs {
		node := NodeActionDeclarationInstance{
			Addr:             addr,
			Config:           t.Config,
			Schema:           t.Schema,
			ResolvedProvider: t.ResolvedProvider,
		}

		log.Printf("[TRACE] ActionCountTransformer: adding %s", addr)
		g.Add(&node)
	}
	return nil
}
