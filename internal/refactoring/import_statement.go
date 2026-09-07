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
// and returns a set of all "import" blocks. These import targets are not deduplicated
// as we must wait until expansion occurs since we don't know the exact resource instance yet.
func FindImportStatements(rootCfg *configs.Config) []ImportStatement {
	imports := findImportStatements(rootCfg, make([]ImportStatement, 0))
	return imports
}

func findImportStatements(cfg *configs.Config, into []ImportStatement) []ImportStatement {
	for _, mi := range cfg.Module.Import {
		// First, stitch together the module path and the RelSubject to form
		// the absolute address of the config resource being removed.
		res := mi.ToResource
		toAddr := addrs.ConfigResource{
			Module:   append(cfg.Path, res.Module...),
			Resource: res.Resource,
		}

		into = append(into, ImportStatement{
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
