// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getmodules/moduleaddrs"
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

	IgnoreNestedDeprecations bool

	Managed *ManagedModule
}

// ManagedModule represents a "resource" block in a module or file.
type ManagedModule struct {
	Connection     *Connection
	Provisioners   []*Provisioner
	ActionTriggers []*ActionTrigger

	CreateBeforeDestroy bool
	PreventDestroy      bool
	IgnoreChanges       []hcl.Traversal
	IgnoreAllChanges    bool

	CreateBeforeDestroySet bool
	PreventDestroySet      bool
}

func decodeModuleCallLifecycleBlock(block *hcl.Block, mc *ModuleCall) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// Allocate only when lifecycle is present (tests expect nil otherwise).
	if mc.Managed == nil {
		mc.Managed = &ManagedModule{}
	}

	lcContent, lcDiags := block.Body.Content(moduleCallLifecycleBlockSchema)
	diags = append(diags, lcDiags...)

	if attr, exists := lcContent.Attributes["create_before_destroy"]; exists {
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &mc.Managed.CreateBeforeDestroy)
		diags = append(diags, valDiags...)
		mc.Managed.CreateBeforeDestroySet = true
	}

	if attr, exists := lcContent.Attributes["prevent_destroy"]; exists {
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &mc.Managed.PreventDestroy)
		diags = append(diags, valDiags...)
		mc.Managed.PreventDestroySet = true
	}

	if attr, exists := lcContent.Attributes["ignore_changes"]; exists {
		kw := hcl.ExprAsKeyword(attr.Expr)

		switch {
		case kw == "all":
			mc.Managed.IgnoreAllChanges = true

		default:
			exprs, listDiags := hcl.ExprList(attr.Expr)
			diags = append(diags, listDiags...)

			var ignoreAllRange hcl.Range

			for _, expr := range exprs {
				if shimIsIgnoreChangesStar(expr) {
					mc.Managed.IgnoreAllChanges = true
					ignoreAllRange = expr.Range()
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid ignore_changes wildcard",
						Detail:   "The [\"*\"] form of ignore_changes wildcard is was deprecated and is now invalid. Use \"ignore_changes = all\" to ignore changes to all attributes.",
						Subject:  attr.Expr.Range().Ptr(),
					})
					continue
				}

				expr, shimDiags := shimTraversalInString(expr, false)
				diags = append(diags, shimDiags...)

				traversal, travDiags := hcl.RelTraversalForExpr(expr)
				diags = append(diags, travDiags...)
				if len(traversal) != 0 {
					mc.Managed.IgnoreChanges = append(mc.Managed.IgnoreChanges, traversal)
				}
			}

			if mc.Managed.IgnoreAllChanges && len(mc.Managed.IgnoreChanges) != 0 {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid ignore_changes ruleset",
					Detail:   "Cannot mix wildcard string \"*\" with non-wildcard references.",
					Subject:  &ignoreAllRange,
					Context:  attr.Expr.Range().Ptr(),
				})
			}
		}
	}

	return diags
}

func decodeModuleBlock(block *hcl.Block, override bool) (*ModuleCall, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	mc := &ModuleCall{
		Name:      block.Labels[0],
		DeclRange: block.DefRange,
		// IMPORTANT: Managed must stay nil unless we actually decode a lifecycle
		// (or other managed-module features in future). Tests depend on this.
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

	haveVersionArg := false
	if attr, exists := content.Attributes["version"]; exists {
		var versionDiags hcl.Diagnostics
		mc.Version, versionDiags = decodeVersionConstraint(attr)
		diags = append(diags, versionDiags...)
		haveVersionArg = true
	}

	if attr, exists := content.Attributes["source"]; exists {
		mc.SourceSet = true
		mc.SourceAddrRange = attr.Expr.Range()
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &mc.SourceAddrRaw)
		diags = append(diags, valDiags...)
		if !valDiags.HasErrors() {
			var addr addrs.ModuleSource
			var err error
			if haveVersionArg {
				addr, err = moduleaddrs.ParseModuleSourceRegistry(mc.SourceAddrRaw)
			} else {
				addr, err = moduleaddrs.ParseModuleSource(mc.SourceAddrRaw)
			}
			mc.SourceAddr = addr
			if err != nil {
				// NOTE: We leave mc.SourceAddr as nil for any situation where the
				// source attribute is invalid, so any code which tries to carefully
				// use the partial result of a failed config decode must be
				// resilient to that.
				mc.SourceAddr = nil

				// NOTE: In practice it's actually very unlikely to end up here,
				// because our source address parser can turn just about any string
				// into some sort of remote package address, and so for most errors
				// we'll detect them only during module installation. There are
				// still a _few_ purely-syntax errors we can catch at parsing time,
				// though, mostly related to remote package sub-paths and local
				// paths.
				switch err := err.(type) {
				case *moduleaddrs.MaybeRelativePathErr:
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
					if haveVersionArg {
						// In this case we'll include some extra context that
						// we assumed a registry source address due to the
						// version argument.
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid registry module source address",
							Detail:   fmt.Sprintf("Failed to parse module registry address: %s.\n\nTerraform assumed that you intended a module registry source address because you also set the argument \"version\", which applies only to registry modules.", err),
							Subject:  mc.SourceAddrRange.Ptr(),
						})
					} else {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid module source address",
							Detail:   fmt.Sprintf("Failed to parse module source address: %s.", err),
							Subject:  mc.SourceAddrRange.Ptr(),
						})
					}
				}
			}
		}
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
		deps, depsDiags := DecodeDependsOn(attr)
		diags = append(diags, depsDiags...)
		mc.DependsOn = append(mc.DependsOn, deps...)
	}

	if attr, exists := content.Attributes["providers"]; exists {
		providers, providerDiags := decodePassedProviderConfigs(attr)
		diags = append(diags, providerDiags...)
		mc.Providers = append(mc.Providers, providers...)
	}

	if attr, exists := content.Attributes["ignore_nested_deprecations"]; exists {
		// We only allow static boolean values for this argument.
		val, evalDiags := attr.Expr.Value(&hcl.EvalContext{})
		if len(evalDiags.Errs()) > 0 {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid value for ignore_nested_deprecations",
				Detail:   "The value for ignore_nested_deprecations must be a static boolean (true or false).",
				Subject:  attr.Expr.Range().Ptr(),
			})
		}

		if val.Type() != cty.Bool {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid type for ignore_nested_deprecations",
				Detail:   fmt.Sprintf("The value for ignore_nested_deprecations must be a boolean (true or false), but the given value has type %s.", val.Type().FriendlyName()),
				Subject:  attr.Expr.Range().Ptr(),
			})
		}

		mc.IgnoreNestedDeprecations = val.True()
	}

	var seenEscapeBlock *hcl.Block
	var seenLifecycle *hcl.Block

	for _, inner := range content.Blocks {
		switch inner.Type {
		case "_":
			if seenEscapeBlock != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate escaping block",
					Detail: fmt.Sprintf(
						"The special block type \"_\" can be used to force particular arguments to be interpreted as module input variables rather than as meta-arguments, but each module block can have only one such block. The first escaping block was at %s.",
						seenEscapeBlock.DefRange,
					),
					Subject: &inner.DefRange,
				})
				continue
			}
			seenEscapeBlock = inner
			mc.Config = hcl.MergeBodies([]hcl.Body{mc.Config, inner.Body})

		case "lifecycle":
			if seenLifecycle != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate lifecycle block",
					Detail:   fmt.Sprintf("This module call already has a lifecycle block at %s.", seenLifecycle.DefRange),
					Subject:  &inner.DefRange,
				})
				continue
			}
			seenLifecycle = inner

			diags = append(diags, decodeModuleCallLifecycleBlock(inner, mc)...)

		default:
			// All of the other block types in our schema are reserved.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reserved block type name in module block",
				Detail:   fmt.Sprintf("The block type name %q is reserved for use by Terraform in a future version.", inner.Type),
				Subject:  &inner.TypeRange,
			})
		}
	}

	return mc, diags
}

// PassedProviderConfig represents a provider config explicitly passed down to
// a child module, possibly giving it a new local address in the process.
type PassedProviderConfig struct {
	InChild  *ProviderConfigRef
	InParent *ProviderConfigRef
}

func decodePassedProviderConfigs(attr *hcl.Attribute) ([]PassedProviderConfig, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var providers []PassedProviderConfig

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
		providers = append(providers, PassedProviderConfig{
			InChild:  key,
			InParent: value,
		})
	}
	return providers, diags
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
		{
			Name: "ignore_nested_deprecations",
		},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "_"}, // meta-argument escaping block

		// "lifecycle" is interpreted as Terraform meta-configuration for this module call.
		// Other block types are reserved for future use.
		{Type: "lifecycle"},
		{Type: "locals"},
		{Type: "provider", LabelNames: []string{"type"}},
	},
}

var moduleCallLifecycleBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "create_before_destroy"},
		{Name: "prevent_destroy"},
		{Name: "ignore_changes"},
	},
}
