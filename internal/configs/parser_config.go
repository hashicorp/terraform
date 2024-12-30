// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"github.com/hashicorp/hcl/v2"
)

// LoadConfigFile reads the file at the given path and parses it as a config
// file.
//
// If the file cannot be read -- for example, if it does not exist -- then
// a nil *File will be returned along with error diagnostics. Callers may wish
// to disregard the returned diagnostics in this case and instead generate
// their own error message(s) with additional context.
//
// If the returned diagnostics has errors when a non-nil map is returned
// then the map may be incomplete but should be valid enough for careful
// static analysis.
//
// This method wraps LoadHCLFile, and so it inherits the syntax selection
// behaviors documented for that method.
func (p *Parser) LoadConfigFile(path string) (*File, hcl.Diagnostics) {
	return p.loadConfigFile(path, false)
}

// LoadConfigFileOverride is the same as LoadConfigFile except that it relaxes
// certain required attribute constraints in order to interpret the given
// file as an overrides file.
func (p *Parser) LoadConfigFileOverride(path string) (*File, hcl.Diagnostics) {
	return p.loadConfigFile(path, true)
}

// LoadTestFile reads the file at the given path and parses it as a Terraform
// test file.
//
// It references the same LoadHCLFile as LoadConfigFile, so inherits the same
// syntax selection behaviours.
func (p *Parser) LoadTestFile(path string) (*TestFile, hcl.Diagnostics) {
	body, diags := p.LoadHCLFile(path)
	if body == nil {
		return nil, diags
	}

	test, testDiags := loadTestFile(body)
	diags = append(diags, testDiags...)
	return test, diags
}

// LoadMockDataFile reads the file at the given path and parses it as a
// Terraform mock data file.
//
// It references the same LoadHCLFile as LoadConfigFile, so inherits the same
// syntax selection behaviours.
func (p *Parser) LoadMockDataFile(path string) (*MockData, hcl.Diagnostics) {
	body, diags := p.LoadHCLFile(path)
	if body == nil {
		return nil, diags
	}

	data, dataDiags := decodeMockDataBody(body, MockDataFileOverrideSource)
	diags = append(diags, dataDiags...)
	return data, diags
}

func (p *Parser) loadConfigFile(path string, override bool) (*File, hcl.Diagnostics) {
	body, diags := p.LoadHCLFile(path)
	if body == nil {
		return nil, diags
	}

	return parseConfigFile(body, diags, override, p.allowExperiments)
}

func parseConfigFile(body hcl.Body, diags hcl.Diagnostics, override, allowExperiments bool) (*File, hcl.Diagnostics) {
	file := &File{}

	var reqDiags hcl.Diagnostics
	file.CoreVersionConstraints, reqDiags = sniffCoreVersionRequirements(body)
	diags = append(diags, reqDiags...)

	// We'll load the experiments first because other decoding logic in the
	// loop below might depend on these experiments.
	var expDiags hcl.Diagnostics
	file.ActiveExperiments, expDiags = sniffActiveExperiments(body, allowExperiments)
	diags = append(diags, expDiags...)

	content, contentDiags := body.Content(configFileSchema)
	diags = append(diags, contentDiags...)

	for _, block := range content.Blocks {
		switch block.Type {

		case "terraform":
			content, contentDiags := block.Body.Content(terraformBlockSchema)
			diags = append(diags, contentDiags...)

			// We ignore the "terraform_version", "language" and "experiments"
			// attributes here because sniffCoreVersionRequirements and
			// sniffActiveExperiments already dealt with those above.

			for _, innerBlock := range content.Blocks {
				switch innerBlock.Type {

				case "backend":
					backendCfg, cfgDiags := decodeBackendBlock(innerBlock)
					diags = append(diags, cfgDiags...)
					if backendCfg != nil {
						file.Backends = append(file.Backends, backendCfg)
					}

				case "cloud":
					cloudCfg, cfgDiags := decodeCloudBlock(innerBlock)
					diags = append(diags, cfgDiags...)
					if cloudCfg != nil {
						file.CloudConfigs = append(file.CloudConfigs, cloudCfg)
					}

				case "required_providers":
					reqs, reqsDiags := decodeRequiredProvidersBlock(innerBlock)
					diags = append(diags, reqsDiags...)
					if reqs != nil {
						file.RequiredProviders = append(file.RequiredProviders, reqs)
					}

				case "provider_meta":
					providerCfg, cfgDiags := decodeProviderMetaBlock(innerBlock)
					diags = append(diags, cfgDiags...)
					if providerCfg != nil {
						file.ProviderMetas = append(file.ProviderMetas, providerCfg)
					}

				default:
					// Should never happen because the above cases should be exhaustive
					// for all block type names in our schema.
					continue

				}
			}

		case "required_providers":
			// required_providers should be nested inside a "terraform" block
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid required_providers block",
				Detail:   "A \"required_providers\" block must be nested inside a \"terraform\" block.",
				Subject:  block.TypeRange.Ptr(),
			})

		case "provider":
			cfg, cfgDiags := decodeProviderBlock(block, false)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.ProviderConfigs = append(file.ProviderConfigs, cfg)
			}

		case "variable":
			cfg, cfgDiags := decodeVariableBlock(block, override)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.Variables = append(file.Variables, cfg)
			}

		case "locals":
			defs, defsDiags := decodeLocalsBlock(block)
			diags = append(diags, defsDiags...)
			file.Locals = append(file.Locals, defs...)

		case "output":
			cfg, cfgDiags := decodeOutputBlock(block, override)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.Outputs = append(file.Outputs, cfg)
			}

		case "module":
			cfg, cfgDiags := decodeModuleBlock(block, override)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.ModuleCalls = append(file.ModuleCalls, cfg)
			}

		case "resource":
			cfg, cfgDiags := decodeResourceBlock(block, override)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.ManagedResources = append(file.ManagedResources, cfg)
			}

		case "data":
			cfg, cfgDiags := decodeDataBlock(block, override, false)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.DataResources = append(file.DataResources, cfg)
			}

		case "ephemeral":
			cfg, cfgDiags := decodeEphemeralBlock(block, override)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.EphemeralResources = append(file.EphemeralResources, cfg)
			}

		case "moved":
			cfg, cfgDiags := decodeMovedBlock(block)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.Moved = append(file.Moved, cfg)
			}

		case "removed":
			cfg, cfgDiags := decodeRemovedBlock(block)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.Removed = append(file.Removed, cfg)
			}

		case "import":
			cfg, cfgDiags := decodeImportBlock(block)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.Import = append(file.Import, cfg)
			}

		case "check":
			cfg, cfgDiags := decodeCheckBlock(block, override)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.Checks = append(file.Checks, cfg)
			}

		default:
			// Should never happen because the above cases should be exhaustive
			// for all block type names in our schema.
			continue

		}
	}

	return file, diags
}

// sniffCoreVersionRequirements does minimal parsing of the given body for
// "terraform" blocks with "required_version" attributes, returning the
// requirements found.
//
// This is intended to maximize the chance that we'll be able to read the
// requirements (syntax errors notwithstanding) even if the config file contains
// constructs that might've been added in future Terraform versions
//
// This is a "best effort" sort of method which will return constraints it is
// able to find, but may return no constraints at all if the given body is
// so invalid that it cannot be decoded at all.
func sniffCoreVersionRequirements(body hcl.Body) ([]VersionConstraint, hcl.Diagnostics) {
	rootContent, _, diags := body.PartialContent(configFileTerraformBlockSniffRootSchema)

	var constraints []VersionConstraint

	for _, block := range rootContent.Blocks {
		content, _, blockDiags := block.Body.PartialContent(configFileVersionSniffBlockSchema)
		diags = append(diags, blockDiags...)

		attr, exists := content.Attributes["required_version"]
		if !exists {
			continue
		}

		constraint, constraintDiags := decodeVersionConstraint(attr)
		diags = append(diags, constraintDiags...)
		if !constraintDiags.HasErrors() {
			constraints = append(constraints, constraint)
		}
	}

	return constraints, diags
}

// configFileSchema is the schema for the top-level of a config file. We use
// the low-level HCL API for this level so we can easily deal with each
// block type separately with its own decoding logic.
var configFileSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "terraform",
		},
		{
			// This one is not really valid, but we include it here so we
			// can create a specialized error message hinting the user to
			// nest it inside a "terraform" block.
			Type: "required_providers",
		},
		{
			Type:       "provider",
			LabelNames: []string{"name"},
		},
		{
			Type:       "variable",
			LabelNames: []string{"name"},
		},
		{
			Type: "locals",
		},
		{
			Type:       "output",
			LabelNames: []string{"name"},
		},
		{
			Type:       "module",
			LabelNames: []string{"name"},
		},
		{
			Type:       "resource",
			LabelNames: []string{"type", "name"},
		},
		{
			Type:       "data",
			LabelNames: []string{"type", "name"},
		},
		{
			Type:       "ephemeral",
			LabelNames: []string{"type", "name"},
		},
		{
			Type: "moved",
		},
		{
			Type: "removed",
		},
		{
			Type: "import",
		},
		{
			Type:       "check",
			LabelNames: []string{"name"},
		},
	},
}

// terraformBlockSchema is the schema for a top-level "terraform" block in
// a configuration file.
var terraformBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "required_version"},
		{Name: "experiments"},
		{Name: "language"},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "backend",
			LabelNames: []string{"type"},
		},
		{
			Type: "cloud",
		},
		{
			Type: "required_providers",
		},
		{
			Type:       "provider_meta",
			LabelNames: []string{"provider"},
		},
	},
}

// configFileTerraformBlockSniffRootSchema is a schema for
// sniffCoreVersionRequirements and sniffActiveExperiments.
var configFileTerraformBlockSniffRootSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "terraform",
		},
	},
}

// configFileVersionSniffBlockSchema is a schema for sniffCoreVersionRequirements
var configFileVersionSniffBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "required_version",
		},
	},
}

// configFileExperimentsSniffBlockSchema is a schema for sniffActiveExperiments,
// to decode a single attribute from inside a "terraform" block.
var configFileExperimentsSniffBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "experiments"},
		{Name: "language"},
	},
}
