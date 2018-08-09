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
	"github.com/hashicorp/terraform/registry"
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
//
// This function will panic if called on a loader that cannot install modules.
// Use CanInstallModules to determine if a loader can install modules, or
// refer to the documentation for that method for situations where module
// installation capability is guaranteed.
func (l *Loader) InstallModules(rootDir string, upgrade bool, hooks InstallHooks) hcl.Diagnostics {
	if !l.CanInstallModules() {
		panic(fmt.Errorf("InstallModules called on loader that cannot install modules"))
	}

	rootMod, diags := l.parser.LoadConfigDir(rootDir)
	if rootMod == nil {
		return diags
	}

	getter := reusingGetter{}
	instDiags := l.installDescendentModules(rootMod, rootDir, upgrade, hooks, getter)
	diags = append(diags, instDiags...)

	return diags
}

func (l *Loader) installDescendentModules(rootMod *configs.Module, rootDir string, upgrade bool, hooks InstallHooks, getter reusingGetter) hcl.Diagnostics {
	var diags hcl.Diagnostics

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

				mod, v, mDiags := l.installRegistryModule(req, key, instPath, addr, hooks, getter)
				diags = append(diags, mDiags...)
				return mod, v, diags

			default:
				log.Printf("[TRACE] %s address %q will be handled by go-getter", key, req.SourceAddr)

				mod, mDiags := l.installGoGetterModule(req, key, instPath, hooks, getter)
				diags = append(diags, mDiags...)
				return mod, nil, diags
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

// CanInstallModules returns true if InstallModules can be used with this
// loader.
//
// Loaders created with NewLoader can always install modules. Loaders created
// from plan files (where the configuration is embedded in the plan file itself)
// cannot install modules, because the plan file is read-only.
func (l *Loader) CanInstallModules() bool {
	return l.modules.CanInstall
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
	log.Printf("[DEBUG] Module installer: %s installed at %s", key, newDir)
	hooks.Install(key, nil, newDir)

	return mod, diags
}

func (l *Loader) installRegistryModule(req *configs.ModuleRequest, key string, instPath string, addr *regsrc.Module, hooks InstallHooks, getter reusingGetter) (*configs.Module, *version.Version, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	hostname, err := addr.SvcHost()
	if err != nil {
		// If it looks like the user was trying to use punycode then we'll generate
		// a specialized error for that case. We require the unicode form of
		// hostname so that hostnames are always human-readable in configuration
		// and punycode can't be used to hide a malicious module hostname.
		if strings.HasPrefix(addr.RawHost.Raw, "xn--") {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid module registry hostname",
				Detail:   "The hostname portion of this source address is not an acceptable hostname. Internationalized domain names must be given in unicode form rather than ASCII (\"punycode\") form.",
				Subject:  &req.SourceAddrRange,
			})
		} else {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid module registry hostname",
				Detail:   "The hostname portion of this source address is not a valid hostname.",
				Subject:  &req.SourceAddrRange,
			})
		}
		return nil, nil, diags
	}

	reg := l.modules.Registry

	log.Printf("[DEBUG] %s listing available versions of %s at %s", key, addr, hostname)
	resp, err := reg.ModuleVersions(addr)
	if err != nil {
		if registry.IsModuleNotFound(err) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Module not found",
				Detail:   fmt.Sprintf("The specified module could not be found in the module registry at %s.", hostname),
				Subject:  &req.SourceAddrRange,
			})
		} else {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Error accessing remote module registry",
				Detail:   fmt.Sprintf("Failed to retrieve available versions for this module from %s: %s.", hostname, err),
				Subject:  &req.SourceAddrRange,
			})
		}
		return nil, nil, diags
	}

	// The response might contain information about dependencies to allow us
	// to potentially optimize future requests, but we don't currently do that
	// and so for now we'll just take the first item which is guaranteed to
	// be the address we requested.
	if len(resp.Modules) < 1 {
		// Should never happen, but since this is a remote service that may
		// be implemented by third-parties we will handle it gracefully.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid response from remote module registry",
			Detail:   fmt.Sprintf("The registry at %s returned an invalid response when Terraform requested available versions for this module.", hostname),
			Subject:  &req.SourceAddrRange,
		})
		return nil, nil, diags
	}

	modMeta := resp.Modules[0]

	var latestMatch *version.Version
	var latestVersion *version.Version
	for _, mv := range modMeta.Versions {
		v, err := version.NewVersion(mv.Version)
		if err != nil {
			// Should never happen if the registry server is compliant with
			// the protocol, but we'll warn if not to assist someone who
			// might be developing a module registry server.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Invalid response from remote module registry",
				Detail:   fmt.Sprintf("The registry at %s returned an invalid version string %q for this module, which Terraform ignored.", hostname, mv.Version),
				Subject:  &req.SourceAddrRange,
			})
			continue
		}

		// If we've found a pre-release version then we'll ignore it unless
		// it was exactly requested.
		if v.Prerelease() != "" && req.VersionConstraint.Required.String() != v.String() {
			log.Printf("[TRACE] %s ignoring %s because it is a pre-release and was not requested exactly", key, v)
			continue
		}

		if latestVersion == nil || v.GreaterThan(latestVersion) {
			latestVersion = v
		}

		if req.VersionConstraint.Required.Check(v) {
			if latestMatch == nil || v.GreaterThan(latestMatch) {
				latestMatch = v
			}
		}
	}

	if latestVersion == nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Module has no versions",
			Detail:   fmt.Sprintf("The specified module does not have any available versions."),
			Subject:  &req.SourceAddrRange,
		})
		return nil, nil, diags
	}

	if latestMatch == nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unresolvable module version constraint",
			Detail:   fmt.Sprintf("There is no available version of %q that matches the given version constraint. The newest available version is %s.", addr, latestVersion),
			Subject:  &req.VersionConstraint.DeclRange,
		})
		return nil, nil, diags
	}

	// Report up to the caller that we're about to start downloading.
	packageAddr, _ := splitAddrSubdir(req.SourceAddr)
	hooks.Download(key, packageAddr, latestMatch)

	// If we manage to get down here then we've found a suitable version to
	// install, so we need to ask the registry where we should download it from.
	// The response to this is a go-getter-style address string.
	dlAddr, err := reg.ModuleLocation(addr, latestMatch.String())
	if err != nil {
		log.Printf("[ERROR] %s from %s %s: %s", key, addr, latestMatch, err)
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid response from remote module registry",
			Detail:   fmt.Sprintf("The remote registry at %s failed to return a download URL for %s %s.", hostname, addr, latestMatch),
			Subject:  &req.VersionConstraint.DeclRange,
		})
		return nil, nil, diags
	}

	log.Printf("[TRACE] %s %s %s is available at %q", key, addr, latestMatch, dlAddr)

	modDir, err := getter.getWithGoGetter(instPath, dlAddr)
	if err != nil {
		// Errors returned by go-getter have very inconsistent quality as
		// end-user error messages, but for now we're accepting that because
		// we have no way to recognize any specific errors to improve them
		// and masking the error entirely would hide valuable diagnostic
		// information from the user.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to download module",
			Detail:   fmt.Sprintf("Error attempting to download module source code from %q: %s", dlAddr, err),
			Subject:  &req.CallRange,
		})
		return nil, nil, diags
	}

	log.Printf("[TRACE] %s %q was downloaded to %s", key, dlAddr, modDir)

	if addr.RawSubmodule != "" {
		// Append the user's requested subdirectory to any subdirectory that
		// was implied by any of the nested layers we expanded within go-getter.
		modDir = filepath.Join(modDir, addr.RawSubmodule)
	}

	log.Printf("[TRACE] %s should now be at %s", key, modDir)

	// Finally we are ready to try actually loading the module.
	mod, mDiags := l.parser.LoadConfigDir(modDir)
	if mod == nil {
		// nil indicates missing or unreadable directory, so we'll
		// discard the returned diags and return a more specific
		// error message here. For registry modules this actually
		// indicates a bug in the code above, since it's not the
		// user's responsibility to create the directory in this case.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unreadable module directory",
			Detail:   fmt.Sprintf("The directory %s could not be read. This is a bug in Terraform and should be reported.", modDir),
			Subject:  &req.CallRange,
		})
	} else {
		diags = append(diags, mDiags...)
	}

	// Note the local location in our manifest.
	l.modules.manifest[key] = moduleRecord{
		Key:        key,
		Version:    latestMatch,
		Dir:        modDir,
		SourceAddr: req.SourceAddr,
	}
	log.Printf("[DEBUG] Module installer: %s installed at %s", key, modDir)
	hooks.Install(key, latestMatch, modDir)

	return mod, latestMatch, diags
}

func (l *Loader) installGoGetterModule(req *configs.ModuleRequest, key string, instPath string, hooks InstallHooks, getter reusingGetter) (*configs.Module, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	// Report up to the caller that we're about to start downloading.
	packageAddr, _ := splitAddrSubdir(req.SourceAddr)
	hooks.Download(key, packageAddr, nil)

	modDir, err := getter.getWithGoGetter(instPath, req.SourceAddr)
	if err != nil {
		// Errors returned by go-getter have very inconsistent quality as
		// end-user error messages, but for now we're accepting that because
		// we have no way to recognize any specific errors to improve them
		// and masking the error entirely would hide valuable diagnostic
		// information from the user.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to download module",
			Detail:   fmt.Sprintf("Error attempting to download module source code from %q: %s", packageAddr, err),
			Subject:  &req.SourceAddrRange,
		})
		return nil, diags
	}

	log.Printf("[TRACE] %s %q was downloaded to %s", key, req.SourceAddr, modDir)

	mod, mDiags := l.parser.LoadConfigDir(modDir)
	if mod == nil {
		// nil indicates missing or unreadable directory, so we'll
		// discard the returned diags and return a more specific
		// error message here. For registry modules this actually
		// indicates a bug in the code above, since it's not the
		// user's responsibility to create the directory in this case.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unreadable module directory",
			Detail:   fmt.Sprintf("The directory %s could not be read. This is a bug in Terraform and should be reported.", modDir),
			Subject:  &req.CallRange,
		})
	} else {
		diags = append(diags, mDiags...)
	}

	// Note the local location in our manifest.
	l.modules.manifest[key] = moduleRecord{
		Key:        key,
		Dir:        modDir,
		SourceAddr: req.SourceAddr,
	}
	log.Printf("[DEBUG] Module installer: %s installed at %s", key, modDir)
	hooks.Install(key, nil, modDir)

	return mod, diags
}

func (l *Loader) packageInstallPath(modulePath []string) string {
	return filepath.Join(l.modules.Dir, strings.Join(modulePath, "."))
}
