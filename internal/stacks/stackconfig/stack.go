// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Stack represents a single stack, which can potentially call other
// "embedded stacks" in a similar manner to how Terraform modules can call
// other modules.
type Stack struct {
	SourceAddr sourceaddrs.FinalSource

	// ConfigFiles describes the individual .tfstack.hcl or .tfstack.json
	// files that this stack configuration object was built from. Most callers
	// should ignore the detail of which file each declaration originated
	// in, but we retain this in case it's useful for generating better error
	// messages, etc.
	//
	// The keys of this map are the string representations of each file's
	// source address, which also matches how we populate the "Filename"
	// field of source ranges referring to the files and so callers can
	// attempt to look up files by the diagnostic range filename, but must
	// be resilient to cases where nothing matches because not all diagnostics
	// will refer to stack configuration files.
	ConfigFiles map[string]*File

	Declarations
}

// LoadSingleStackConfig loads the configuration for only a single stack from
// the given source address.
//
// If the given address is a local source then it's interpreted relative to
// the process's current working directory. Otherwise it will be loaded from
// the provided source bundle.
//
// This is exported for unusual situations where it's useful to analyze just
// a single stack configuration directory in isolation, without considering
// its context in a configuration tree. Some fields of the objects representing
// declarations in the configuration will be unpopulated when loading through
// this entry point. Prefer [LoadConfigDir] in most cases.
func LoadSingleStackConfig(sourceAddr sourceaddrs.FinalSource, sources *sourcebundle.Bundle) (*Stack, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	localDir, err := sources.LocalPathForSource(sourceAddr)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Cannot find configuration source code",
			fmt.Sprintf("Failed to load %s from the pre-installed source packages: %s.", sourceAddr, err),
		))
		return nil, diags
	}

	allEntries, err := os.ReadDir(localDir)
	if err != nil {
		if os.IsNotExist(err) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Missing stack configuration",
				fmt.Sprintf("There is no stack configuration directory at %s.", sourceAddr),
			))
		} else {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Cannot read stack configuration",
				// In this case the error message from the Go standard library
				// is likely to disclose the real local directory name
				// from the source bundle, but that's okay because it may
				// sometimes help with debugging.
				fmt.Sprintf("Error while reading the cached snapshot of %s: %s.", sourceAddr, err),
			))
		}
		return nil, diags
	}

	ret := &Stack{
		SourceAddr:   sourceAddr,
		ConfigFiles:  make(map[string]*File),
		Declarations: makeDeclarations(),
	}

	for _, entry := range allEntries {
		if suffix := validFilenameSuffix(entry.Name()); suffix == "" {
			// not a file we're interested in, then
			continue
		}

		asLocalSourcePath := "./" + filepath.Base(entry.Name())
		relSource, err := sourceaddrs.ParseLocalSource(asLocalSourcePath)
		if err != nil {
			// If we get here then it's a bug in how we constructed the
			// path above, not invalid user input.
			panic(fmt.Sprintf("constructed invalid relative source path: %s", err))
		}
		fileSourceAddr, err := sourceaddrs.ResolveRelativeFinalSource(sourceAddr, relSource)
		if err != nil {
			// If we get here then it's a bug in how we constructed the
			// path above, not invalid user input.
			panic(fmt.Sprintf("constructed invalid relative source path: %s", err))
		}
		if entry.IsDir() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid stack configuration directory",
				fmt.Sprintf("The entry %s is a directory. All entries with the stack configuration name suffixes must be files.", fileSourceAddr),
			))
		}

		src, err := os.ReadFile(filepath.Join(localDir, entry.Name()))
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Cannot read stack configuration",
				// In this case the error message from the Go standard library
				// is likely to disclose the real local directory name
				// from the source bundle, but that's okay because it may
				// sometimes help with debugging.
				fmt.Sprintf("Error while reading the cached snapshot of %s: %s.", fileSourceAddr, err),
			))
		}

		file, moreDiags := ParseFileSource(src, fileSourceAddr)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			// We'll still try to analyze other files, so we can gather up
			// as many diagnostics as possible to return all together in
			// case there's some pattern between them that the user can
			// fix systematically across all instances.
			continue
		}

		// Incorporate this file's declarations into the overall stack
		// configuration.
		diags = diags.Append(ret.Declarations.merge(&file.Declarations))
		ret.ConfigFiles[file.SourceAddr.String()] = file
	}

	for _, pc := range ret.ProviderConfigs {
		localName := pc.LocalAddr.LocalName
		providerAddr, ok := ret.RequiredProviders.ProviderForLocalName(localName)
		if !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Undeclared provider local name",
				Detail: fmt.Sprintf(
					"This configuration's required_providers block does not include a definition for the local name %q.",
					localName,
				),
			})
			continue
		}
		pc.ProviderAddr = providerAddr
	}

	return ret, diags
}
