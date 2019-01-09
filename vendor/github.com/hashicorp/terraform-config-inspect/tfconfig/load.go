package tfconfig

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl2/hcl"
)

// LoadModule reads the directory at the given path and attempts to interpret
// it as a Terraform module.
func LoadModule(dir string) (*Module, Diagnostics) {

	// For broad compatibility here we actually have two separate loader
	// codepaths. The main one uses the new HCL parser and API and is intended
	// for configurations from Terraform 0.12 onwards (though will work for
	// many older configurations too), but we'll also fall back on one that
	// uses the _old_ HCL implementation so we can deal with some edge-cases
	// that are not valid in new HCL.

	module, diags := loadModule(dir)
	if diags.HasErrors() {
		// Try using the legacy HCL parser and see if we fare better.
		legacyModule, legacyDiags := loadModuleLegacyHCL(dir)
		if !legacyDiags.HasErrors() {
			legacyModule.init(legacyDiags)
			return legacyModule, legacyDiags
		}
	}

	module.init(diags)
	return module, diags
}

// IsModuleDir checks if the given path contains terraform configuration files.
// This allows the caller to decide how to handle directories that do not have tf files.
func IsModuleDir(dir string) bool {
	primaryPaths, _ := dirFiles(dir)
	if len(primaryPaths) == 0 {
		return false
	}
	return true
}

func (m *Module) init(diags Diagnostics) {
	// Fill in any additional provider requirements that are implied by
	// resource configurations, to avoid the caller from needing to apply
	// this logic itself. Implied requirements don't have version constraints,
	// but we'll make sure the requirement value is still non-nil in this
	// case so callers can easily recognize it.
	for _, r := range m.ManagedResources {
		if _, exists := m.RequiredProviders[r.Provider.Name]; !exists {
			m.RequiredProviders[r.Provider.Name] = []string{}
		}
	}
	for _, r := range m.DataResources {
		if _, exists := m.RequiredProviders[r.Provider.Name]; !exists {
			m.RequiredProviders[r.Provider.Name] = []string{}
		}
	}

	// We redundantly also reference the diagnostics from inside the module
	// object, primarily so that we can easily included in JSON-serialized
	// versions of the module object.
	m.Diagnostics = diags
}

func dirFiles(dir string) (primary []string, diags hcl.Diagnostics) {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to read module directory",
			Detail:   fmt.Sprintf("Module directory %s does not exist or cannot be read.", dir),
		})
		return
	}

	var override []string
	for _, info := range infos {
		if info.IsDir() {
			// We only care about files
			continue
		}

		name := info.Name()
		ext := fileExt(name)
		if ext == "" || isIgnoredFile(name) {
			continue
		}

		baseName := name[:len(name)-len(ext)] // strip extension
		isOverride := baseName == "override" || strings.HasSuffix(baseName, "_override")

		fullPath := filepath.Join(dir, name)
		if isOverride {
			override = append(override, fullPath)
		} else {
			primary = append(primary, fullPath)
		}
	}

	// We are assuming that any _override files will be logically named,
	// and processing the files in alphabetical order. Primaries first, then overrides.
	primary = append(primary, override...)

	return
}

// fileExt returns the Terraform configuration extension of the given
// path, or a blank string if it is not a recognized extension.
func fileExt(path string) string {
	if strings.HasSuffix(path, ".tf") {
		return ".tf"
	} else if strings.HasSuffix(path, ".tf.json") {
		return ".tf.json"
	} else {
		return ""
	}
}

// isIgnoredFile returns true if the given filename (which must not have a
// directory path ahead of it) should be ignored as e.g. an editor swap file.
func isIgnoredFile(name string) bool {
	return strings.HasPrefix(name, ".") || // Unix-like hidden files
		strings.HasSuffix(name, "~") || // vim
		strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#") // emacs
}
