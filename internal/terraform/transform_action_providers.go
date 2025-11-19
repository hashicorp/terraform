// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
)

type ActionProviderTransformer struct {
	Changes *plans.ChangesSrc
	Config  *configs.Config
}

// for all GraphNodeProviderConsumer (node resource abs)
// if it's a managed resource
//	if it has a planned action
//	add the action provider to the ActionsProvidersByThingName()

// then the provider transformer will get an additional
// ActionsProvidedBy() step
// this could be part of the provider transformers func

func (t *ActionProviderTransformer) Transform(g *Graph) error {

	return nil
}
