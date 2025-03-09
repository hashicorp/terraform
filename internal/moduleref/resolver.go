// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleref

import (
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/modsdir"
)

// Resolver is the struct responsible for finding all modules references in
// Terraform configuration for a given internal module manifest.
type Resolver struct {
	manifest         *Manifest
	internalManifest modsdir.Manifest
}

// NewResolver creates a new Resolver, storing a copy of the internal manifest
// that is passed.
func NewResolver(internalManifest modsdir.Manifest) *Resolver {
	// Since maps are pointers, create a copy of the internal manifest to
	// prevent introducing side effects to the original
	internalManifestCopy := make(modsdir.Manifest, len(internalManifest))
	for k, v := range internalManifest {
		internalManifestCopy[k] = v
	}

	// Remove the root module entry from the internal manifest as it is
	// never directly referenced.
	delete(internalManifestCopy, "")

	return &Resolver{
		internalManifest: internalManifestCopy,
		manifest: &Manifest{
			FormatVersion: FormatVersion,
			Records:       Records{},
		},
	}
}

// Resolve will attempt to find all module references for the passed configuration
// and return a new manifest encapsulating this information.
func (r *Resolver) Resolve(cfg *configs.Config) *Manifest {
	// First find all the referenced modules.
	r.findAndTrimReferencedEntries(cfg, nil, nil)

	return r.manifest
}

// findAndTrimReferencedEntries will traverse a given Terraform configuration
// and attempt find a caller for every entry in the internal module manifest.
// If an entry is found, it will be removed from the internal manifest and
// appended to the manifest that records this new information in a nested heirarchy.
func (r *Resolver) findAndTrimReferencedEntries(cfg *configs.Config, parentRecord *Record, parentKey *string) {
	var name string
	var versionConstraints version.Constraints
	if parentKey != nil {
		for key := range cfg.Parent.Children {
			if key == *parentKey {
				name = key
				if cfg.Parent.Module.ModuleCalls[key] != nil {
					versionConstraints = cfg.Parent.Module.ModuleCalls[key].Version.Required
				}
				break
			}
		}
	}

	childRecord := &Record{
		Key:                name,
		Source:             cfg.SourceAddr,
		VersionConstraints: versionConstraints,
	}
	key := strings.Join(cfg.Path, ".")

	for entryKey, entry := range r.internalManifest {
		if entryKey == key {
			// Use resolved version from manifest
			childRecord.Version = entry.Version
			if parentRecord.Source != nil {
				parentRecord.addChild(childRecord)
			} else {
				r.manifest.addModuleEntry(childRecord)
			}
			// "Trim" the entry from the internal manifest, saving us cycles
			// as we descend into the module tree.
			delete(r.internalManifest, entryKey)
			break
		}
	}

	// Traverse the child configurations
	for childKey, childCfg := range cfg.Children {
		r.findAndTrimReferencedEntries(childCfg, childRecord, &childKey)
	}
}
