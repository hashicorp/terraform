// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"fmt"

	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	stackparser "github.com/hashicorp/terraform/internal/stacks/stackconfig/parser"
	"github.com/hashicorp/terraform/internal/stacks/stackmodule"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// EphemeralModule represents the declaration of a single ephemeral module within a
// particular [Stack].
//
// An ephemeral module represents a stack declaration that evaluates a Terraform
// module to produce ephemeral data. It cannot create a plan or state, so its
// module tree cannot include managed resources and any root outputs must be ephemeral.
type EphemeralModule struct {
	Name string

	SourceAddr                               sourceaddrs.Source
	VersionConstraints                       constraints.IntersectionSpec
	SourceAddrRange, VersionConstraintsRange tfdiags.SourceRange

	// FinalSourceAddr is populated only when a configuration is loaded
	// through [LoadConfigDir], and in that case contains the finalized
	// address produced by resolving the SourceAddr field relative to
	// the address of the file where the ephemeral module was declared. This
	// is the address to use if you intend to load the ephemeral module's
	// root module from a source bundle.
	//
	// If this EphemeralModule was created through one of the narrower configuration
	// loading functions, such as [LoadSingleStackConfig] or [ParseFileSource],
	// then this field will be nil and it won't be possible to determine the
	// finalized source location for the root module.
	FinalSourceAddr sourceaddrs.FinalSource

	// Inputs is an expression that should produce a value that can convert
	// to an object type derived from the ephemeral module's input variable
	// declarations, and whose attribute values will then be used to populate
	// those input variables.
	Inputs hcl.Expression

	// ProviderConfigs describes the mapping between the static provider
	// configuration slots declared in the ephemeral module's root module and the
	// dynamic provider configuration objects in scope in the calling
	// stack configuration.
	//
	// This map deals with the slight schism between the stacks language's
	// treatment of provider configurations as regular values of a special
	// data type vs. the main Terraform language's treatment of provider
	// configurations as something special passed out of band from the
	// input variables. The overall structure and the map keys are fixed
	// statically during decoding, but the final provider configuration objects
	// are determined only at runtime by normal expression evaluation.
	//
	// The keys of this map refer to provider configuration slots inside
	// the module being called, but use the local names defined in the
	// calling stack configuration. The stacks language runtime will
	// translate the caller's local names into the callee's declared provider
	// configurations by using the stack configuration's table of local
	// provider names.
	ProviderConfigs map[addrs.LocalProviderConfig]hcl.Expression

	ForEach   hcl.Expression
	DeclRange tfdiags.SourceRange
}

// ModuleConfig returns the module configuration for the given address within
// the provided source bundle.
func (c *EphemeralModule) ModuleConfig(bundle *sourcebundle.Bundle) (*configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	parser := configs.NewSourceBundleParser(bundle)
	if !parser.IsConfigDir(c.FinalSourceAddr) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Ephemeral module configuration not found",
			Detail:   fmt.Sprintf("No module configuration found for ephemeral module %q at %s.", c.Name, c.FinalSourceAddr),
			Subject:  c.SourceAddrRange.ToHCL().Ptr(),
		})
		return nil, diags
	}

	module, moreDiags := parser.LoadConfigDir(c.FinalSourceAddr)
	diags = diags.Append(moreDiags)

	if module != nil {
		walker := stackparser.NewSourceBundleModuleWalker(c.FinalSourceAddr, bundle, parser)
		config, moreDiags := stackmodule.BuildConfig(module, walker, nil)
		diags = diags.Append(moreDiags)
		return config, diags
	}

	return nil, diags
}

var ephemeralModuleBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "source", Required: true},
		{Name: "version", Required: false},
		{Name: "for_each", Required: false},
		{Name: "inputs", Required: false},
		{Name: "providers", Required: false},
	},
}

func decodeEphemeralModuleBlock(block *hcl.Block) (*EphemeralModule, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &EphemeralModule{
		Name:      block.Labels[0],
		DeclRange: tfdiags.SourceRangeFromHCL(block.DefRange),
	}
	if !hclsyntax.ValidIdentifier(ret.Name) {
		diags = diags.Append(invalidNameDiagnostic(
			"Invalid ephemeral module name",
			block.LabelRanges[0],
		))
		return nil, diags
	}

	content, hclDiags := block.Body.Content(ephemeralModuleBlockSchema)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return nil, diags
	}

	sourceAddr, versionConstraints, moreDiags := decodeSourceAddrArguments(
		content.Attributes["source"],
		content.Attributes["version"],
	)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	ret.SourceAddr = sourceAddr
	ret.VersionConstraints = versionConstraints
	ret.SourceAddrRange = tfdiags.SourceRangeFromHCL(content.Attributes["source"].Range)
	if content.Attributes["version"] != nil {
		ret.VersionConstraintsRange = tfdiags.SourceRangeFromHCL(content.Attributes["version"].Range)
	}
	// Now that we've populated the mandatory source location fields we can
	// safely return a partial ret if we encounter any further errors, as
	// long as we leave the other fields either unset or in some other
	// reasonable state for careful partial analysis.

	if attr, ok := content.Attributes["for_each"]; ok {
		ret.ForEach = attr.Expr
	}
	if attr, ok := content.Attributes["inputs"]; ok {
		ret.Inputs = attr.Expr
	}
	if attr, ok := content.Attributes["providers"]; ok {
		var providerDiags tfdiags.Diagnostics
		ret.ProviderConfigs, providerDiags = decodeProvidersAttribute(attr)
		diags = diags.Append(providerDiags)
	}

	return ret, diags
}
