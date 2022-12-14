package testconfigs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Scenario represents the configuration for an entire testing scenario.
type Scenario struct {
	Name     string
	Filename string

	ProviderReqs        *configs.RequiredProviders
	RealProviderConfigs map[addrs.LocalProviderConfig]*configs.Provider
	MockProviderConfigs map[addrs.LocalProviderConfig]*MockProvider
	Steps               map[string]*Step
	StepsOrder          []string
}

func LoadScenarioFile(filename string) (*Scenario, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	ret := &Scenario{
		Filename: filename,
	}

	ext := filepath.Ext(filename)
	if ext != ".tftest" {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid test scenario filename",
			fmt.Sprintf("Can't use %q as a test scenario: filename must have .tftest suffix.", filename),
		))
		return ret, diags
	}
	baseName := filepath.Base(filename[:len(filename)-len(ext)])

	if !hclsyntax.ValidIdentifier(baseName) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid test scenario filename",
			fmt.Sprintf("Can't use %q as a test scenario: file base name (ignoring the .tftest suffix) must be a valid identifier.", filename),
		))
		return ret, diags
	}
	ret.Name = baseName

	f, err := os.Open(filename)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Cannot read test scenario file",
			fmt.Sprintf("Error reading test scenario configuration from %s: %s.", filename, err),
		))
		return ret, diags
	}
	src, err := io.ReadAll(f)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Cannot read test scenario file",
			fmt.Sprintf("Error reading test scenario configuration from %s: %s.", filename, err),
		))
		return ret, diags
	}

	hclF, hclDiags := hclsyntax.ParseConfig(src, filename, hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return ret, diags
	}

	ret.RealProviderConfigs = make(map[addrs.LocalProviderConfig]*configs.Provider)
	ret.MockProviderConfigs = make(map[addrs.LocalProviderConfig]*MockProvider)
	ret.Steps = make(map[string]*Step)

	content, hclDiags := hclF.Body.Content(&scenarioFileSchema)
	diags = diags.Append(hclDiags)
	diags = diags.Append(checkScenarioConfigBlockOrder(content.Blocks))

	for _, block := range content.Blocks {
		switch block.Type {
		case "required_providers":
			rp, hclDiags := configs.DecodeRequiredProvidersBlock(block)
			diags = diags.Append(hclDiags)
			if ret.ProviderReqs == nil {
				ret.ProviderReqs = rp
			} else {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate required_providers block",
					Detail:   fmt.Sprintf("This test scenario already defined its provider requirements at %s.", ret.ProviderReqs.DeclRange),
					Subject:  rp.DeclRange.Ptr(),
				})
			}
		case "provider":
			p, hclDiags := configs.DecodeProviderBlock(block)
			diags = diags.Append(hclDiags)
			newAddr := p.Addr()
			if existing, exists := ret.RealProviderConfigs[newAddr]; exists {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate provider configuration",
					Detail:   fmt.Sprintf("This provider was already configured at %s.", existing.DeclRange),
					Subject:  p.DeclRange.Ptr(),
				})
			} else if existing, exists := ret.MockProviderConfigs[newAddr]; exists {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate provider configuration",
					Detail:   fmt.Sprintf("This provider configuration conflicts with a mock provider instance declared at %s.", existing.DeclRange),
					Subject:  p.DeclRange.Ptr(),
				})
			} else {
				ret.RealProviderConfigs[newAddr] = p
			}
		case "mock_provider":
			p, moreDiags := decodeMockProviderBlock(block)
			diags = diags.Append(moreDiags)
			newAddr := p.Addr
			if existing, exists := ret.RealProviderConfigs[newAddr]; exists {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate provider configuration",
					Detail:   fmt.Sprintf("This mock provider instance conflicts with the provider configuration at %s.", existing.DeclRange),
					Subject:  p.DeclRange.Ptr(),
				})
			} else if existing, exists := ret.MockProviderConfigs[newAddr]; exists {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate provider configuration",
					Detail:   fmt.Sprintf("This provider configuration conflicts with another mock provider instance declared at %s.", existing.DeclRange),
					Subject:  p.DeclRange.Ptr(),
				})
			} else {
				ret.MockProviderConfigs[newAddr] = p
			}
		case "step":
			s, moreDiags := decodeStepBlock(block)
			diags = diags.Append(moreDiags)
			if existing, exists := ret.Steps[s.Name]; exists {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate step block",
					Detail:   fmt.Sprintf("A step named %q was already declared at %s.", s.Name, existing.DeclRange),
					Subject:  s.DeclRange.Ptr(),
				})
			} else {
				ret.Steps[s.Name] = s
				ret.StepsOrder = append(ret.StepsOrder, s.Name)
			}
		default:
			// Should not get here because the cases above should cover all
			// of the block types in scenarioFileSchema.
			panic(fmt.Sprintf("unhandled scenario file block type %q", block.Type))
		}
	}

	// Before we'll return we'll do some normalization of the steps to make
	// life easier for the caller.
	var defProviders []*configs.PassedProviderConfig
	for _, cfg := range ret.RealProviderConfigs {
		if cfg.Alias == "" {
			defProviders = append(defProviders, &configs.PassedProviderConfig{
				InChild: &configs.ProviderConfigRef{
					Name:      cfg.Name,
					NameRange: cfg.NameRange,
				},
			})
		}
	}
	for _, cfg := range ret.MockProviderConfigs {
		if cfg.Addr.Alias == "" {
			defProviders = append(defProviders, &configs.PassedProviderConfig{
				InChild: &configs.ProviderConfigRef{
					Name:      cfg.Addr.LocalName,
					NameRange: cfg.DeclRange,
				},
			})
		}
	}
	for _, step := range ret.Steps {
		// NOTE: We're intentionally testing for nil here, rather than length
		// zero, because it's valid (albeit strange) for a step to have
		// providers = {} explicitly, which is different than omitting it.
		if step.Providers == nil {
			step.Providers = defProviders
		}
	}
	for _, pc := range ret.RealProviderConfigs {
		if _, ok := ret.ProviderReqs.RequiredProviders[pc.Name]; !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Configuration for unknown provider",
				Detail:   fmt.Sprintf("The required_providers block does not include an entry with the local name %q.", pc.Name),
				Subject:  pc.NameRange.Ptr(),
			})
		}
	}
	for _, pc := range ret.MockProviderConfigs {
		if _, ok := ret.ProviderReqs.RequiredProviders[pc.Addr.LocalName]; !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Mock instance for unknown provider",
				Detail:   fmt.Sprintf("The required_providers block does not include an entry with the local name %q.", pc.Addr.LocalName),
				Subject:  pc.DeclRange.Ptr(),
			})
		}
	}
	for _, s := range ret.Steps {
		for _, pp := range s.Providers {
			if _, ok := ret.ProviderReqs.RequiredProviders[pp.InChild.Name]; !ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to unknown provider",
					Detail:   fmt.Sprintf("The required_providers block does not include an entry with the local name %q.", pp.InChild.Name),
					Subject:  pp.InChild.NameRange.Ptr(),
				})
			}
			if _, ok := ret.ProviderReqs.RequiredProviders[pp.InParent.Name]; !ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to unknown provider",
					Detail:   fmt.Sprintf("The required_providers block does not include an entry with the local name %q.", pp.InParent.Name),
					Subject:  pp.InParent.NameRange.Ptr(),
				})
			}
		}
	}

	return ret, diags
}

func checkScenarioConfigBlockOrder(blocks []*hcl.Block) tfdiags.Diagnostics {
	// To help keep the scenario files easy to read and consistent, we require
	// the block types to be in a particular order.
	//
	// The order of the "step" blocks also represents the execution order for
	// the steps, but this function doesn't do anything to verify that.

	var diags tfdiags.Diagnostics
	seenProviders := 0
	seenSteps := 0

	for _, block := range blocks {
		switch block.Type {
		case "required_providers":
			if seenProviders > 0 || seenSteps > 0 {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Misplaced required_providers block",
					Detail:   "The required_providers block must be the first block in a test scenario file.",
					Subject:  block.DefRange.Ptr(),
				})
			}
		case "provider", "mock_provider":
			if seenSteps > 0 {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Misplaced provider configuration",
					Detail:   "Provider configurations must all appear before declaring any steps.",
					Subject:  block.DefRange.Ptr(),
				})
			}
			seenProviders++
		case "step":
			seenSteps++
		default:
			// Should not get here because the cases above should cover all
			// of the block types in scenarioFileSchema.
			panic(fmt.Sprintf("unhandled scenario file block type %q", block.Type))
		}
	}

	return diags
}

var scenarioFileSchema = hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "required_providers"},
		{Type: "provider", LabelNames: []string{"local_name"}},
		{Type: "mock_provider", LabelNames: []string{"local_name"}},
		{Type: "step", LabelNames: []string{"name"}},
	},
}
