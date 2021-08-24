package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getmodules"
)

// ModuleCall represents a "module" block in a module or file.
type ModuleCall struct {
	Name string

	SourceAddr      addrs.ModuleSource
	SourceAddrRaw   string
	SourceAddrRange hcl.Range
	SourceSet       bool

	Config hcl.Body

	Version VersionConstraint

	Count   hcl.Expression
	ForEach hcl.Expression

	Providers []PassedProviderConfig

	DependsOn []hcl.Traversal

	DeclRange hcl.Range
}

func decodeModuleBlock(block *hcl.Block, override bool) (*ModuleCall, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	mc := &ModuleCall{
		Name:      block.Labels[0],
		DeclRange: block.DefRange,
	}

	schema := moduleBlockSchema
	if override {
		schema = schemaForOverrides(schema)
	}

	content, remain, moreDiags := block.Body.PartialContent(schema)
	diags = append(diags, moreDiags...)
	mc.Config = remain

	if !hclsyntax.ValidIdentifier(mc.Name) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid module instance name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[0],
		})
	}

	if attr, exists := content.Attributes["source"]; exists {
		mc.SourceSet = true
		mc.SourceAddrRange = attr.Expr.Range()
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &mc.SourceAddrRaw)
		diags = append(diags, valDiags...)
		if !valDiags.HasErrors() {
			addr, err := addrs.ParseModuleSource(mc.SourceAddrRaw)
			mc.SourceAddr = addr
			if err != nil {
				// NOTE: In practice it's actually very unlikely to end up here,
				// because our source address parser can turn just about any string
				// into some sort of remote package address, and so for most errors
				// we'll detect them only during module installation. There are
				// still a _few_ purely-syntax errors we can catch at parsing time,
				// though, mostly related to remote package sub-paths and local
				// paths.
				switch err := err.(type) {
				case *getmodules.MaybeRelativePathErr:
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid module source address",
						Detail: fmt.Sprintf(
							"Terraform failed to determine your intended installation method for remote module package %q.\n\nIf you intended this as a path relative to the current module, use \"./%s\" instead. The \"./\" prefix indicates that the address is a relative filesystem path.",
							err.Addr, err.Addr,
						),
						Subject: mc.SourceAddrRange.Ptr(),
					})
				default:
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid module source address",
						Detail:   fmt.Sprintf("Failed to parse module source address: %s.", err),
						Subject:  mc.SourceAddrRange.Ptr(),
					})
				}
			}
		}
		// NOTE: We leave mc.SourceAddr as nil for any situation where the
		// source attribute is invalid, so any code which tries to carefully
		// use the partial result of a failed config decode must be
		// resilient to that.
	}

	if attr, exists := content.Attributes["version"]; exists {
		var versionDiags hcl.Diagnostics
		mc.Version, versionDiags = decodeVersionConstraint(attr)
		diags = append(diags, versionDiags...)
	}

	if attr, exists := content.Attributes["count"]; exists {
		mc.Count = attr.Expr
	}

	if attr, exists := content.Attributes["for_each"]; exists {
		if mc.Count != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid combination of "count" and "for_each"`,
				Detail:   `The "count" and "for_each" meta-arguments are mutually-exclusive, only one should be used to be explicit about the number of resources to be created.`,
				Subject:  &attr.NameRange,
			})
		}

		mc.ForEach = attr.Expr
	}

	if attr, exists := content.Attributes["depends_on"]; exists {
		deps, depsDiags := decodeDependsOn(attr)
		diags = append(diags, depsDiags...)
		mc.DependsOn = append(mc.DependsOn, deps...)
	}

	if attr, exists := content.Attributes["providers"]; exists {
		seen := make(map[string]hcl.Range)
		pairs, pDiags := hcl.ExprMap(attr.Expr)
		diags = append(diags, pDiags...)
		for _, pair := range pairs {
			key, keyDiags := decodeProviderConfigRef(pair.Key, "providers")
			diags = append(diags, keyDiags...)
			value, valueDiags := decodeProviderConfigRef(pair.Value, "providers")
			diags = append(diags, valueDiags...)
			if keyDiags.HasErrors() || valueDiags.HasErrors() {
				continue
			}

			matchKey := key.String()
			if prev, exists := seen[matchKey]; exists {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate provider address",
					Detail:   fmt.Sprintf("A provider configuration was already passed to %s at %s. Each child provider configuration can be assigned only once.", matchKey, prev),
					Subject:  pair.Value.Range().Ptr(),
				})
				continue
			}

			rng := hcl.RangeBetween(pair.Key.Range(), pair.Value.Range())
			seen[matchKey] = rng
			mc.Providers = append(mc.Providers, PassedProviderConfig{
				InChild:  key,
				InParent: value,
			})
		}
	}

	var seenEscapeBlock *hcl.Block
	for _, block := range content.Blocks {
		switch block.Type {
		case "_":
			if seenEscapeBlock != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate escaping block",
					Detail: fmt.Sprintf(
						"The special block type \"_\" can be used to force particular arguments to be interpreted as module input variables rather than as meta-arguments, but each module block can have only one such block. The first escaping block was at %s.",
						seenEscapeBlock.DefRange,
					),
					Subject: &block.DefRange,
				})
				continue
			}
			seenEscapeBlock = block

			// When there's an escaping block its content merges with the
			// existing config we extracted earlier, so later decoding
			// will see a blend of both.
			mc.Config = hcl.MergeBodies([]hcl.Body{mc.Config, block.Body})

		default:
			// All of the other block types in our schema are reserved.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reserved block type name in module block",
				Detail:   fmt.Sprintf("The block type name %q is reserved for use by Terraform in a future version.", block.Type),
				Subject:  &block.TypeRange,
			})
		}
	}

	return mc, diags
}

// EntersNewPackage returns true if this call is to an external module, either
// directly via a remote source address or indirectly via a registry source
// address.
//
// Other behaviors in Terraform may treat package crossings as a special
// situation, because that indicates that the caller and callee can change
// independently of one another and thus we should disallow using any features
// where the caller assumes anything about the callee other than its input
// variables, required provider configurations, and output values.
func (mc *ModuleCall) EntersNewPackage() bool {
	return moduleSourceAddrEntersNewPackage(mc.SourceAddr)
}

// PassedProviderConfig represents a provider config explicitly passed down to
// a child module, possibly giving it a new local address in the process.
type PassedProviderConfig struct {
	InChild  *ProviderConfigRef
	InParent *ProviderConfigRef
}

var moduleBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "source",
			Required: true,
		},
		{
			Name: "version",
		},
		{
			Name: "count",
		},
		{
			Name: "for_each",
		},
		{
			Name: "depends_on",
		},
		{
			Name: "providers",
		},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "_"}, // meta-argument escaping block

		// These are all reserved for future use.
		{Type: "lifecycle"},
		{Type: "locals"},
		{Type: "provider", LabelNames: []string{"type"}},
	},
}

func moduleSourceAddrEntersNewPackage(addr addrs.ModuleSource) bool {
	switch addr.(type) {
	case nil:
		// There are only two situations where we should get here:
		// - We've been asked about the source address of the root module,
		//   which is always nil.
		// - We've been asked about a ModuleCall that is part of the partial
		//   result of a failed decode.
		// The root module exists outside of all module packages, so we'll
		// just return false for that case. For the error case it doesn't
		// really matter what we return as long as we don't panic, because
		// we only make a best-effort to allow careful inspection of objects
		// representing invalid configuration.
		return false
	case addrs.ModuleSourceLocal:
		// Local source addresses are the only address type that remains within
		// the same package.
		return false
	default:
		// All other address types enter a new package.
		return true
	}
}
