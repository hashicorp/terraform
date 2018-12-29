package configload

import (
	"fmt"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/configs"
)

// LoadConfig reads the Terraform module in the given directory and uses it as the
// root module to build the static module tree that represents a configuration,
// assuming that all required descendent modules have already been installed.
//
// If error diagnostics are returned, the returned configuration may be either
// nil or incomplete. In the latter case, cautious static analysis is possible
// in spite of the errors.
//
// LoadConfig performs the basic syntax and uniqueness validations that are
// required to process the individual modules, and also detects
func (l *Loader) LoadConfig(rootDir string) (*configs.Config, hcl.Diagnostics) {
	rootMod, diags := l.parser.LoadConfigDir(rootDir)
	if rootMod == nil {
		return nil, diags
	}

	cfg, cDiags := configs.BuildConfig(rootMod, configs.ModuleWalkerFunc(l.moduleWalkerLoad))
	diags = append(diags, cDiags...)

	return cfg, diags
}

// moduleWalkerLoad is a configs.ModuleWalkerFunc for loading modules that
// are presumed to have already been installed. A different function
// (moduleWalkerInstall) is used for installation.
func (l *Loader) moduleWalkerLoad(req *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {
	// Since we're just loading here, we expect that all referenced modules
	// will be already installed and described in our manifest. However, we
	// do verify that the manifest and the configuration are in agreement
	// so that we can prompt the user to run "terraform init" if not.

	key := manifestKey(req.Path)
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

	// Check for inconsistencies between manifest and config
	if req.SourceAddr != record.SourceAddr {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Module source has changed",
			Detail:   "The source address was changed since this module was installed. Run \"terraform init\" to install all modules required by this configuration.",
			Subject:  &req.SourceAddrRange,
		})
	}
	if !req.VersionConstraint.Required.Check(record.Version) {
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
