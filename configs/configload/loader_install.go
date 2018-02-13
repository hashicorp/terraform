package configload

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/registry/regsrc"
)

// InstallModules analyses the root module in the given directory and installs
// all of its direct and transitive dependencies into the loader's modules
// directory, which must already exist.
//
// Since InstallModules makes possibly-time-consuming calls to remote services,
// a hook interface is supported to allow the caller to be notified when
// each module is installed and, for remote modules, when downloading begins.
// LoadConfig guarantees that two hook calls will not happen concurrently but
// it does not guarantee any particular ordering of hook calls. This mechanism
// is for UI feedback only and does not give the caller any control over the
// process.
//
// If modules are already installed in the target directory, they will be
// skipped unless their source address or version have changed or unless
// the upgrade flag is set.
//
// InstallModules never deletes any directory, except in the case where it
// needs to replace a directory that is already present with a newly-extracted
// package.
//
// If the returned diagnostics contains errors then the module installation
// may have wholly or partially completed. Modules must be loaded in order
// to find their dependencies, so this function does many of the same checks
// as LoadConfig as a side-effect.
func (l *Loader) InstallModules(rootDir string, upgrade bool, hooks InstallHooks) hcl.Diagnostics {
	rootMod, diags := l.parser.LoadConfigDir(rootDir)
	if rootMod == nil {
		return diags
	}

	if hooks == nil {
		// Use our no-op implementation as a placeholder
		hooks = InstallHooksImpl{}
	}

	// Create a manifest record for the root module. This will be used if
	// there are any relative-pathed modules in the root.
	l.modules.manifest[""] = moduleRecord{
		Key: "",
		Dir: rootDir,
	}

	_, cDiags := configs.BuildConfig(rootMod, configs.ModuleWalkerFunc(
		func(req *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {

			key := manifestKey(req.Path)
			instPath := l.packageInstallPath(req.Path)

			log.Printf("[DEBUG] Module installer: begin %s", key)

			// First we'll check if we need to upgrade/replace an existing
			// installed module, and delete it out of the way if so.
			replace := upgrade
			if !replace {
				record, recorded := l.modules.manifest[key]
				switch {
				case !recorded:
					log.Printf("[TRACE] %s is not yet installed", key)
					replace = true
				case record.SourceAddr != req.SourceAddr:
					log.Printf("[TRACE] %s source address has changed from %q to %q", key, record.SourceAddr, req.SourceAddr)
					replace = true
				case record.Version != nil && !req.VersionConstraint.Required.Check(record.Version):
					log.Printf("[TRACE] %s version %s no longer compatible with constraints %s", key, record.Version, req.VersionConstraint.Required)
					replace = true
				}
			}

			// If we _are_ planning to replace this module, then we'll remove
			// it now so our installation code below won't conflict with any
			// existing remnants.
			if replace {
				if _, recorded := l.modules.manifest[key]; recorded {
					log.Printf("[TRACE] discarding previous record of %s prior to reinstall", key)
				}
				delete(l.modules.manifest, key)
				// Deleting a module invalidates all of its descendent modules too.
				keyPrefix := key + "."
				for subKey := range l.modules.manifest {
					if strings.HasPrefix(subKey, keyPrefix) {
						if _, recorded := l.modules.manifest[subKey]; recorded {
							log.Printf("[TRACE] also discarding downstream %s", subKey)
						}
						delete(l.modules.manifest, subKey)
					}
				}
			}

			record, recorded := l.modules.manifest[key]
			if !recorded {
				// Clean up any stale cache directory that might be present.
				// If this is a local (relative) source then the dir will
				// not exist, but we'll ignore that.
				log.Printf("[TRACE] cleaning directory %s prior to install of %s", instPath, key)
				err := l.modules.FS.RemoveAll(instPath)
				if err != nil && !os.IsNotExist(err) {
					log.Printf("[TRACE] failed to remove %s: %s", key, err)
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Failed to remove local module cache",
						Detail: fmt.Sprintf(
							"Terraform tried to remove %s in order to reinstall this module, but encountered an error: %s",
							instPath, err,
						),
						Subject: &req.CallRange,
					})
					return nil, nil, diags
				}
			} else {
				// If this module is already recorded and its root directory
				// exists then we will just load what's already there and
				// keep our existing record.
				info, err := l.modules.FS.Stat(record.Dir)
				if err == nil && info.IsDir() {
					mod, mDiags := l.parser.LoadConfigDir(record.Dir)
					diags = append(diags, mDiags...)

					log.Printf("[TRACE] Module installer: %s %s already installed in %s", key, record.Version, record.Dir)
					return mod, record.Version, diags
				}
			}

			// If we get down here then it's finally time to actually install
			// the module. There are some variants to this process depending
			// on what type of module source address we have.
			switch {

			case isLocalSourceAddr(req.SourceAddr):
				log.Printf("[TRACE] %s has local path %q", key, req.SourceAddr)
				mod, mDiags := l.installLocalModule(req, key, hooks)
				diags = append(diags, mDiags...)
				return mod, nil, diags

			case isRegistrySourceAddr(req.SourceAddr):
				addr, err := regsrc.ParseModuleSource(req.SourceAddr)
				if err != nil {
					// Should never happen because isRegistrySourceAddr already validated
					panic(err)
				}
				log.Printf("[TRACE] %s is a registry module at %s", key, addr)

				// TODO: Implement
				panic("registry source installation not yet implemented")

			default:
				log.Printf("[TRACE] %s address %q will be interpreted with go-getter", key, req.SourceAddr)

				// TODO: Implement
				panic("fallback source installation not yet implemented")

			}

		},
	))
	diags = append(diags, cDiags...)

	err := l.modules.writeModuleManifestSnapshot()
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to update module manifest",
			Detail:   fmt.Sprintf("Unable to write the module manifest file: %s", err),
		})
	}

	return diags
}

func (l *Loader) installLocalModule(req *configs.ModuleRequest, key string, hooks InstallHooks) (*configs.Module, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	parentKey := manifestKey(req.Parent.Path)
	parentRecord, recorded := l.modules.manifest[parentKey]
	if !recorded {
		// This is indicative of a bug rather than a user-actionable error
		panic(fmt.Errorf("missing manifest record for parent module %s", parentKey))
	}

	if len(req.VersionConstraint.Required) != 0 {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid version constraint",
			Detail:   "A version constraint cannot be applied to a module at a relative local path.",
			Subject:  &req.VersionConstraint.DeclRange,
		})
	}

	// For local sources we don't actually need to modify the
	// filesystem at all because the parent already wrote
	// the files we need, and so we just load up what's already here.
	newDir := filepath.Join(parentRecord.Dir, req.SourceAddr)
	log.Printf("[TRACE] %s uses directory from parent: %s", key, newDir)
	mod, mDiags := l.parser.LoadConfigDir(newDir)
	if mod == nil {
		// nil indicates missing or unreadable directory, so we'll
		// discard the returned diags and return a more specific
		// error message here.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unreadable module directory",
			Detail:   fmt.Sprintf("The directory %s could not be read.", newDir),
			Subject:  &req.SourceAddrRange,
		})
	} else {
		diags = append(diags, mDiags...)
	}

	// Note the local location in our manifest.
	l.modules.manifest[key] = moduleRecord{
		Key:        key,
		Dir:        newDir,
		SourceAddr: req.SourceAddr,
	}
	log.Printf("[TRACE] Module installer: %s installed at %s", key, newDir)
	hooks.Install(key, nil, newDir)

	return mod, diags
}

func (l *Loader) packageInstallPath(modulePath []string) string {
	return filepath.Join(l.modules.Dir, strings.Join(modulePath, "."))
}
