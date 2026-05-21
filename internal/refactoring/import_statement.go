// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package refactoring

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

type ImportStatement struct {
	// AbsToResource is the original ImportConfig ToResource+ContainingModule
	AbsToResource    addrs.ConfigResource
	ContainingModule addrs.Module
	Import           *configs.Import
}

// FindImportStatements recurses through the modules of the given configuration
// and returns a set of all "import" blocks defined within after deduplication
// on the To address.
//
// An "import" block in a parent module overrides an import block in a child
// module when both target the same configuration object.
func FindImportStatements(rootCfg *configs.Config) addrs.Map[addrs.ConfigResource, ImportStatement] {
	imports := findImportStatements(rootCfg, addrs.MakeMap[addrs.ConfigResource, ImportStatement]())
	return imports
}

func findImportStatements(cfg *configs.Config, into addrs.Map[addrs.ConfigResource, ImportStatement]) addrs.Map[addrs.ConfigResource, ImportStatement] {
	for _, mi := range cfg.Module.Import {
		// First, stitch together the module path and the RelSubject to form
		// the absolute address of the config resource being removed.
		res := mi.ToResource
		toAddr := addrs.ConfigResource{
			Module:   append(cfg.Path, res.Module...),
			Resource: res.Resource,
		}

		// If we already have an import statement for this ConfigResource, it
		// must have come from a parent module, because duplicate import
		// blocks in the same module result in an error.
		// The import block in the parent module overrides the block in the
		// child module.
		existingResource, ok := into.GetOk(toAddr)
		if ok {
			if existingResource.AbsToResource.Equal(toAddr) {
				continue
			}
		}

		into.Put(toAddr, ImportStatement{
			AbsToResource:    toAddr,
			ContainingModule: cfg.Path,
			Import:           mi,
		})
	}

	for _, childCfg := range cfg.Children {
		into = findImportStatements(childCfg, into)
	}

	return into
}
