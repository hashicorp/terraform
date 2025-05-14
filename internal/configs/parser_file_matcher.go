// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/spf13/afero"
)

// ConfigFileSet holds the different types of configuration files found in a directory.
type ConfigFileSet struct {
	Primary  []string // Regular .tf and .tf.json files
	Override []string // Override files (override.tf or *_override.tf)
	Tests    []string // Test files (.tftest.hcl or .tftest.json)
	Queries  []string // Query files (.tfquery.hcl)
}

// FileMatcher is an interface for components that can match and process specific file types
// in a Terraform module directory.

type FileMatcher interface {
	// Matches returns true if the given filename should be processed by this matcher
	Matches(name string) bool

	// DirFiles allows the matcher to process files in a directory
	// only relevant to its type.
	DirFiles(dir string, cfg *parserConfig, fileSet *ConfigFileSet) hcl.Diagnostics
}

// Option is a functional option type for configuring the parser
type Option func(*parserConfig)

type parserConfig struct {
	matchers      []FileMatcher
	testDirectory string
	fs            afero.Afero
}

// dirFileSet finds Terraform configuration files within directory dir
// and returns a ConfigFileSet containing the found files.
// It uses the given options to determine which types of files to look for
// and how to process them. The returned ConfigFileSet contains the paths
// to the found files, categorized by their type (primary, override, test, query).
func (p *Parser) dirFileSet(dir string, opts ...Option) (ConfigFileSet, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	fileSet := ConfigFileSet{
		Primary:  []string{},
		Override: []string{},
		Tests:    []string{},
		Queries:  []string{},
	}

	// Set up the parser configuration
	cfg := &parserConfig{
		// We always match .tf files
		matchers:      []FileMatcher{&moduleFiles{}},
		testDirectory: DefaultTestDirectory,
		fs:            p.fs,
	}
	if p.AllowsLanguageExperiments() {
		cfg.matchers = append(cfg.matchers, &queryFiles{})
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Scan and categorize main directory files
	mainDirDiags := p.rootFiles(dir, cfg.matchers, &fileSet)
	diags = append(diags, mainDirDiags...)
	if diags.HasErrors() {
		return fileSet, diags
	}

	// Process matcher-specific files
	for _, matcher := range cfg.matchers {
		matcherDiags := matcher.DirFiles(dir, cfg, &fileSet)
		diags = append(diags, matcherDiags...)
	}

	return fileSet, diags
}

// rootFiles scans the main directory for configuration files
// and categorizes them using the appropriate file matchers.
func (p *Parser) rootFiles(dir string, matchers []FileMatcher, fileSet *ConfigFileSet) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// Read main directory files
	infos, err := p.fs.ReadDir(dir)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to read module directory",
			Detail:   fmt.Sprintf("Module directory %s does not exist or cannot be read.", dir),
		})
		return diags
	}

	for _, info := range infos {
		if info.IsDir() || IsIgnoredFile(info.Name()) {
			continue
		}

		name := info.Name()
		fullPath := filepath.Join(dir, name)

		// Try each matcher to see if it matches
		for _, matcher := range matchers {
			if matcher.Matches(name) {
				switch p := matcher.(type) {
				case *moduleFiles:
					if p.isOverride(name) {
						fileSet.Override = append(fileSet.Override, fullPath)
					} else {
						fileSet.Primary = append(fileSet.Primary, fullPath)
					}
				case *testFiles:
					fileSet.Tests = append(fileSet.Tests, fullPath)
				case *queryFiles:
					fileSet.Queries = append(fileSet.Queries, fullPath)
				}
				break // Stop checking other matchers once a match is found
			}
		}
	}

	return diags
}

// MatchTestFiles adds a matcher for Terraform test files (.tftest.hcl and .tftest.json)
func MatchTestFiles(dir string) Option {
	return func(o *parserConfig) {
		o.testDirectory = dir
		o.matchers = append(o.matchers, &testFiles{})
	}
}

// moduleFiles matches regular Terraform configuration files (.tf and .tf.json)
type moduleFiles struct{}

func (m *moduleFiles) Matches(name string) bool {
	ext := fileExt(name)
	if ext != ".tf" && ext != ".tf.json" {
		return false
	}

	return true
}

func (m *moduleFiles) isOverride(name string) bool {
	ext := fileExt(name)
	if ext != ".tf" && ext != ".tf.json" {
		return false
	}

	baseName := name[:len(name)-len(ext)] // strip extension
	isOverride := baseName == "override" || strings.HasSuffix(baseName, "_override")
	return isOverride
}

func (m *moduleFiles) DirFiles(dir string, options *parserConfig, fileSet *ConfigFileSet) hcl.Diagnostics {
	return nil
}

// testFiles matches Terraform test files (.tftest.hcl and .tftest.json)
type testFiles struct{}

func (t *testFiles) Matches(name string) bool {
	return strings.HasSuffix(name, ".tftest.hcl") || strings.HasSuffix(name, ".tftest.json")
}

func (t *testFiles) DirFiles(dir string, opts *parserConfig, fileSet *ConfigFileSet) hcl.Diagnostics {
	var diags hcl.Diagnostics

	testPath := path.Join(dir, opts.testDirectory)
	testInfos, err := opts.fs.ReadDir(testPath)

	if err != nil {
		// Then we couldn't read from the testing directory for some reason.
		if os.IsNotExist(err) {
			// Then this means the testing directory did not exist.
			// We won't actually stop loading the rest of the configuration
			// for this, we will add a warning to explain to the user why
			// test files weren't processed but leave it at that.
			if opts.testDirectory != DefaultTestDirectory {
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
			return diags
		}
		return diags
	}

	// Process test files
	for _, info := range testInfos {
		if !t.Matches(info.Name()) {
			continue
		}

		name := info.Name()
		fileSet.Tests = append(fileSet.Tests, filepath.Join(testPath, name))
	}

	return diags
}

// queryFiles matches Terraform query files (.tfquery.hcl and .tfquery.json)
type queryFiles struct{}

func (q *queryFiles) Matches(name string) bool {
	return strings.HasSuffix(name, ".tfquery.hcl") || strings.HasSuffix(name, ".tfquery.json")
}

func (q *queryFiles) DirFiles(dir string, options *parserConfig, fileSet *ConfigFileSet) hcl.Diagnostics {
	return nil
}
