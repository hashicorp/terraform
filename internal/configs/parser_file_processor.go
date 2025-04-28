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
)

// FileProcessor is an interface for components that can match and process specific file types
type FileProcessor interface {
	// Matches returns true if the given filename should be processed by this processor
	Matches(name string) bool

	// DirFiles finds and processes terraform configuration files in the given directory
	DirFiles(p *Parser, dir string, opts *ProcessorOptions) ([]string, hcl.Diagnostics)
}

// ProcessorOptions contains configuration options for file processors
type ProcessorOptions struct {
	TestDirectory string
}

// Option is a functional option type for configuring the parser
type Option func(*Options)

type Options struct {
	Processors    []FileProcessor
	TestDirectory string
}

// ConfigFileSet holds the different types of configuration files found in a directory.
type ConfigFileSet struct {
	Primary  []string // Regular .tf and .tf.json files
	Override []string // Override files (override.tf or *_override.tf)
	Tests    []string // Test files (.tftest.hcl or .tftest.json)
	Queries  []string // Query files (.tfquery.hcl)
}

// LoadConfigDir reads the configuration files in the given directory
// as config files (using LoadConfigFile) and then combines these files into
// a single Module.
//
// Main terraform configuration files (.tf and .tf.json) are loaded as the primary
// module, while override files (override.tf and *_override.tf) are loaded as
// overrides.
// Optionally, test files (.tftest.hcl and .tftest.json) can be loaded from
// a subdirectory of the given directory, which is specified by the
// TestDirectory option. If this option is not specified, test files will
// not be loaded.
// Query files (.tfquery.hcl) are also loaded from the given directory if
// specified by the WithQueryFiles option.
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

	// Load the actual files
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

	// Initialize options with defaults
	options := &Options{
		// We always process module and override files
		Processors: []FileProcessor{
			&moduleFiles{},
			&overrideFiles{},
		},
		TestDirectory: DefaultTestDirectory,
	}

	// Apply the provided options
	for _, opt := range opts {
		opt(options)
	}

	pOpts := &ProcessorOptions{TestDirectory: options.TestDirectory}

	for _, processor := range options.Processors {
		files, procDiags := processor.DirFiles(p, dir, pOpts)
		diags = append(diags, procDiags...)

		// Determine where to store the files based on processor type
		switch processor.(type) {
		case *moduleFiles:
			fileSet.Primary = append(fileSet.Primary, files...)
		case *overrideFiles:
			fileSet.Override = append(fileSet.Override, files...)
		case *testFiles:
			fileSet.Tests = append(fileSet.Tests, files...)
		case *queryFiles:
			fileSet.Queries = append(fileSet.Queries, files...)
		}
	}

	return fileSet, diags

}

// WithProcessor adds a file processor to the parser
func WithProcessor(processor FileProcessor) Option {
	return func(o *Options) {
		o.Processors = append(o.Processors, processor)
	}
}

// WithModuleFiles adds a processor for standard Terraform configuration files (.tf and .tf.json)
func WithModuleFiles() Option {
	return WithProcessor(&moduleFiles{})
}

// WithOverrideFiles adds a processor for override Terraform configuration files
func WithOverrideFiles() Option {
	return WithProcessor(&overrideFiles{})
}

// WithTestFiles adds a processor for Terraform test files (.tftest.hcl and .tftest.json)
func WithTestFiles(dir string) Option {
	return func(o *Options) {
		o.TestDirectory = dir
		WithProcessor(&testFiles{})(o)
	}
}

// WithQueryFiles adds a processor for Terraform query files (.tfquery.hcl)
func WithQueryFiles() Option {
	return WithProcessor(&queryFiles{})
}

// moduleFiles processes regular Terraform configuration files (.tf and .tf.json)
type moduleFiles struct{}

func (m *moduleFiles) Matches(name string) bool {
	return strings.HasSuffix(name, ".tf") || strings.HasSuffix(name, ".tf.json")
}

func (m *moduleFiles) DirFiles(p *Parser, dir string, opts *ProcessorOptions) ([]string, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	primary := []string{}

	infos, err := p.fs.ReadDir(dir)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to read module directory",
			Detail:   fmt.Sprintf("Module directory %s does not exist or cannot be read.", dir),
		})
		return nil, diags
	}

	for _, info := range infos {
		if info.IsDir() || IsIgnoredFile(info.Name()) {
			continue
		}

		name := info.Name()
		ext := fileExt(name)
		if ext != ".tf" && ext != ".tf.json" {
			continue
		}

		baseName := name[:len(name)-len(ext)] // strip extension
		isOverride := baseName == "override" || strings.HasSuffix(baseName, "_override")

		if !isOverride {
			primary = append(primary, filepath.Join(dir, name))
		}
	}

	return primary, diags
}

// overrideFiles processes override Terraform configuration files
type overrideFiles struct{}

func (o *overrideFiles) Matches(name string) bool {
	ext := fileExt(name)
	if ext != ".tf" && ext != ".tf.json" {
		return false
	}

	baseName := name[:len(name)-len(ext)] // strip extension
	return baseName == "override" || strings.HasSuffix(baseName, "_override")
}

func (o *overrideFiles) DirFiles(p *Parser, dir string, opts *ProcessorOptions) ([]string, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	override := []string{}

	infos, err := p.fs.ReadDir(dir)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to read module directory",
			Detail:   fmt.Sprintf("Module directory %s does not exist or cannot be read.", dir),
		})
		return nil, diags
	}

	for _, info := range infos {
		if info.IsDir() || IsIgnoredFile(info.Name()) {
			continue
		}

		name := info.Name()
		ext := fileExt(name)
		if ext != ".tf" && ext != ".tf.json" {
			continue
		}

		baseName := name[:len(name)-len(ext)] // strip extension
		isOverride := baseName == "override" || strings.HasSuffix(baseName, "_override")

		if isOverride {
			override = append(override, filepath.Join(dir, name))
		}
	}

	return override, diags
}

// testFiles processes Terraform test files (.tftest.hcl and .tftest.json)
type testFiles struct{}

func (t *testFiles) Matches(name string) bool {
	return strings.HasSuffix(name, ".tftest.hcl") || strings.HasSuffix(name, ".tftest.json")
}

func (t *testFiles) DirFiles(p *Parser, dir string, opts *ProcessorOptions) ([]string, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	tests := []string{}

	// Skip if no test directory is specified
	if opts.TestDirectory == "" {
		return tests, diags
	}

	// First check in the main directory
	infos, err := p.fs.ReadDir(dir)
	if err == nil {
		for _, info := range infos {
			if info.IsDir() || IsIgnoredFile(info.Name()) {
				continue
			}

			name := info.Name()
			if t.Matches(name) {
				tests = append(tests, filepath.Join(dir, name))
			}
		}
	}

	// Then check in the test directory
	testPath := path.Join(dir, opts.TestDirectory)
	infos, err = p.fs.ReadDir(testPath)
	if err != nil {
		// Then we couldn't read from the testing directory for some reason.

		if os.IsNotExist(err) {
			// Then this means the testing directory did not exist.
			// We won't actually stop loading the rest of the configuration
			// for this, we will add a warning to explain to the user why
			// test files weren't processed but leave it at that.
			if opts.TestDirectory != DefaultTestDirectory {
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
			return tests, diags
		}
	}

	for _, info := range infos {
		if info.IsDir() || IsIgnoredFile(info.Name()) {
			continue
		}

		name := info.Name()
		if t.Matches(name) {
			tests = append(tests, filepath.Join(testPath, name))
		}
	}

	return tests, diags
}

// queryFiles processes Terraform query files (.tfquery.hcl)
type queryFiles struct{}

func (q *queryFiles) Matches(name string) bool {
	return strings.HasSuffix(name, ".tfquery.hcl")
}

func (q *queryFiles) DirFiles(p *Parser, dir string, opts *ProcessorOptions) ([]string, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	queries := []string{}

	infos, err := p.fs.ReadDir(dir)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to read module directory",
			Detail:   fmt.Sprintf("Module directory %s does not exist or cannot be read.", dir),
		})
		return nil, diags
	}

	for _, info := range infos {
		if info.IsDir() || IsIgnoredFile(info.Name()) {
			continue
		}

		name := info.Name()
		if q.Matches(name) {
			queries = append(queries, filepath.Join(dir, name))
		}
	}

	return queries, diags
}
