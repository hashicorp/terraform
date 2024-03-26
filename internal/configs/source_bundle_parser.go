// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// SourceBundleParser is the main interface to read configuration files and
// other related files from a source bundle. This is a subset of the
// functionality implemented by [Parser], specifically ignoring tftest files,
// which are not relevant for now.
type SourceBundleParser struct {
	sources *sourcebundle.Bundle
	p       *hclparse.Parser

	// allowExperiments controls whether we will allow modules to opt in to
	// experimental language features. In main code this will be set only
	// for alpha releases and some development builds. Test code must decide
	// for itself whether to enable it so that tests can cover both the
	// allowed and not-allowed situations.
	allowExperiments bool
}

// NewSourceBundleParser creates a new [SourceBundleParser] for the given
// source bundle.
func NewSourceBundleParser(sources *sourcebundle.Bundle) *SourceBundleParser {
	return &SourceBundleParser{
		sources: sources,
		p:       hclparse.NewParser(),
	}
}

// LoadConfigDir is the primary public entry point for [SourceBundleParser],
// and is similar to [Parser.LoadConfigDir]. It reads the .tf and .tf.json
// files at the given source address as config files, and combines these into a
// single [Module].
func (p *SourceBundleParser) LoadConfigDir(source sourceaddrs.FinalSource) (*Module, hcl.Diagnostics) {
	primarySources, overrideSources, diags := p.dirSources(source)
	if diags.HasErrors() {
		return nil, diags
	}

	primary, fDiags := p.loadSources(primarySources, false)
	diags = append(diags, fDiags...)
	override, fDiags := p.loadSources(overrideSources, true)
	diags = append(diags, fDiags...)

	mod, modDiags := NewModule(primary, override)
	diags = append(diags, modDiags...)

	mod.SourceDir = source.String()

	return mod, diags
}

// IsConfigDir is used to detect directories which have no config files, so
// that we can return useful early diagnostics when a given root module source
// address points at a directory which is not Terraform module.
func (p *SourceBundleParser) IsConfigDir(source sourceaddrs.FinalSource) bool {
	primaryPaths, overridePaths, _ := p.dirSources(source)
	return (len(primaryPaths) + len(overridePaths)) > 0
}

func (p *SourceBundleParser) dirSources(source sourceaddrs.FinalSource) (primary, override []sourceaddrs.FinalSource, diags hcl.Diagnostics) {
	localDir, err := p.sources.LocalPathForSource(source)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot find configuration source code",
			Detail:   fmt.Sprintf("Failed to load %s from the pre-installed source packages: %s.", source, err),
		})
		return
	}

	allEntries, err := os.ReadDir(localDir)
	if err != nil {
		if os.IsNotExist(err) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing Terraform configuration",
				Detail:   fmt.Sprintf("There is no Terraform configuration directory at %s.", source),
			})
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Cannot read Terraform configuration",
				// In this case the error message from the Go standard library
				// is likely to disclose the real local directory name
				// from the source bundle, but that's okay because it may
				// sometimes help with debugging.
				Detail: fmt.Sprintf("Error while reading the cached snapshot of %s: %s.", source, err),
			})
		}
		return
	}

	for _, entry := range allEntries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := fileExt(name)
		if ext == "" || IsIgnoredFile(name) {
			continue
		}

		if ext == ".tftest.hcl" || ext == ".tftest.json" {
			continue
		}

		baseName := name[:len(name)-len(ext)] // strip extension
		isOverride := baseName == "override" || strings.HasSuffix(baseName, "_override")

		asLocalSourcePath := "./" + filepath.Base(name)
		relSource, err := sourceaddrs.ParseLocalSource(asLocalSourcePath)
		if err != nil {
			// If we get here then it's a bug in how we constructed the
			// path above, not invalid user input.
			panic(fmt.Sprintf("constructed invalid relative source path: %s", err))
		}
		fileSourceAddr, err := sourceaddrs.ResolveRelativeFinalSource(source, relSource)
		if err != nil {
			// If we get here then it's a bug in how we constructed the
			// path above, not invalid user input.
			panic(fmt.Sprintf("constructed invalid relative source path: %s", err))
		}

		if isOverride {
			override = append(override, fileSourceAddr)
		} else {
			primary = append(primary, fileSourceAddr)
		}
	}

	return
}

func (p *SourceBundleParser) loadSources(sources []sourceaddrs.FinalSource, override bool) ([]*File, hcl.Diagnostics) {
	var files []*File
	var diags hcl.Diagnostics

	for _, path := range sources {
		f, fDiags := p.loadConfigFile(path, override)
		diags = append(diags, fDiags...)
		if f != nil {
			files = append(files, f)
		}
	}

	return files, diags
}

func (p *SourceBundleParser) loadConfigFile(source sourceaddrs.FinalSource, override bool) (*File, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	path, err := p.sources.LocalPathForSource(source)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot find configuration source code",
			Detail:   fmt.Sprintf("Failed to load %s from the pre-installed source packages: %s.", source, err),
		})
		return nil, diags
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Failed to read file",
				Detail:   fmt.Sprintf("The file %q could not be read.", path),
			},
		}
	}

	// NOTE: this synthetic filename is intentionally a string rendering of the
	// file's source address, which in many cases is _not_ a path name. We use
	// the full source address in order to allow later consumers of diagnostics
	// to look up the configuration file from the source bundle. We use this in
	// the filename field of the diagnostic source to achieve this.
	syntheticFilename := source.String()

	var file *hcl.File
	var fdiags hcl.Diagnostics
	switch {
	case strings.HasSuffix(path, ".json"):
		file, fdiags = p.p.ParseJSON(src, syntheticFilename)
	default:
		file, fdiags = p.p.ParseHCL(src, syntheticFilename)
	}
	diags = append(diags, fdiags...)

	body := hcl.EmptyBody()
	if file != nil && file.Body != nil {
		body = file.Body
	}

	return parseConfigFile(body, diags, override, p.allowExperiments)
}

// AllowLanguageExperiments specifies whether subsequent LoadConfigFile (and
// similar) calls will allow opting in to experimental language features.
//
// If this method is never called for a particular parser, the default behavior
// is to disallow language experiments.
//
// Main code should set this only for alpha or development builds. Test code
// is responsible for deciding for itself whether and how to call this
// method.
func (p *SourceBundleParser) AllowLanguageExperiments(allowed bool) {
	p.allowExperiments = allowed
}
