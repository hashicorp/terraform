package configs

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl2/hcl"
)

// LoadConfigDir reads the .tf and .tf.json files in the given directory
// as config files (using LoadConfigFile) and then combines these files into
// a single Module.
//
// If this method returns nil, that indicates that the given directory does not
// exist at all or could not be opened for some reason. Callers may wish to
// detect this case and ignore the returned diagnostics so that they can
// produce a more context-aware error message in that case.
//
// If this method returns a non-nil module while error diagnostics are returned
// then the module may be incomplete but can be used carefully for static
// analysis.
//
// This file does not consider a directory with no files to be an error, and
// will simply return an empty module in that case. Callers should first call
// Parser.IsConfigDir if they wish to recognize that situation.
//
// .tf files are parsed using the HCL native syntax while .tf.json files are
// parsed using the HCL JSON syntax.
func (p *Parser) LoadConfigDir(path string) (*Module, hcl.Diagnostics) {
	primaryPaths, overridePaths, diags := p.dirFiles(path)
	if diags.HasErrors() {
		return nil, diags
	}

	primary, fDiags := p.loadFiles(primaryPaths, false)
	diags = append(diags, fDiags...)
	override, fDiags := p.loadFiles(overridePaths, true)
	diags = append(diags, fDiags...)

	mod, modDiags := NewModule(primary, override)
	diags = append(diags, modDiags...)

	mod.SourceDir = path

	return mod, diags
}

// IsConfigDir determines whether the given path refers to a directory that
// exists and contains at least one Terraform config file (with a .tf or
// .tf.json extension.)
func (p *Parser) IsConfigDir(path string) bool {
	primaryPaths, overridePaths, _ := p.dirFiles(path)
	return (len(primaryPaths) + len(overridePaths)) > 0
}

func (p *Parser) loadFiles(paths []string, override bool) ([]*File, hcl.Diagnostics) {
	var files []*File
	var diags hcl.Diagnostics

	for _, path := range paths {
		var f *File
		var fDiags hcl.Diagnostics
		if override {
			f, fDiags = p.LoadConfigFileOverride(path)
		} else {
			f, fDiags = p.LoadConfigFile(path)
		}
		diags = append(diags, fDiags...)
		if f != nil {
			files = append(files, f)
		}
	}

	return files, diags
}

func (p *Parser) dirFiles(dir string) (primary, override []string, diags hcl.Diagnostics) {
	infos, err := p.fs.ReadDir(dir)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to read module directory",
			Detail:   fmt.Sprintf("Module directory %s does not exist or cannot be read.", dir),
		})
		return
	}

	for _, info := range infos {
		if info.IsDir() {
			// We only care about files
			continue
		}

		name := info.Name()
		ext := fileExt(name)
		if ext == "" || IsIgnoredFile(name) {
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

// IsIgnoredFile returns true if the given filename (which must not have a
// directory path ahead of it) should be ignored as e.g. an editor swap file.
func IsIgnoredFile(name string) bool {
	return strings.HasPrefix(name, ".") || // Unix-like hidden files
		strings.HasSuffix(name, "~") || // vim
		strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#") // emacs
}
