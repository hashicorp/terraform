// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

const (
	DefaultTestDirectory = "tests"
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
	primaryPaths, overridePaths, _, diags := p.dirFiles(path, "")
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

// LoadConfigDirWithTests matches LoadConfigDir, but the return Module also
// contains any relevant .tftest.hcl files.
func (p *Parser) LoadConfigDirWithTests(path string, testDirectory string) (*Module, hcl.Diagnostics) {
	primaryPaths, overridePaths, testPaths, diags := p.dirFiles(path, testDirectory)
	if diags.HasErrors() {
		return nil, diags
	}

	primary, fDiags := p.loadFiles(primaryPaths, false)
	diags = append(diags, fDiags...)
	override, fDiags := p.loadFiles(overridePaths, true)
	diags = append(diags, fDiags...)
	tests, fDiags := p.loadTestFiles(path, testPaths)
	diags = append(diags, fDiags...)

	mod, modDiags := NewModuleWithTests(primary, override, tests)
	diags = append(diags, modDiags...)

	mod.SourceDir = path

	return mod, diags
}

// ConfigDirFiles returns lists of the primary and override files configuration
// files in the given directory.
//
// If the given directory does not exist or cannot be read, error diagnostics
// are returned. If errors are returned, the resulting lists may be incomplete.
func (p Parser) ConfigDirFiles(dir string) (primary, override []string, diags hcl.Diagnostics) {
	primary, override, _, diags = p.dirFiles(dir, "")
	return primary, override, diags
}

// ConfigDirFilesWithTests matches ConfigDirFiles except it also returns the
// paths to any test files within the module.
func (p Parser) ConfigDirFilesWithTests(dir string, testDirectory string) (primary, override, tests []string, diags hcl.Diagnostics) {
	return p.dirFiles(dir, testDirectory)
}

// IsConfigDir determines whether the given path refers to a directory that
// exists and contains at least one Terraform config file (with a .tf or
// .tf.json extension.). Note, we explicitely exclude checking for tests here
// as tests must live alongside actual .tf config files.
func (p *Parser) IsConfigDir(path string) bool {
	primaryPaths, overridePaths, _, _ := p.dirFiles(path, "")
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

// dirFiles finds Terraform configuration files within dir, splitting them into
// primary and override files based on the filename.
//
// If testsDir is not empty, dirFiles will also retrieve Terraform testing files
// both directly within dir and within testsDir as a subdirectory of dir. In
// this way, testsDir acts both as a direction to retrieve test files within the
// main direction and as the location for additional test files.
func (p *Parser) dirFiles(dir string, testsDir string) (primary, override, tests []string, diags hcl.Diagnostics) {
	includeTests := len(testsDir) > 0

	if includeTests {
		testPath := path.Join(dir, testsDir)

		infos, err := p.fs.ReadDir(testPath)
		if err != nil {
			// Then we couldn't read from the testing directory for some reason.

			if os.IsNotExist(err) {
				// Then this means the testing directory did not exist.
				// We won't actually stop loading the rest of the configuration
				// for this, we will add a warning to explain to the user why
				// test files weren't processed but leave it at that.
				if testsDir != DefaultTestDirectory {
					// We'll only add the warning if a directory other than the
					// default has been requested. If the user is just loading
					// the default directory then we have no expectation that
					// it should actually exist.
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  "Test directory does not exist",
						Detail:   fmt.Sprintf("Requested test directory %s does not exist.", testPath),
					})
				}
			} else {
				// Then there is some other reason we couldn't load. We will
				// treat this as a full error.
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Failed to read test directory",
					Detail:   fmt.Sprintf("Test directory %s could not be read: %v.", testPath, err),
				})

				// We'll also stop loading the rest of the config for this.
				return
			}

		} else {
			for _, testInfo := range infos {
				if testInfo.IsDir() || IsIgnoredFile(testInfo.Name()) {
					continue
				}

				if strings.HasSuffix(testInfo.Name(), ".tftest.hcl") || strings.HasSuffix(testInfo.Name(), ".tftest.json") {
					tests = append(tests, filepath.Join(testPath, testInfo.Name()))
				}
			}
		}

	}

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
			// We only care about terraform configuration files.
			continue
		}

		name := info.Name()
		ext := fileExt(name)
		if ext == "" || IsIgnoredFile(name) {
			continue
		}

		if ext == ".tftest.hcl" || ext == ".tftest.json" {
			if includeTests {
				tests = append(tests, filepath.Join(dir, name))
			}
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
// configuration files.
//
// Unlike the methods of the Parser type, this function always consults the
// real filesystem, and thus it isn't appropriate to use when working with
// configuration loaded from a plan file.
func IsEmptyDir(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return true, nil
	}

	p := NewParser(nil)
	fs, os, _, diags := p.dirFiles(path, "")
	if diags.HasErrors() {
		return false, diags
	}

	return len(fs) == 0 && len(os) == 0, nil
}
