package configload

import (
	"fmt"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
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
// required to process the individual modules
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

	// Check for inconsistencies between manifest and config
	if req.SourceAddr != record.SourceAddr {
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

	// The providers associated with expanding modules must be present in the proxy/passed providers
	// block. Guarding here for accessing the module call just in case.
	if mc, exists := req.Parent.Module.ModuleCalls[req.Name]; exists {
		var validateDiags hcl.Diagnostics
		validateDiags = validateProviderConfigs(mc, mod, req.Parent, validateDiags)
		diags = append(diags, validateDiags...)
	}
	return mod, record.Version, diags
}

func validateProviderConfigs(mc *configs.ModuleCall, mod *configs.Module, parent *configs.Config, diags hcl.Diagnostics) hcl.Diagnostics {
	if mc.Count != nil || mc.ForEach != nil || mc.DependsOn != nil {
		for key, pc := range mod.ProviderConfigs {
			// Use these to track if a provider is configured (not allowed),
			// or if we've found its matching proxy
			var isConfigured bool
			var foundMatchingProxy bool

			// Validate the config against an empty schema to see if it's empty.
			_, pcConfigDiags := pc.Config.Content(&hcl.BodySchema{})
			if pcConfigDiags.HasErrors() || pc.Version.Required != nil {
				isConfigured = true
			}

			// If it is empty or only has an alias,
			// does this provider exist in our proxy configs?
			for _, r := range mc.Providers {
				// Must match on name and Alias
				if pc.Name == r.InChild.Name && pc.Alias == r.InChild.Alias {
					foundMatchingProxy = true
					break
				}
			}
			if isConfigured || !foundMatchingProxy {
				if mc.Count != nil {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Module does not support count",
						Detail:   fmt.Sprintf(moduleProviderError, mc.Name, "count", key, pc.NameRange),
						Subject:  mc.Count.Range().Ptr(),
					})
				}
				if mc.ForEach != nil {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Module does not support for_each",
						Detail:   fmt.Sprintf(moduleProviderError, mc.Name, "for_each", key, pc.NameRange),
						Subject:  mc.ForEach.Range().Ptr(),
					})
				}
				if mc.DependsOn != nil {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Module does not support depends_on",
						Detail:   fmt.Sprintf(moduleProviderError, mc.Name, "depends_on", key, pc.NameRange),
						Subject:  mc.SourceAddrRange.Ptr(),
					})
				}
			}
		}
	}
	// If this module has further parents, go through them recursively
	if !parent.Path.IsRoot() {
		// Use the path to get the name so we can look it up in the parent module calls
		path := parent.Path
		name := path[len(path)-1]
		// This parent's module call, so we can check for count/for_each here,
		// guarding with exists just in case. We pass the diags through to the recursive
		// call so they will accumulate if needed.
		if mc, exists := parent.Parent.Module.ModuleCalls[name]; exists {
			return validateProviderConfigs(mc, mod, parent.Parent, diags)
		}
	}

	return diags
}

var moduleProviderError = `Module "%s" cannot be used with %s because it contains a nested provider configuration for "%s", at %s.

This module can be made compatible with %[2]s by changing it to receive all of its provider configurations from the calling module, by using the "providers" argument in the calling module block.`
