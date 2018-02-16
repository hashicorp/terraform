package configs

import (
	"github.com/hashicorp/hcl2/hcl"
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

func (p *Parser) loadConfigFile(path string, override bool) (*File, hcl.Diagnostics) {

	body, diags := p.LoadHCLFile(path)
	if body == nil {
		return nil, diags
	}

	file := &File{}

	var reqDiags hcl.Diagnostics
	file.CoreVersionConstraints, reqDiags = sniffCoreVersionRequirements(body)
	diags = append(diags, reqDiags...)

	content, contentDiags := body.Content(configFileSchema)
	diags = append(diags, contentDiags...)

	for _, block := range content.Blocks {
		switch block.Type {

		case "terraform":
			content, contentDiags := block.Body.Content(terraformBlockSchema)
			diags = append(diags, contentDiags...)

			// We ignore the "terraform_version" attribute here because
			// sniffCoreVersionRequirements already dealt with that above.

			for _, innerBlock := range content.Blocks {
				switch innerBlock.Type {

				case "backend":
					backendCfg, cfgDiags := decodeBackendBlock(innerBlock)
					diags = append(diags, cfgDiags...)
					if backendCfg != nil {
						file.Backends = append(file.Backends, backendCfg)
					}

				case "required_providers":
					reqs, reqsDiags := decodeRequiredProvidersBlock(innerBlock)
					diags = append(diags, reqsDiags...)
					file.ProviderRequirements = append(file.ProviderRequirements, reqs...)

				default:
					// Should never happen because the above cases should be exhaustive
					// for all block type names in our schema.
					continue

				}
			}

		case "provider":
			cfg, cfgDiags := decodeProviderBlock(block)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.ProviderConfigs = append(file.ProviderConfigs, cfg)
			}

		case "variable":
			cfg, cfgDiags := decodeVariableBlock(block)
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
			cfg, cfgDiags := decodeResourceBlock(block)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.ManagedResources = append(file.ManagedResources, cfg)
			}

		case "data":
			cfg, cfgDiags := decodeDataBlock(block)
			diags = append(diags, cfgDiags...)
			if cfg != nil {
				file.DataResources = append(file.DataResources, cfg)
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
	rootContent, _, diags := body.PartialContent(configFileVersionSniffRootSchema)

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
	},
}

// terraformBlockSchema is the schema for a top-level "terraform" block in
// a configuration file.
var terraformBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "required_version",
		},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "backend",
			LabelNames: []string{"type"},
		},
		{
			Type: "required_providers",
		},
	},
}

// configFileVersionSniffRootSchema is a schema for sniffCoreVersionRequirements
var configFileVersionSniffRootSchema = &hcl.BodySchema{
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
