package initwd

import (
	"fmt"
	"github.com/hashicorp/terraform/registry"
	"log"
	"os"
	"path/filepath"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/earlyconfig"
	"github.com/hashicorp/terraform/internal/modsdir"
	"github.com/hashicorp/terraform/registry/regsrc"
	"github.com/hashicorp/terraform/tfdiags"
)

type ModuleInstaller struct {
	modsDir string
	reg     *registry.Client
}

func NewModuleInstaller(modsDir string, reg *registry.Client) *ModuleInstaller {
	return &ModuleInstaller{
		modsDir: modsDir,
		reg:     reg,
	}
}

// InstallModules analyses the root module in the given directory and installs
// all of its direct and transitive dependencies into the given modules
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
// If successful (the returned diagnostics contains no errors) then the
// first return value is the early configuration tree that was constructed by
// the installation process.
func (i *ModuleInstaller) InstallModules(rootDir string, upgrade bool, hooks ModuleInstallHooks) (*earlyconfig.Config, tfdiags.Diagnostics) {
	log.Printf("[TRACE] ModuleInstaller: installing child modules for %s into %s", rootDir, i.modsDir)

	rootMod, diags := earlyconfig.LoadModule(rootDir)
	if rootMod == nil {
		return nil, diags
	}

	manifest, err := modsdir.ReadManifestSnapshotForDir(i.modsDir)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to read modules manifest file",
			fmt.Sprintf("Error reading manifest for %s: %s.", i.modsDir, err),
		))
		return nil, diags
	}

	getter := reusingGetter{}
	cfg, instDiags := i.installDescendentModules(rootMod, rootDir, manifest, upgrade, hooks, getter)
	diags = append(diags, instDiags...)

	return cfg, diags
}

func (i *ModuleInstaller) installDescendentModules(rootMod *tfconfig.Module, rootDir string, manifest modsdir.Manifest, upgrade bool, hooks ModuleInstallHooks, getter reusingGetter) (*earlyconfig.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if hooks == nil {
		// Use our no-op implementation as a placeholder
		hooks = ModuleInstallHooksImpl{}
	}

	// Create a manifest record for the root module. This will be used if
	// there are any relative-pathed modules in the root.
	manifest[""] = modsdir.Record{
		Key: "",
		Dir: rootDir,
	}

	cfg, cDiags := earlyconfig.BuildConfig(rootMod, earlyconfig.ModuleWalkerFunc(
		func(req *earlyconfig.ModuleRequest) (*tfconfig.Module, *version.Version, tfdiags.Diagnostics) {

			key := manifest.ModuleKey(req.Path)
			instPath := i.packageInstallPath(req.Path)

			log.Printf("[DEBUG] Module installer: begin %s", key)

			// First we'll check if we need to upgrade/replace an existing
			// installed module, and delete it out of the way if so.
			replace := upgrade
			if !replace {
				record, recorded := manifest[key]
				switch {
				case !recorded:
					log.Printf("[TRACE] ModuleInstaller: %s is not yet installed", key)
					replace = true
				case record.SourceAddr != req.SourceAddr:
					log.Printf("[TRACE] ModuleInstaller: %s source address has changed from %q to %q", key, record.SourceAddr, req.SourceAddr)
					replace = true
				case record.Version != nil && !req.VersionConstraints.Check(record.Version):
					log.Printf("[TRACE] ModuleInstaller: %s version %s no longer compatible with constraints %s", key, record.Version, req.VersionConstraints)
					replace = true
				}
			}

			// If we _are_ planning to replace this module, then we'll remove
			// it now so our installation code below won't conflict with any
			// existing remnants.
			if replace {
				if _, recorded := manifest[key]; recorded {
					log.Printf("[TRACE] ModuleInstaller: discarding previous record of %s prior to reinstall", key)
				}
				delete(manifest, key)
				// Deleting a module invalidates all of its descendent modules too.
				keyPrefix := key + "."
				for subKey := range manifest {
					if strings.HasPrefix(subKey, keyPrefix) {
						if _, recorded := manifest[subKey]; recorded {
							log.Printf("[TRACE] ModuleInstaller: also discarding downstream %s", subKey)
						}
						delete(manifest, subKey)
					}
				}
			}

			record, recorded := manifest[key]
			if !recorded {
				// Clean up any stale cache directory that might be present.
				// If this is a local (relative) source then the dir will
				// not exist, but we'll ignore that.
				log.Printf("[TRACE] ModuleInstaller: cleaning directory %s prior to install of %s", instPath, key)
				err := os.RemoveAll(instPath)
				if err != nil && !os.IsNotExist(err) {
					log.Printf("[TRACE] ModuleInstaller: failed to remove %s: %s", key, err)
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Failed to remove local module cache",
						fmt.Sprintf(
							"Terraform tried to remove %s in order to reinstall this module, but encountered an error: %s",
							instPath, err,
						),
					))
					return nil, nil, diags
				}
			} else {
				// If this module is already recorded and its root directory
				// exists then we will just load what's already there and
				// keep our existing record.
				info, err := os.Stat(record.Dir)
				if err == nil && info.IsDir() {
					mod, mDiags := earlyconfig.LoadModule(record.Dir)
					diags = diags.Append(mDiags)

					log.Printf("[TRACE] ModuleInstaller: Module installer: %s %s already installed in %s", key, record.Version, record.Dir)
					return mod, record.Version, diags
				}
			}

			// If we get down here then it's finally time to actually install
			// the module. There are some variants to this process depending
			// on what type of module source address we have.
			switch {

			case isLocalSourceAddr(req.SourceAddr):
				log.Printf("[TRACE] ModuleInstaller: %s has local path %q", key, req.SourceAddr)
				mod, mDiags := i.installLocalModule(req, key, manifest, hooks)
				diags = append(diags, mDiags...)
				return mod, nil, diags

			case isRegistrySourceAddr(req.SourceAddr):
				addr, err := regsrc.ParseModuleSource(req.SourceAddr)
				if err != nil {
					// Should never happen because isRegistrySourceAddr already validated
					panic(err)
				}
				log.Printf("[TRACE] ModuleInstaller: %s is a registry module at %s", key, addr)

				mod, v, mDiags := i.installRegistryModule(req, key, instPath, addr, manifest, hooks, getter)
				diags = append(diags, mDiags...)
				return mod, v, diags

			default:
				log.Printf("[TRACE] ModuleInstaller: %s address %q will be handled by go-getter", key, req.SourceAddr)

				mod, mDiags := i.installGoGetterModule(req, key, instPath, manifest, hooks, getter)
				diags = append(diags, mDiags...)
				return mod, nil, diags
			}

		},
	))
	diags = append(diags, cDiags...)

	err := manifest.WriteSnapshotToDir(i.modsDir)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to update module manifest",
			fmt.Sprintf("Unable to write the module manifest file: %s", err),
		))
	}

	return cfg, diags
}

func (i *ModuleInstaller) installLocalModule(req *earlyconfig.ModuleRequest, key string, manifest modsdir.Manifest, hooks ModuleInstallHooks) (*tfconfig.Module, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	parentKey := manifest.ModuleKey(req.Parent.Path)
	parentRecord, recorded := manifest[parentKey]
	if !recorded {
		// This is indicative of a bug rather than a user-actionable error
		panic(fmt.Errorf("missing manifest record for parent module %s", parentKey))
	}

	if len(req.VersionConstraints) != 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid version constraint",
			fmt.Sprintf("Cannot apply a version constraint to module %q (at %s:%d) because it has a relative local path.", req.Name, req.CallPos.Filename, req.CallPos.Line),
		))
	}

	// For local sources we don't actually need to modify the
	// filesystem at all because the parent already wrote
	// the files we need, and so we just load up what's already here.
	newDir := filepath.Join(parentRecord.Dir, req.SourceAddr)
	log.Printf("[TRACE] ModuleInstaller: %s uses directory from parent: %s", key, newDir)
	mod, mDiags := earlyconfig.LoadModule(newDir)
	if mod == nil {
		// nil indicates missing or unreadable directory, so we'll
		// discard the returned diags and return a more specific
		// error message here.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unreadable module directory",
			fmt.Sprintf("The directory %s could not be read for module %q at %s:%d.", newDir, req.Name, req.CallPos.Filename, req.CallPos.Line),
		))
	} else {
		diags = diags.Append(mDiags)
	}

	// Note the local location in our manifest.
	manifest[key] = modsdir.Record{
		Key:        key,
		Dir:        newDir,
		SourceAddr: req.SourceAddr,
	}
	log.Printf("[DEBUG] Module installer: %s installed at %s", key, newDir)
	hooks.Install(key, nil, newDir)

	return mod, diags
}

func (i *ModuleInstaller) installRegistryModule(req *earlyconfig.ModuleRequest, key string, instPath string, addr *regsrc.Module, manifest modsdir.Manifest, hooks ModuleInstallHooks, getter reusingGetter) (*tfconfig.Module, *version.Version, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	hostname, err := addr.SvcHost()
	if err != nil {
		// If it looks like the user was trying to use punycode then we'll generate
		// a specialized error for that case. We require the unicode form of
		// hostname so that hostnames are always human-readable in configuration
		// and punycode can't be used to hide a malicious module hostname.
		if strings.HasPrefix(addr.RawHost.Raw, "xn--") {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid module registry hostname",
				fmt.Sprintf("The hostname portion of the module %q source address (at %s:%d) is not an acceptable hostname. Internationalized domain names must be given in unicode form rather than ASCII (\"punycode\") form.", req.Name, req.CallPos.Filename, req.CallPos.Line),
			))
		} else {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid module registry hostname",
				fmt.Sprintf("The hostname portion of the module %q source address (at %s:%d) is not a valid hostname.", req.Name, req.CallPos.Filename, req.CallPos.Line),
			))
		}
		return nil, nil, diags
	}

	reg := i.reg

	log.Printf("[DEBUG] %s listing available versions of %s at %s", key, addr, hostname)
	resp, err := reg.ModuleVersions(addr)
	if err != nil {
		if registry.IsModuleNotFound(err) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Module not found",
				fmt.Sprintf("Module %q (from %s:%d) cannot be found in the module registry at %s.", req.Name, req.CallPos.Filename, req.CallPos.Line, hostname),
			))
		} else {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Error accessing remote module registry",
				fmt.Sprintf("Failed to retrieve available versions for module %q (%s:%d) from %s: %s.", req.Name, req.CallPos.Filename, req.CallPos.Line, hostname, err),
			))
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
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid response from remote module registry",
			fmt.Sprintf("The registry at %s returned an invalid response when Terraform requested available versions for module %q (%s:%d).", hostname, req.Name, req.CallPos.Filename, req.CallPos.Line),
		))
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
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"Invalid response from remote module registry",
				fmt.Sprintf("The registry at %s returned an invalid version string %q for module %q (%s:%d), which Terraform ignored.", hostname, mv.Version, req.Name, req.CallPos.Filename, req.CallPos.Line),
			))
			continue
		}

		// If we've found a pre-release version then we'll ignore it unless
		// it was exactly requested.
		if v.Prerelease() != "" && req.VersionConstraints.String() != v.String() {
			log.Printf("[TRACE] ModuleInstaller: %s ignoring %s because it is a pre-release and was not requested exactly", key, v)
			continue
		}

		if latestVersion == nil || v.GreaterThan(latestVersion) {
			latestVersion = v
		}

		if req.VersionConstraints.Check(v) {
			if latestMatch == nil || v.GreaterThan(latestMatch) {
				latestMatch = v
			}
		}
	}

	if latestVersion == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Module has no versions",
			fmt.Sprintf("Module %q (%s:%d) has no versions available on %s.", addr, req.CallPos.Filename, req.CallPos.Line, hostname),
		))
		return nil, nil, diags
	}

	if latestMatch == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unresolvable module version constraint",
			fmt.Sprintf("There is no available version of module %q (%s:%d) which matches the given version constraint. The newest available version is %s.", addr, req.CallPos.Filename, req.CallPos.Line, latestVersion),
		))
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
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid response from remote module registry",
			fmt.Sprintf("The remote registry at %s failed to return a download URL for %s %s.", hostname, addr, latestMatch),
		))
		return nil, nil, diags
	}

	log.Printf("[TRACE] ModuleInstaller: %s %s %s is available at %q", key, addr, latestMatch, dlAddr)

	modDir, err := getter.getWithGoGetter(instPath, dlAddr)
	if err != nil {
		// Errors returned by go-getter have very inconsistent quality as
		// end-user error messages, but for now we're accepting that because
		// we have no way to recognize any specific errors to improve them
		// and masking the error entirely would hide valuable diagnostic
		// information from the user.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to download module",
			fmt.Sprintf("Error attempting to download module %q (%s:%d) source code from %q: %s.", req.Name, req.CallPos.Filename, req.CallPos.Line, dlAddr, err),
		))
		return nil, nil, diags
	}

	log.Printf("[TRACE] ModuleInstaller: %s %q was downloaded to %s", key, dlAddr, modDir)

	if addr.RawSubmodule != "" {
		// Append the user's requested subdirectory to any subdirectory that
		// was implied by any of the nested layers we expanded within go-getter.
		modDir = filepath.Join(modDir, addr.RawSubmodule)
	}

	log.Printf("[TRACE] ModuleInstaller: %s should now be at %s", key, modDir)

	// Finally we are ready to try actually loading the module.
	mod, mDiags := earlyconfig.LoadModule(modDir)
	if mod == nil {
		// nil indicates missing or unreadable directory, so we'll
		// discard the returned diags and return a more specific
		// error message here. For registry modules this actually
		// indicates a bug in the code above, since it's not the
		// user's responsibility to create the directory in this case.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unreadable module directory",
			fmt.Sprintf("The directory %s could not be read. This is a bug in Terraform and should be reported.", modDir),
		))
	} else {
		diags = append(diags, mDiags...)
	}

	// Note the local location in our manifest.
	manifest[key] = modsdir.Record{
		Key:        key,
		Version:    latestMatch,
		Dir:        modDir,
		SourceAddr: req.SourceAddr,
	}
	log.Printf("[DEBUG] Module installer: %s installed at %s", key, modDir)
	hooks.Install(key, latestMatch, modDir)

	return mod, latestMatch, diags
}

func (i *ModuleInstaller) installGoGetterModule(req *earlyconfig.ModuleRequest, key string, instPath string, manifest modsdir.Manifest, hooks ModuleInstallHooks, getter reusingGetter) (*tfconfig.Module, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Report up to the caller that we're about to start downloading.
	packageAddr, _ := splitAddrSubdir(req.SourceAddr)
	hooks.Download(key, packageAddr, nil)

	modDir, err := getter.getWithGoGetter(instPath, req.SourceAddr)
	if err != nil {
		if err, ok := err.(*MaybeRelativePathErr); ok {
			log.Printf(
				"[TRACE] ModuleInstaller: %s looks like a local path but is missing ./ or ../",
				req.SourceAddr,
			)
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Module not found",
				fmt.Sprintf(
					"The module address %q could not be resolved.\n\n"+
						"If you intended this as a path relative to the current "+
						"module, use \"./%s\" instead. The \"./\" prefix "+
						"indicates that the address is a relative filesystem path.",
					req.SourceAddr, req.SourceAddr,
				),
			))
		} else {
			// Errors returned by go-getter have very inconsistent quality as
			// end-user error messages, but for now we're accepting that because
			// we have no way to recognize any specific errors to improve them
			// and masking the error entirely would hide valuable diagnostic
			// information from the user.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to download module",
				fmt.Sprintf("Error attempting to download module %q (%s:%d) source code from %q: %s", req.Name, req.CallPos.Filename, req.CallPos.Line, packageAddr, err),
			))
		}
		return nil, diags

	}

	log.Printf("[TRACE] ModuleInstaller: %s %q was downloaded to %s", key, req.SourceAddr, modDir)

	mod, mDiags := earlyconfig.LoadModule(modDir)
	if mod == nil {
		// nil indicates missing or unreadable directory, so we'll
		// discard the returned diags and return a more specific
		// error message here. For go-getter modules this actually
		// indicates a bug in the code above, since it's not the
		// user's responsibility to create the directory in this case.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unreadable module directory",
			fmt.Sprintf("The directory %s could not be read. This is a bug in Terraform and should be reported.", modDir),
		))
	} else {
		diags = append(diags, mDiags...)
	}

	// Note the local location in our manifest.
	manifest[key] = modsdir.Record{
		Key:        key,
		Dir:        modDir,
		SourceAddr: req.SourceAddr,
	}
	log.Printf("[DEBUG] Module installer: %s installed at %s", key, modDir)
	hooks.Install(key, nil, modDir)

	return mod, diags
}

func (i *ModuleInstaller) packageInstallPath(modulePath addrs.Module) string {
	return filepath.Join(i.modsDir, strings.Join(modulePath, "."))
}
