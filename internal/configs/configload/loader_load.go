// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configload

import (
	"fmt"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/configs"
)

// LoadConfig reads the Terraform module in the given directory and uses it as the
// root module to build the static module tree that represents a configuration,
// assuming that all required descendant modules have already been installed.
//
// If error diagnostics are returned, the returned configuration may be either
// nil or incomplete. In the latter case, cautious static analysis is possible
// in spite of the errors.
//
// LoadConfig performs the basic syntax and uniqueness validations that are
// required to process the individual modules
func (l *Loader) LoadConfig(rootDir string) (*configs.Config, hcl.Diagnostics) {
	return l.loadConfig(l.parser.LoadConfigDir(rootDir))
}

// LoadConfigWithTests matches LoadConfig, except the configs.Config contains
// any relevant .tftest.hcl files.
func (l *Loader) LoadConfigWithTests(rootDir string, testDir string) (*configs.Config, hcl.Diagnostics) {
	return l.loadConfig(l.parser.LoadConfigDirWithTests(rootDir, testDir))
}

func (l *Loader) loadConfig(rootMod *configs.Module, diags hcl.Diagnostics) (*configs.Config, hcl.Diagnostics) {
	if rootMod == nil || diags.HasErrors() {
		// Ensure we return any parsed modules here so that required_version
		// constraints can be verified even when encountering errors.
		cfg := &configs.Config{
			Module: rootMod,
		}

		return cfg, diags
	}

	cfg, cDiags := configs.BuildConfig(rootMod, configs.ModuleWalkerFunc(l.moduleWalkerLoad), configs.MockDataLoaderFunc(l.LoadExternalMockData))
	diags = append(diags, cDiags...)

	return cfg, diags
}

// LoadExternalMockData reads the external mock data files for the given
// provider, if they are present.
func (l *Loader) LoadExternalMockData(provider *configs.Provider) (*configs.MockData, hcl.Diagnostics) {
	if len(provider.MockDataExternalSource) == 0 {
		// We have no external mock data, so don't do anything.
		return nil, nil
	}

	// Otherwise, just hand this off to the parser to handle.
	return l.parser.LoadMockDataDir(provider.MockDataExternalSource, provider.MockDataDuringPlan, provider.DeclRange)
}

// moduleWalkerLoad is a configs.ModuleWalkerFunc for loading modules that
// are presumed to have already been installed.
func (l *Loader) moduleWalkerLoad(req *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {
	// Since we're just loading here, we expect that all referenced modules
	// will be already installed and described in our manifest. However, we
	// do verify that the manifest and the configuration are in agreement
	// so that we can prompt the user to run "terraform init" if not.

	key := l.modules.manifest.ModuleKey(req.Path)
	record, exists := l.modules.manifest[key]

	if !exists {
		return nil, nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Module not installed",
				Detail:   "This module is not yet installed. Run \"terraform init\" to install all modules required by this configuration.",
				Subject:  &req.CallRange,
			},
		}
	}

	var diags hcl.Diagnostics

	// Check for inconsistencies between manifest and config.

	// We ignore a nil SourceAddr here, which represents a failure during
	// configuration parsing, and will be reported in a diagnostic elsewhere.
	if req.SourceAddr != nil && req.SourceAddr.String() != record.SourceAddr {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Module source has changed",
			Detail:   "The source address was changed since this module was installed. Run \"terraform init\" to install all modules required by this configuration.",
			Subject:  &req.SourceAddrRange,
		})
	}
	if len(req.VersionConstraint.Required) > 0 && record.Version == nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Module version requirements have changed",
			Detail:   "The version requirements have changed since this module was installed and the installed version is no longer acceptable. Run \"terraform init\" to install all modules required by this configuration.",
			Subject:  &req.SourceAddrRange,
		})
	}
	if record.Version != nil && !req.VersionConstraint.Required.Check(record.Version) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Module version requirements have changed",
			Detail: fmt.Sprintf(
				"The version requirements have changed since this module was installed and the installed version (%s) is no longer acceptable. Run \"terraform init\" to install all modules required by this configuration.",
				record.Version,
			),
			Subject: &req.SourceAddrRange,
		})
	}

	mod, mDiags := l.parser.LoadConfigDir(record.Dir)
	diags = append(diags, mDiags...)
	if mod == nil {
		// nil specifically indicates that the directory does not exist or
		// cannot be read, so in this case we'll discard any generic diagnostics
		// returned from LoadConfigDir and produce our own context-sensitive
		// error message.
		return nil, nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Module not installed",
				Detail:   fmt.Sprintf("This module's local cache directory %s could not be read. Run \"terraform init\" to install all modules required by this configuration.", record.Dir),
				Subject:  &req.CallRange,
			},
		}
	}

	return mod, record.Version, diags
}
