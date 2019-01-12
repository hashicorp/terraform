package configload

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/configs"
)

const initFromModuleRootCallName = "root"
const initFromModuleRootKeyPrefix = initFromModuleRootCallName + "."

// InitDirFromModule populates the given directory (which must exist and be
// empty) with the contents of the module at the given source address.
//
// It does this by installing the given module and all of its descendent
// modules in a temporary root directory and then copying the installed
// files into suitable locations. As a consequence, any diagnostics it
// generates will reveal the location of this temporary directory to the
// user.
//
// This rather roundabout installation approach is taken to ensure that
// installation proceeds in a manner identical to normal module installation.
//
// If the given source address specifies a sub-directory of the given
// package then only the sub-directory and its descendents will be copied
// into the given root directory, which will cause any relative module
// references using ../ from that module to be unresolvable. Error diagnostics
// are produced in that case, to prompt the user to rewrite the source strings
// to be absolute references to the original remote module.
//
// This can be installed only on a loder that can install modules, and will
// panic otherwise. Use CanInstallModules to determine if this method can be
// used, or refer to the documentation of that method for situations where
// install ability is guaranteed.
func (l *Loader) InitDirFromModule(rootDir, sourceAddr string, hooks InstallHooks) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// The way this function works is pretty ugly, but we accept it because
	// -from-module is a less important case than normal module installation
	// and so it's better to keep this ugly complexity out here rather than
	// adding even more complexity to the normal module installer.

	// The target directory must exist but be empty.
	{
		entries, err := l.modules.FS.ReadDir(rootDir)
		if err != nil {
			if os.IsNotExist(err) {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Target directory does not exist",
					Detail:   fmt.Sprintf("Cannot initialize non-existent directory %s.", rootDir),
				})
			} else {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Failed to read target directory",
					Detail:   fmt.Sprintf("Error reading %s to ensure it is empty: %s.", rootDir, err),
				})
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
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Can't populate non-empty directory",
				Detail:   fmt.Sprintf("The target directory %s is not empty, so it cannot be initialized with the -from-module=... option.", rootDir),
			})
			return diags
		}
	}

	// We use a hidden sub-loader to manage our inner installation directory,
	// but it shares dependencies with the receiver to allow it to access the
	// same remote resources and ensure it populates the same source code
	// cache in case .
	subLoader := &Loader{
		parser:  l.parser,
		modules: l.modules, // this is a shallow copy, so we can safely mutate below
	}

	// Our sub-loader will have its own independent manifest and install
	// directory, so we can install with it and know we won't interfere
	// with the receiver.
	subLoader.modules.manifest = make(moduleManifest)
	subLoader.modules.Dir = filepath.Join(rootDir, ".terraform/init-from-module")

	log.Printf("[DEBUG] using a child module loader in %s to initialize working directory from %q", subLoader.modules.Dir, sourceAddr)

	subLoader.modules.FS.RemoveAll(subLoader.modules.Dir) // if this fails then we'll fail on MkdirAll below too

	err := subLoader.modules.FS.MkdirAll(subLoader.modules.Dir, os.ModePerm)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to create temporary directory",
			Detail:   fmt.Sprintf("Failed to create temporary directory %s: %s.", subLoader.modules.Dir, err),
		})
		return diags
	}

	fakeFilename := fmt.Sprintf("-from-module=%q", sourceAddr)
	fakeRange := hcl.Range{
		Filename: fakeFilename,
		Start: hcl.Pos{
			Line:   1,
			Column: 1,
			Byte:   0,
		},
		End: hcl.Pos{
			Line:   1,
			Column: len(fakeFilename) + 1, // not accurate if the address contains unicode, but irrelevant since we have no source cache for this anyway
			Byte:   len(fakeFilename),
		},
	}

	// -from-module allows relative paths but it's different than a normal
	// module address where it'd be resolved relative to the module call
	// (which is synthetic, here.) To address this, we'll just patch up any
	// relative paths to be absolute paths before we run, ensuring we'll
	// get the right result. This also, as an important side-effect, ensures
	// that the result will be "downloaded" with go-getter (copied from the
	// source location), rather than just recorded as a relative path.
	{
		maybePath := filepath.ToSlash(sourceAddr)
		if maybePath == "." || strings.HasPrefix(maybePath, "./") || strings.HasPrefix(maybePath, "../") {
			if wd, err := os.Getwd(); err == nil {
				sourceAddr = filepath.Join(wd, sourceAddr)
				log.Printf("[TRACE] -from-module relative path rewritten to absolute path %s", sourceAddr)
			}
		}
	}

	// Now we need to create an artificial root module that will seed our
	// installation process.
	fakeRootModule := &configs.Module{
		ModuleCalls: map[string]*configs.ModuleCall{
			initFromModuleRootCallName: &configs.ModuleCall{
				Name: initFromModuleRootCallName,

				SourceAddr:      sourceAddr,
				SourceAddrRange: fakeRange,
				SourceSet:       true,

				DeclRange: fakeRange,
			},
		},
	}

	// wrapHooks filters hook notifications to only include Download calls
	// and to trim off the initFromModuleRootCallName prefix. We'll produce
	// our own Install notifications directly below.
	wrapHooks := installHooksInitDir{
		Wrapped: hooks,
	}
	getter := reusingGetter{}
	instDiags := subLoader.installDescendentModules(fakeRootModule, rootDir, true, wrapHooks, getter)
	diags = append(diags, instDiags...)
	if instDiags.HasErrors() {
		return diags
	}

	// If all of that succeeded then we'll now migrate what was installed
	// into the final directory structure.
	modulesDir := l.modules.Dir
	err = subLoader.modules.FS.MkdirAll(modulesDir, os.ModePerm)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to create local modules directory",
			Detail:   fmt.Sprintf("Failed to create modules directory %s: %s.", modulesDir, err),
		})
		return diags
	}

	manifest := subLoader.modules.manifest
	recordKeys := make([]string, 0, len(manifest))
	for k := range manifest {
		recordKeys = append(recordKeys, k)
	}
	sort.Strings(recordKeys)

	for _, recordKey := range recordKeys {
		record := manifest[recordKey]

		if record.Key == initFromModuleRootCallName {
			// We've found the module the user requested, which we must
			// now copy into rootDir so it can be used directly.
			log.Printf("[TRACE] copying new root module from %s to %s", record.Dir, rootDir)
			err := copyDir(rootDir, record.Dir)
			if err != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Failed to copy root module",
					Detail:   fmt.Sprintf("Error copying root module %q from %s to %s: %s.", sourceAddr, record.Dir, rootDir, err),
				})
				continue
			}

			// We'll try to load the newly-copied module here just so we can
			// sniff for any module calls that ../ out of the root directory
			// and must thus be rewritten to be absolute addresses again.
			// For now we can't do this rewriting automatically, but we'll
			// generate an error to help the user do it manually.
			mod, _ := l.parser.LoadConfigDir(rootDir) // ignore diagnostics since we're just doing value-add here anyway
			for _, mc := range mod.ModuleCalls {
				if pathTraversesUp(sourceAddr) {
					packageAddr, givenSubdir := splitAddrSubdir(sourceAddr)
					newSubdir := filepath.Join(givenSubdir, mc.SourceAddr)
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
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Root module references parent directory",
						Detail:   fmt.Sprintf("The requested module %q refers to a module via its parent directory. To use this as a new root module this source string must be rewritten as a remote source address, such as %q.", sourceAddr, newAddr),
						Subject:  &mc.SourceAddrRange,
					})
					continue
				}
			}

			l.modules.manifest[""] = moduleRecord{
				Key: "",
				Dir: rootDir,
			}
			continue
		}

		if !strings.HasPrefix(record.Key, initFromModuleRootKeyPrefix) {
			// Ignore the *real* root module, whose key is empty, since
			// we're only interested in the module named "root" and its
			// descendents.
			continue
		}

		newKey := record.Key[len(initFromModuleRootKeyPrefix):]
		instPath := filepath.Join(l.modules.Dir, newKey)
		tempPath := filepath.Join(subLoader.modules.Dir, record.Key)

		// tempPath won't be present for a module that was installed from
		// a relative path, so in that case we just record the installation
		// directory and assume it was already copied into place as part
		// of its parent.
		if _, err := os.Stat(tempPath); err != nil {
			if !os.IsNotExist(err) {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Failed to stat temporary module install directory",
					Detail:   fmt.Sprintf("Error from stat %s for module %s: %s.", instPath, newKey, err),
				})
				continue
			}

			var parentKey string
			if lastDot := strings.LastIndexByte(newKey, '.'); lastDot != -1 {
				parentKey = newKey[:lastDot]
			} else {
				parentKey = "" // parent is the root module
			}

			parentOld := manifest[initFromModuleRootKeyPrefix+parentKey]
			parentNew := l.modules.manifest[parentKey]

			// We need to figure out which portion of our directory is the
			// parent package path and which portion is the subdirectory
			// under that.
			baseDirRel, err := filepath.Rel(parentOld.Dir, record.Dir)
			if err != nil {
				// Should never happen, because we constructed both directories
				// from the same base and so they must have a common prefix.
				panic(err)
			}

			newDir := filepath.Join(parentNew.Dir, baseDirRel)
			log.Printf("[TRACE] relative reference for %s rewritten from %s to %s", newKey, record.Dir, newDir)
			newRecord := record // shallow copy
			newRecord.Dir = newDir
			newRecord.Key = newKey
			l.modules.manifest[newKey] = newRecord
			hooks.Install(newRecord.Key, newRecord.Version, newRecord.Dir)
			continue
		}

		err = subLoader.modules.FS.MkdirAll(instPath, os.ModePerm)
		if err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Failed to create module install directory",
				Detail:   fmt.Sprintf("Error creating directory %s for module %s: %s.", instPath, newKey, err),
			})
			continue
		}

		// We copy rather than "rename" here because renaming between directories
		// can be tricky in edge-cases like network filesystems, etc.
		log.Printf("[TRACE] copying new module %s from %s to %s", newKey, record.Dir, instPath)
		err := copyDir(instPath, tempPath)
		if err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Failed to copy descendent module",
				Detail:   fmt.Sprintf("Error copying module %q from %s to %s: %s.", newKey, tempPath, rootDir, err),
			})
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
		l.modules.manifest[newKey] = newRecord
		hooks.Install(newRecord.Key, newRecord.Version, newRecord.Dir)
	}

	err = l.modules.writeModuleManifestSnapshot()
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to write module manifest",
			Detail:   fmt.Sprintf("Error writing module manifest: %s.", err),
		})
	}

	if !diags.HasErrors() {
		// Try to clean up our temporary directory, but don't worry if we don't
		// succeed since it shouldn't hurt anything.
		subLoader.modules.FS.RemoveAll(subLoader.modules.Dir)
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
	Wrapped InstallHooks
	InstallHooksImpl
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
