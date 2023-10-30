// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package refactoring

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type RemoveStatement struct {
	From      *addrs.ConfigResource
	Destroy   bool
	DeclRange tfdiags.SourceRange
}

// FindRemoveStatements recurses through the modules of the given configuration
// and returns a flat set of all "removed" blocks defined within, in a
// deterministic but undefined order.
func FindRemoveStatements(rootCfg *configs.Config) []RemoveStatement {
	return findRemoveStatements(rootCfg, nil)
}

func findRemoveStatements(cfg *configs.Config, into []RemoveStatement) []RemoveStatement {
	for _, mc := range cfg.Module.Removed {

		// FIXME KEM: Only works for resources right now
		from := mc.From.ConfigMoveable(cfg.Path)
		if fromR, isRes := from.(addrs.ConfigResource); isRes {
			into = append(into, RemoveStatement{
				From: &addrs.ConfigResource{
					Module:   cfg.Path,
					Resource: fromR.Resource,
				},
				Destroy:   mc.Destroy,
				DeclRange: tfdiags.SourceRangeFromHCL(mc.DeclRange),
			})
		} else {
			panic("sorry")
		}

	}

	for _, childCfg := range cfg.Children {
		into = findRemoveStatements(childCfg, into)
	}

	return into
}
