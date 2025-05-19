// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package initwd

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/copy"
	"github.com/hashicorp/terraform/internal/getmodules"
	"github.com/hashicorp/terraform/internal/getmodules/moduleaddrs"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/internal/modsdir"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const initFromModuleRootCallName = "root"
const initFromModuleRootFilename = "<main configuration>"
const initFromModuleRootKeyPrefix = initFromModuleRootCallName + "."

// DirFromModule populates the given directory (which must exist and be
// empty) with the contents of the module at the given source address.
//
// It does this by installing the given module and all of its descendant
// modules in a temporary root directory and then copying the installed
// files into suitable locations. As a consequence, any diagnostics it
// generates will reveal the location of this temporary directory to the
// user.
//
// This rather roundabout installation approach is taken to ensure that
// installation proceeds in a manner identical to normal module installation.
//
// If the given source address specifies a sub-directory of the given
// package then only the sub-directory and its descendants will be copied
// into the given root directory, which will cause any relative module
// references using ../ from that module to be unresolvable. Error diagnostics
// are produced in that case, to prompt the user to rewrite the source strings
// to be absolute references to the original remote module.
func DirFromModule(ctx context.Context, loader *configload.Loader, rootDir, modulesDir, sourceAddrStr string, reg *registry.Client, hooks ModuleInstallHooks) tfdiags.Diagnostics {

	var diags tfdiags.Diagnostics

	// The way this function works is pretty ugly, but we accept it because
	// -from-module is a less important case than normal module installation
	// and so it's better to keep this ugly complexity out here rather than
	// adding even more complexity to the normal module installer.

	// The target directory must exist but be empty.
	{
		entries, err := ioutil.ReadDir(rootDir)
		if err != nil {
			if os.IsNotExist(err) {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Target directory does not exist",
					fmt.Sprintf("Cannot initialize non-existent directory %s.", rootDir),
				))
			} else {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to read target directory",
					fmt.Sprintf("Error reading %s to ensure it is empty: %s.", rootDir, err),
				))
			}
			return diags
		}
		haveEntries := false
		for _, entry := range entries {
			if entry.Name() == "." || entry.Name() == ".." || entry.Name() == ".terraform" {
				continue
			}
			haveEntries = true
		}
		if haveEntries {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Can't populate non-empty directory",
				fmt.Sprintf("The target directory %s is not empty, so it cannot be initialized with the -from-module=... option.", rootDir),
			))
			return diags
		}
	}

	instDir := filepath.Join(rootDir, ".terraform/init-from-module")
	inst := NewModuleInstaller(instDir, loader, reg)
	log.Printf("[DEBUG] installing modules in %s to initialize working directory from %q", instDir, sourceAddrStr)
	os.RemoveAll(instDir) // if this fails then we'll fail on MkdirAll below too
	err := os.MkdirAll(instDir, os.ModePerm)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to create temporary directory",
			fmt.Sprintf("Failed to create temporary directory %s: %s.", instDir, err),
		))
		return diags
	}

	instManifest := make(modsdir.Manifest)
	retManifest := make(modsdir.Manifest)

	// -from-module allows relative paths but it's different than a normal
	// module address where it'd be resolved relative to the module call
	// (which is synthetic, here.) To address this, we'll just patch up any
	// relative paths to be absolute paths before we run, ensuring we'll
	// get the right result. This also, as an important side-effect, ensures
	// that the result will be "downloaded" with go-getter (copied from the
	// source location), rather than just recorded as a relative path.
	{
		maybePath := filepath.ToSlash(sourceAddrStr)
		if maybePath == "." || strings.HasPrefix(maybePath, "./") || strings.HasPrefix(maybePath, "../") {
			if wd, err := os.Getwd(); err == nil {
				sourceAddrStr = filepath.Join(wd, sourceAddrStr)
				log.Printf("[TRACE] -from-module relative path rewritten to absolute path %s", sourceAddrStr)
			}
		}
	}

	// Now we need to create an artificial root module that will seed our
	// installation process.
	sourceAddr, err := moduleaddrs.ParseModuleSource(sourceAddrStr)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid module source address",
			fmt.Sprintf("Failed to parse module source address: %s", err),
		))
	}
	fakeRootModule := &configs.Module{
		ModuleCalls: map[string]*configs.ModuleCall{
			initFromModuleRootCallName: {
				Name:       initFromModuleRootCallName,
				SourceAddr: sourceAddr,
				DeclRange: hcl.Range{
					Filename: initFromModuleRootFilename,
					Start:    hcl.InitialPos,
					End:      hcl.InitialPos,
				},
			},
		},
		ProviderRequirements: &configs.RequiredProviders{},
	}

	// wrapHooks filters hook notifications to only include Download calls
	// and to trim off the initFromModuleRootCallName prefix. We'll produce
	// our own Install notifications directly below.
	wrapHooks := installHooksInitDir{
		Wrapped: hooks,
	}
	// Create a manifest record for the root module. This will be used if
	// there are any relative-pathed modules in the root.
	instManifest[""] = modsdir.Record{
		Key: "",
		Dir: rootDir,
	}
	fetcher := getmodules.NewPackageFetcher()

	walker := inst.moduleInstallWalker(ctx, instManifest, true, wrapHooks, fetcher)
	_, cDiags := inst.installDescendantModules(fakeRootModule, instManifest, walker, true)
	if cDiags.HasErrors() {
		return diags.Append(cDiags)
	}

	// If all of that succeeded then we'll now migrate what was installed
	// into the final directory structure.
	err = os.MkdirAll(modulesDir, os.ModePerm)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to create local modules directory",
			fmt.Sprintf("Failed to create modules directory %s: %s.", modulesDir, err),
		))
		return diags
	}

	recordKeys := make([]string, 0, len(instManifest))
	for k := range instManifest {
		recordKeys = append(recordKeys, k)
	}
	sort.Strings(recordKeys)

	for _, recordKey := range recordKeys {
		record := instManifest[recordKey]

		if record.Key == initFromModuleRootCallName {
			// We've found the module the user requested, which we must
			// now copy into rootDir so it can be used directly.
			log.Printf("[TRACE] copying new root module from %s to %s", record.Dir, rootDir)
			err := copy.CopyDir(rootDir, record.Dir)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to copy root module",
					fmt.Sprintf("Error copying root module %q from %s to %s: %s.", sourceAddrStr, record.Dir, rootDir, err),
				))
				continue
			}

			// We'll try to load the newly-copied module here just so we can
			// sniff for any module calls that ../ out of the root directory
			// and must thus be rewritten to be absolute addresses again.
			// For now we can't do this rewriting automatically, but we'll
			// generate an error to help the user do it manually.
			mod, _ := loader.Parser().LoadConfigDir(rootDir) // ignore diagnostics since we're just doing value-add here anyway
			if mod != nil {
				for _, mc := range mod.ModuleCalls {
					if pathTraversesUp(mc.SourceAddrRaw) {
						packageAddr, givenSubdir := moduleaddrs.SplitPackageSubdir(sourceAddrStr)
						newSubdir := filepath.Join(givenSubdir, mc.SourceAddrRaw)
						if pathTraversesUp(newSubdir) {
							// This should never happen in any reasonable
							// configuration since this suggests a path that
							// traverses up out of the package root. We'll just
							// ignore this, since we'll fail soon enough anyway
							// trying to resolve this path when this module is
							// loaded.
							continue
						}

						var newAddr = packageAddr
						if newSubdir != "" {
							newAddr = fmt.Sprintf("%s//%s", newAddr, filepath.ToSlash(newSubdir))
						}
						diags = diags.Append(tfdiags.Sourceless(
							tfdiags.Error,
							"Root module references parent directory",
							fmt.Sprintf("The requested module %q refers to a module via its parent directory. To use this as a new root module this source string must be rewritten as a remote source address, such as %q.", sourceAddrStr, newAddr),
						))
						continue
					}
				}
			}

			retManifest[""] = modsdir.Record{
				Key: "",
				Dir: rootDir,
			}
			continue
		}

		if !strings.HasPrefix(record.Key, initFromModuleRootKeyPrefix) {
			// Ignore the *real* root module, whose key is empty, since
			// we're only interested in the module named "root" and its
			// descendants.
			continue
		}

		newKey := record.Key[len(initFromModuleRootKeyPrefix):]
		instPath := filepath.Join(modulesDir, newKey)
		tempPath := filepath.Join(instDir, record.Key)

		// tempPath won't be present for a module that was installed from
		// a relative path, so in that case we just record the installation
		// directory and assume it was already copied into place as part
		// of its parent.
		if _, err := os.Stat(tempPath); err != nil {
			if !os.IsNotExist(err) {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to stat temporary module install directory",
					fmt.Sprintf("Error from stat %s for module %s: %s.", instPath, newKey, err),
				))
				continue
			}

			var parentKey string
			if lastDot := strings.LastIndexByte(newKey, '.'); lastDot != -1 {
				parentKey = newKey[:lastDot]
			}

			var parentOld modsdir.Record
			// "" is the root module; all other modules get `root.` added as a prefix
			if parentKey == "" {
				parentOld = instManifest[parentKey]
			} else {
				parentOld = instManifest[initFromModuleRootKeyPrefix+parentKey]
			}
			parentNew := retManifest[parentKey]

			// We need to figure out which portion of our directory is the
			// parent package path and which portion is the subdirectory
			// under that.
			var baseDirRel string
			baseDirRel, err = filepath.Rel(parentOld.Dir, record.Dir)
			if err != nil {
				// This error may occur when installing a local module with a
				// relative path, for e.g. if the source is in a directory above
				// the destination ("../")
				if parentOld.Dir == "." {
					absDir, err := filepath.Abs(parentOld.Dir)
					if err != nil {
						diags = diags.Append(tfdiags.Sourceless(
							tfdiags.Error,
							"Failed to determine module install directory",
							fmt.Sprintf("Error determine relative source directory for module %s: %s.", newKey, err),
						))
						continue
					}
					baseDirRel, err = filepath.Rel(absDir, record.Dir)
					if err != nil {
						diags = diags.Append(tfdiags.Sourceless(
							tfdiags.Error,
							"Failed to determine relative module source location",
							fmt.Sprintf("Error determining relative source for module %s: %s.", newKey, err),
						))
						continue
					}
				} else {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Failed to determine relative module source location",
						fmt.Sprintf("Error determining relative source for module %s: %s.", newKey, err),
					))
				}
			}

			newDir := filepath.Join(parentNew.Dir, baseDirRel)
			log.Printf("[TRACE] relative reference for %s rewritten from %s to %s", newKey, record.Dir, newDir)
			newRecord := record // shallow copy
			newRecord.Dir = newDir
			newRecord.Key = newKey
			retManifest[newKey] = newRecord
			hooks.Install(newRecord.Key, newRecord.Version, newRecord.Dir)
			continue
		}

		err = os.MkdirAll(instPath, os.ModePerm)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to create module install directory",
				fmt.Sprintf("Error creating directory %s for module %s: %s.", instPath, newKey, err),
			))
			continue
		}

		// We copy rather than "rename" here because renaming between directories
		// can be tricky in edge-cases like network filesystems, etc.
		log.Printf("[TRACE] copying new module %s from %s to %s", newKey, record.Dir, instPath)
		err := copy.CopyDir(instPath, tempPath)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to copy descendant module",
				fmt.Sprintf("Error copying module %q from %s to %s: %s.", newKey, tempPath, rootDir, err),
			))
			continue
		}

		subDir, err := filepath.Rel(tempPath, record.Dir)
		if err != nil {
			// Should never happen, because we constructed both directories
			// from the same base and so they must have a common prefix.
			panic(err)
		}

		newRecord := record // shallow copy
		newRecord.Dir = filepath.Join(instPath, subDir)
		newRecord.Key = newKey
		retManifest[newKey] = newRecord
		hooks.Install(newRecord.Key, newRecord.Version, newRecord.Dir)
	}

	retManifest.WriteSnapshotToDir(modulesDir)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to write module manifest",
			fmt.Sprintf("Error writing module manifest: %s.", err),
		))
	}

	if !diags.HasErrors() {
		// Try to clean up our temporary directory, but don't worry if we don't
		// succeed since it shouldn't hurt anything.
		os.RemoveAll(instDir)
	}

	return diags
}

func pathTraversesUp(path string) bool {
	return strings.HasPrefix(filepath.ToSlash(path), "../")
}

// installHooksInitDir is an adapter wrapper for an InstallHooks that
// does some fakery to make downloads look like they are happening in their
// final locations, rather than in the temporary loader we use.
//
// It also suppresses "Install" calls entirely, since InitDirFromModule
// does its own installation steps after the initial installation pass
// has completed.
type installHooksInitDir struct {
	Wrapped ModuleInstallHooks
	ModuleInstallHooksImpl
}

func (h installHooksInitDir) Download(moduleAddr, packageAddr string, version *version.Version) {
	if !strings.HasPrefix(moduleAddr, initFromModuleRootKeyPrefix) {
		// We won't announce the root module, since hook implementations
		// don't expect to see that and the caller will usually have produced
		// its own user-facing notification about what it's doing anyway.
		return
	}

	trimAddr := moduleAddr[len(initFromModuleRootKeyPrefix):]
	h.Wrapped.Download(trimAddr, packageAddr, version)
}
