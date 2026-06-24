// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import "github.com/hashicorp/terraform/internal/configs"

type ProviderRequirementExprTransformer struct {
	Config *configs.Config
}

var _ GraphTransformer = (*ProviderRequirementExprTransformer)(nil)

func (t *ProviderRequirementExprTransformer) Transform(g *Graph) error {
	if len(t.Config.Module.ProviderRequirements.RequiredProviders) == 0 {
		return nil
	}

	node := &nodeResolveProviderRequirements{
		Addr:   g.Path,
		Module: t.Config.Module,
	}

	g.Add(node)

	return nil
}
