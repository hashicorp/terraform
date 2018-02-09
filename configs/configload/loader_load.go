package configload

import (
	"fmt"
	"log"
	"path/filepath"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/registry/regsrc"
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
	addr, err := regsrc.ParseModuleSource(req.SourceAddr)
	switch err {
	case nil:
		return l.loadRegistryModule(req, addr)
	case regsrc.ErrInvalidModuleSource:
		return l.loadNonRegistryModule(req)
	default:
		log.Printf("[ERROR] Error parsing %q as a module source string: %s", req.SourceAddr, err)
		return nil, nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Invalid module source address",
				Detail:   fmt.Sprintf("Failed to parse source address: %s.", err),
				Subject:  &req.SourceAddrRange,
			},
		}
	}
}

func (l *Loader) loadRegistryModule(req *configs.ModuleRequest, addr *regsrc.Module) (*configs.Module, *version.Version, hcl.Diagnostics) {
	records, err := l.modules.loadManifestRecords()
	if err != nil {
		// Should never happen unless something or someone has tampered with
		// the manifest file.
		return nil, nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Error loading module manifest file",
				Detail:   fmt.Sprintf("The module manifest file %s could not be loaded: %s.", manifestName, err),
				Subject:  &req.CallRange,
			},
		}
	}

	records = records.VersionsForAddr(addr.String())
	if len(records) == 0 {
		return nil, nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Module not installed",
				Detail: fmt.Sprintf(
					"There are no versions of module %q currently installed. Run \"terraform init\" to install all modules needed by this configuration.",
					addr.String(),
				),
				Subject: &req.CallRange,
			},
		}
	}

	latest := records.Newest(req.VersionConstraint.Required)
	fullPath := filepath.Join(latest.Dir, latest.Root)

	mod, diags := l.parser.LoadConfigDir(fullPath)
	if mod == nil {
		// nil means that the directory doesn't exist at all or could not be
		// read, so in that case we'll discard the generic diagnostics returned
		// by LoadConfigDir and generate our own context-sensitive message.
		return nil, nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Module not installed",
				Detail: fmt.Sprintf(
					"A locally-installed copy of module %q %s was not found at %s. Run \"terraform init\" to install all modules needed by this configuration.",
					addr.String(), latest.Version, fullPath,
				),
				Subject: &req.CallRange,
			},
		}
	}

	return mod, latest.Version, diags
}

func (l *Loader) loadNonRegistryModule(req *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {
	records, err := l.modules.loadManifestRecords()
	if err != nil {
		// Should never happen unless something or someone has tampered with
		// the manifest file.
		return nil, nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Error loading module manifest file",
				Detail:   fmt.Sprintf("The module manifest file %s could not be loaded: %s.", manifestName, err),
				Subject:  &req.CallRange,
			},
		}
	}

	for _, record := range records {
		if record.SourceAddr != req.SourceAddr {
			continue
		}
		fullPath := filepath.Join(record.Dir, record.Root)
		mod, diags := l.parser.LoadConfigDir(fullPath)
		if mod == nil {
			// nil means that the directory doesn't exist at all or could not be
			// read, so in that case we'll discard the generic diagnostics returned
			// by LoadConfigDir and generate our own context-sensitive message.
			return nil, nil, hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Module not installed",
					Detail: fmt.Sprintf(
						"A locally-installed copy of module %q was not found at %s. Run \"terraform init\" to install all modules needed by this configuration.",
						req.SourceAddr, fullPath,
					),
					Subject: &req.CallRange,
				},
			}
		}

		return mod, nil, diags
	}

	// If we fall out here then we don't have any record of the requested
	// module in our manifest.
	return nil, nil, hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
			Summary:  "Module not installed",
			Detail:   "Module %q is not currently installed. Run \"terraform init\" to install all modules needed by this configuration.",
			Subject:  &req.CallRange,
		},
	}
}
