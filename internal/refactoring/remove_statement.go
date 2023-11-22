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

// RemoveStatement is the fully-specified form of addrs.Remove
type RemoveStatement struct {
	// From is the absolute address of the configuration object being removed.
	From addrs.ConfigMoveable

	// Destroy indicates that the resource should be destroyed, not just removed
	// from state.
	Destroy   bool
	DeclRange tfdiags.SourceRange
}

// FindRemoveStatements recurses through the modules of the given configuration
// and returns a set of all "removed" blocks defined within after deduplication
// on the From address.
//
// Error diagnostics are returned if any resource or module targeted by a remove
// block is still defined in configuration.
//
// A "removed" block in a parent module overrides a removed block in a child
// module when both target the same configuration object.
func FindRemoveStatements(rootCfg *configs.Config) (addrs.Map[addrs.ConfigMoveable, RemoveStatement], tfdiags.Diagnostics) {
	return findRemoveStatements(rootCfg, addrs.MakeMap[addrs.ConfigMoveable, RemoveStatement]())
}

func findRemoveStatements(cfg *configs.Config, into addrs.Map[addrs.ConfigMoveable, RemoveStatement]) (addrs.Map[addrs.ConfigMoveable, RemoveStatement], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	for _, mc := range cfg.Module.Removed {
		switch mc.From.ObjectKind() {
		case addrs.RemoveTargetResource:
			// First, stitch together the module path and the RelSubject to form
			// the absolute address of the config object being removed.
			res := mc.From.RelSubject.(addrs.ConfigResource)
			fromAddr := addrs.ConfigResource{
				Module:   append(cfg.Path, res.Module...),
				Resource: res.Resource,
			}

			// If we already have a remove statement for this ConfigResource, it
			// must have come from a parent module, because duplicate removed
			// blocks in the same module are ignored during parsing.
			// The removed block in the parent module overrides the block in the
			// child module.
			existingStatement, ok := into.GetOk(fromAddr)
			if ok {
				if existingResource, ok := existingStatement.From.(addrs.ConfigResource); ok &&
					existingResource.Equal(fromAddr) {
					continue
				}
			}

			into.Put(fromAddr, RemoveStatement{
				From:      fromAddr,
				Destroy:   mc.Destroy,
				DeclRange: tfdiags.SourceRangeFromHCL(mc.DeclRange),
			})
		case addrs.RemoveTargetModule:
			// First, stitch together the module path and the RelSubject to form
			// the absolute address of the config object being removed.
			mod := mc.From.RelSubject.(addrs.Module)
			absMod := append(cfg.Path, mod...)

			// If there is already a statement for this Module, it must
			// have come from a parent module, because duplicate removed blocks
			// in the same module are ignored during parsing.
			// The removed block in the parent module overrides the block in the
			// child module.
			existingStatement, ok := into.GetOk(mc.From.RelSubject)
			if ok {
				if existingModule, ok := existingStatement.From.(addrs.Module); ok &&
					existingModule.Equal(absMod) {
					continue
				}
			}

			into.Put(absMod, RemoveStatement{
				From:      absMod,
				Destroy:   mc.Destroy,
				DeclRange: tfdiags.SourceRangeFromHCL(mc.DeclRange),
			})
		default:
			panic("Unsupported remove target kind")
		}
	}

	for _, childCfg := range cfg.Children {
		into, diags = findRemoveStatements(childCfg, into)
	}

	return into, diags
}
