package moduleref

import (
	"strings"

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
	// prevent introducing sideffects to the original
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
			Records: []Record{},
		},
	}
}

// Resolve will attempt to find all module references for the passed configuration
// and return a new manifest encapsulating this information.
func (r *Resolver) Resolve(cfg *configs.Config) *Manifest {
	// First find all the referenced modules.
	r.findAndTrimReferencedEntries(cfg)

	// Lastly append any entry with no reference found
	for _, entry := range r.internalManifest {
		r.manifest.AddModuleEntry(entry, false)
	}

	return r.manifest
}

// findAndTrimReferencedEntries will traverse a given Terraform configuration
// and attempt find a caller for every entry in the internal module manifest.
// If an entry is found, it will be removed from the internal manifest and
// appended to the manifest that records this new information.
func (r *Resolver) findAndTrimReferencedEntries(cfg *configs.Config) {
	for entryKey, entry := range r.internalManifest {
		for callerKey := range cfg.Module.ModuleCalls {
			normalizedPath := r.normalizeModulePath(cfg.Path.String())
			if normalizedPath != "" {
				callerKey = normalizedPath + "." + callerKey
			}

			// This is a sufficient check as caller keys are unique per module
			// entry.
			if callerKey == entryKey {
				// We found a reference!
				r.manifest.AddModuleEntry(entry, true)
				// "Trim" the entry from the internal manifest, saving us cycles
				// as we descend into the module tree.
				delete(r.internalManifest, entryKey)
				break
			}
		}
	}

	// Traverse the child configurations
	for _, childCfg := range cfg.Children {
		r.findAndTrimReferencedEntries(childCfg)
	}
}

// normalizeModulePath will attempt to strip path prefixes from a given module
// path. The intent is so that the path is converted to a internal manifest entry
// key.
// Example: module.foo_module.module.bar becomes foo_module.bar
func (r *Resolver) normalizeModulePath(path string) string {
	path = strings.ReplaceAll(path, ".module.", ".")
	path = strings.TrimPrefix(path, "module.")
	return path
}
