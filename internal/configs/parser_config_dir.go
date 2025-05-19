// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

const (
	DefaultTestDirectory = "tests"
)

// LoadConfigDir reads the configuration files in the given directory
// as config files (using LoadConfigFile) and then combines these files into
// a single Module.
//
// Main terraform configuration files (.tf and .tf.json) are loaded as the primary
// module, while override files (override.tf and *_override.tf) are loaded as
// overrides.
// Optionally, test files (.tftest.hcl and .tftest.json) can be loaded from
// a subdirectory of the given directory, which is specified by the
// MatchTestFiles option, or from the default test directory.
// If this option is not specified, test files will not be loaded.
// Query files (.tfquery.hcl) are also loaded from the given directory.
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
func (p *Parser) LoadConfigDir(path string, opts ...Option) (*Module, hcl.Diagnostics) {
	fileSet, diags := p.dirFileSet(path, opts...)
	if diags.HasErrors() {
		return nil, diags
	}

	// Load the .tf configuration files
	primary, fDiags := p.loadFiles(fileSet.Primary, false)
	diags = diags.Extend(fDiags)

	override, fDiags := p.loadFiles(fileSet.Override, true)
	diags = diags.Extend(fDiags)

	// Initialize the module
	mod, modDiags := NewModule(primary, override)
	diags = diags.Extend(modDiags)

	// Check if we need to load test files
	if len(fileSet.Tests) > 0 {
		testFiles, fDiags := p.loadTestFiles(path, fileSet.Tests)
		diags = diags.Extend(fDiags)
		if mod != nil {
			mod.Tests = testFiles
		}
	}
	// Check if we need to load query files
	if len(fileSet.Queries) > 0 {
		queryFiles, fDiags := p.loadQueryFiles(path, fileSet.Queries)
		diags = append(diags, fDiags...)
		if mod != nil {
			for _, qf := range queryFiles {
				diags = diags.Extend(mod.appendQueryFile(qf))
			}
		}
	}

	if mod != nil {
		mod.SourceDir = path
	}

	return mod, diags
}

// LoadConfigDirWithTests matches LoadConfigDir, but the return Module also
// contains any relevant .tftest.hcl files.
func (p *Parser) LoadConfigDirWithTests(path string, testDirectory string) (*Module, hcl.Diagnostics) {
	return p.LoadConfigDir(path, MatchTestFiles(testDirectory))
}

func (p *Parser) LoadMockDataDir(dir string, useForPlanDefault bool, source hcl.Range) (*MockData, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	infos, err := p.fs.ReadDir(dir)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to read mock data directory",
			Detail:   fmt.Sprintf("Mock data directory %s does not exist or cannot be read.", dir),
			Subject:  source.Ptr(),
		})
		return nil, diags
	}

	var files []string
	for _, info := range infos {
		if info.IsDir() {
			// We only care about terraform configuration files.
			continue
		}

		name := info.Name()
		if !(strings.HasSuffix(name, ".tfmock.hcl") || strings.HasSuffix(name, ".tfmock.json")) {
			continue
		}

		if IsIgnoredFile(name) {
			continue
		}

		files = append(files, filepath.Join(dir, name))
	}

	var data *MockData
	for _, file := range files {
		current, currentDiags := p.LoadMockDataFile(file, useForPlanDefault)
		diags = append(diags, currentDiags...)
		if data != nil {
			diags = append(diags, data.Merge(current, false)...)
			continue
		}
		data = current
	}
	return data, diags
}

// ConfigDirFiles returns lists of the primary and override files configuration
// files in the given directory.
//
// If the given directory does not exist or cannot be read, error diagnostics
// are returned. If errors are returned, the resulting lists may be incomplete.
func (p Parser) ConfigDirFiles(dir string, opts ...Option) (primary, override []string, diags hcl.Diagnostics) {
	fSet, diags := p.dirFileSet(dir, opts...)
	return fSet.Primary, fSet.Override, diags
}

// IsConfigDir determines whether the given path refers to a directory that
// exists and contains at least one Terraform config file (with a .tf or
// .tf.json extension.). Note, we explicitely exclude checking for tests here
// as tests must live alongside actual .tf config files. Same goes for query files.
func (p *Parser) IsConfigDir(path string) bool {
	pathSet, _ := p.dirFileSet(path)
	return (len(pathSet.Primary) + len(pathSet.Override)) > 0
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

func (p *Parser) loadTestFiles(basePath string, paths []string) (map[string]*TestFile, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	tfs := make(map[string]*TestFile)
	for _, path := range paths {
		tf, fDiags := p.LoadTestFile(path)
		diags = append(diags, fDiags...)
		if tf != nil {
			// We index test files relative to the module they are testing, so
			// the key is the relative path between basePath and path.
			relPath, err := filepath.Rel(basePath, path)
			if err != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Failed to calculate relative path",
					Detail:   fmt.Sprintf("Terraform could not calculate the relative path for test file %s and it has been skipped: %s", path, err),
				})
				continue
			}
			tfs[relPath] = tf
		}
	}

	return tfs, diags
}

func (p *Parser) loadQueryFiles(basePath string, paths []string) ([]*QueryFile, hcl.Diagnostics) {
	files := make([]*QueryFile, 0, len(paths))
	var diags hcl.Diagnostics

	for _, path := range paths {
		f, fDiags := p.LoadQueryFile(path)
		diags = append(diags, fDiags...)
		if f != nil {
			files = append(files, f)
		}
	}

	return files, diags
}

// fileExt returns the Terraform configuration extension of the given
// path, or a blank string if it is not a recognized extension.
func fileExt(path string) string {
	if strings.HasSuffix(path, ".tf") {
		return ".tf"
	} else if strings.HasSuffix(path, ".tf.json") {
		return ".tf.json"
	} else if strings.HasSuffix(path, ".tftest.hcl") {
		return ".tftest.hcl"
	} else if strings.HasSuffix(path, ".tftest.json") {
		return ".tftest.json"
	} else if strings.HasSuffix(path, ".tfquery.hcl") {
		return ".tfquery.hcl"
	} else if strings.HasSuffix(path, ".tfquery.json") {
		return ".tfquery.json"
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

// IsEmptyDir returns true if the given filesystem path contains no Terraform
// configuration or test files.
//
// Unlike the methods of the Parser type, this function always consults the
// real filesystem, and thus it isn't appropriate to use when working with
// configuration loaded from a plan file.
func IsEmptyDir(path, testDir string) (bool, error) {
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return true, nil
	}

	p := NewParser(nil)
	fSet, diags := p.dirFileSet(path, MatchTestFiles(testDir))
	if diags.HasErrors() {
		return false, diags
	}

	return len(fSet.Primary) == 0 && len(fSet.Override) == 0 && len(fSet.Tests) == 0, nil
}
