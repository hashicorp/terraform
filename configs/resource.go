package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Resource represents a "resource" or "data" block in a module or file.
type Resource struct {
	Mode    addrs.ResourceMode
	Name    string
	Type    string
	Config  hcl.Body
	Count   hcl.Expression
	ForEach hcl.Expression

	ProviderConfigRef *ProviderConfigRef
	Provider          addrs.Provider

	DependsOn []hcl.Traversal

	// Managed is populated only for Mode = addrs.ManagedResourceMode,
	// containing the additional fields that apply to managed resources.
	// For all other resource modes, this field is nil.
	Managed *ManagedResource

	DeclRange hcl.Range
	TypeRange hcl.Range
}

// ManagedResource represents a "resource" block in a module or file.
type ManagedResource struct {
	Connection   *Connection
	Provisioners []*Provisioner

	CreateBeforeDestroy bool
	PreventDestroy      bool
	IgnoreChanges       []hcl.Traversal
	IgnoreAllChanges    bool

	CreateBeforeDestroySet bool
	PreventDestroySet      bool
}

func (r *Resource) moduleUniqueKey() string {
	return r.Addr().String()
}

// Addr returns a resource address for the receiver that is relative to the
// resource's containing module.
func (r *Resource) Addr() addrs.Resource {
	return addrs.Resource{
		Mode: r.Mode,
		Type: r.Type,
		Name: r.Name,
	}
}

// ProviderConfigAddr returns the address for the provider configuration that
// should be used for this resource. This function returns a default provider
// config addr if an explicit "provider" argument was not provided.
func (r *Resource) ProviderConfigAddr() addrs.LocalProviderConfig {
	if r.ProviderConfigRef == nil {
		// If no specific "provider" argument is given, we want to look up the
		// provider config where the local name matches the implied provider
		// from the resource type. This may be different from the resource's
		// provider type.
		return addrs.LocalProviderConfig{
			LocalName: r.Addr().ImpliedProvider(),
		}
	}

	return addrs.LocalProviderConfig{
		LocalName: r.ProviderConfigRef.Name,
		Alias:     r.ProviderConfigRef.Alias,
	}
}

func decodeResourceBlock(block *hcl.Block) (*Resource, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	r := &Resource{
		Mode:      addrs.ManagedResourceMode,
		Type:      block.Labels[0],
		Name:      block.Labels[1],
		DeclRange: block.DefRange,
		TypeRange: block.LabelRanges[0],
		Managed:   &ManagedResource{},
	}

	content, remain, moreDiags := block.Body.PartialContent(resourceBlockSchema)
	diags = append(diags, moreDiags...)
	r.Config = remain

	if !hclsyntax.ValidIdentifier(r.Type) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid resource type name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[0],
		})
	}
	if !hclsyntax.ValidIdentifier(r.Name) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid resource name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[1],
		})
	}

	if attr, exists := content.Attributes["count"]; exists {
		r.Count = attr.Expr
	}

	if attr, exists := content.Attributes["for_each"]; exists {
		r.ForEach = attr.Expr
		// Cannot have count and for_each on the same resource block
		if r.Count != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid combination of "count" and "for_each"`,
				Detail:   `The "count" and "for_each" meta-arguments are mutually-exclusive, only one should be used to be explicit about the number of resources to be created.`,
				Subject:  &attr.NameRange,
			})
		}
	}

	if attr, exists := content.Attributes["provider"]; exists {
		var providerDiags hcl.Diagnostics
		r.ProviderConfigRef, providerDiags = decodeProviderConfigRef(attr.Expr, "provider")
		diags = append(diags, providerDiags...)
	}

	if attr, exists := content.Attributes["depends_on"]; exists {
		deps, depsDiags := decodeDependsOn(attr)
		diags = append(diags, depsDiags...)
		r.DependsOn = append(r.DependsOn, deps...)
	}

	var seenLifecycle *hcl.Block
	var seenConnection *hcl.Block
	var seenEscapeBlock *hcl.Block
	for _, block := range content.Blocks {
		switch block.Type {
		case "lifecycle":
			if seenLifecycle != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate lifecycle block",
					Detail:   fmt.Sprintf("This resource already has a lifecycle block at %s.", seenLifecycle.DefRange),
					Subject:  &block.DefRange,
				})
				continue
			}
			seenLifecycle = block

			lcContent, lcDiags := block.Body.Content(resourceLifecycleBlockSchema)
			diags = append(diags, lcDiags...)

			if attr, exists := lcContent.Attributes["create_before_destroy"]; exists {
				valDiags := gohcl.DecodeExpression(attr.Expr, nil, &r.Managed.CreateBeforeDestroy)
				diags = append(diags, valDiags...)
				r.Managed.CreateBeforeDestroySet = true
			}

			if attr, exists := lcContent.Attributes["prevent_destroy"]; exists {
				valDiags := gohcl.DecodeExpression(attr.Expr, nil, &r.Managed.PreventDestroy)
				diags = append(diags, valDiags...)
				r.Managed.PreventDestroySet = true
			}

			if attr, exists := lcContent.Attributes["ignore_changes"]; exists {

				// ignore_changes can either be a list of relative traversals
				// or it can be just the keyword "all" to ignore changes to this
				// resource entirely.
				//   ignore_changes = [ami, instance_type]
				//   ignore_changes = all
				// We also allow two legacy forms for compatibility with earlier
				// versions:
				//   ignore_changes = ["ami", "instance_type"]
				//   ignore_changes = ["*"]

				kw := hcl.ExprAsKeyword(attr.Expr)

				switch {
				case kw == "all":
					r.Managed.IgnoreAllChanges = true
				default:
					exprs, listDiags := hcl.ExprList(attr.Expr)
					diags = append(diags, listDiags...)

					var ignoreAllRange hcl.Range

					for _, expr := range exprs {

						// our expr might be the literal string "*", which
						// we accept as a deprecated way of saying "all".
						if shimIsIgnoreChangesStar(expr) {
							r.Managed.IgnoreAllChanges = true
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
							r.Managed.IgnoreChanges = append(r.Managed.IgnoreChanges, traversal)
						}
					}

					if r.Managed.IgnoreAllChanges && len(r.Managed.IgnoreChanges) != 0 {
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

		case "connection":
			if seenConnection != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate connection block",
					Detail:   fmt.Sprintf("This resource already has a connection block at %s.", seenConnection.DefRange),
					Subject:  &block.DefRange,
				})
				continue
			}
			seenConnection = block

			r.Managed.Connection = &Connection{
				Config:    block.Body,
				DeclRange: block.DefRange,
			}

		case "provisioner":
			pv, pvDiags := decodeProvisionerBlock(block)
			diags = append(diags, pvDiags...)
			if pv != nil {
				r.Managed.Provisioners = append(r.Managed.Provisioners, pv)
			}

		case "_":
			if seenEscapeBlock != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate escaping block",
					Detail: fmt.Sprintf(
						"The special block type \"_\" can be used to force particular arguments to be interpreted as resource-type-specific rather than as meta-arguments, but each resource block can have only one such block. The first escaping block was at %s.",
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
			r.Config = hcl.MergeBodies([]hcl.Body{r.Config, block.Body})

		default:
			// Any other block types are ones we've reserved for future use,
			// so they get a generic message.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reserved block type name in resource block",
				Detail:   fmt.Sprintf("The block type name %q is reserved for use by Terraform in a future version.", block.Type),
				Subject:  &block.TypeRange,
			})
		}
	}

	// Now we can validate the connection block references if there are any destroy provisioners.
	// TODO: should we eliminate standalone connection blocks?
	if r.Managed.Connection != nil {
		for _, p := range r.Managed.Provisioners {
			if p.When == ProvisionerWhenDestroy {
				diags = append(diags, onlySelfRefs(r.Managed.Connection.Config)...)
				break
			}
		}
	}

	return r, diags
}

func decodeDataBlock(block *hcl.Block) (*Resource, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	r := &Resource{
		Mode:      addrs.DataResourceMode,
		Type:      block.Labels[0],
		Name:      block.Labels[1],
		DeclRange: block.DefRange,
		TypeRange: block.LabelRanges[0],
	}

	content, remain, moreDiags := block.Body.PartialContent(dataBlockSchema)
	diags = append(diags, moreDiags...)
	r.Config = remain

	if !hclsyntax.ValidIdentifier(r.Type) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid data source name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[0],
		})
	}
	if !hclsyntax.ValidIdentifier(r.Name) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid data resource name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[1],
		})
	}

	if attr, exists := content.Attributes["count"]; exists {
		r.Count = attr.Expr
	}

	if attr, exists := content.Attributes["for_each"]; exists {
		r.ForEach = attr.Expr
		// Cannot have count and for_each on the same data block
		if r.Count != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid combination of "count" and "for_each"`,
				Detail:   `The "count" and "for_each" meta-arguments are mutually-exclusive, only one should be used to be explicit about the number of resources to be created.`,
				Subject:  &attr.NameRange,
			})
		}
	}

	if attr, exists := content.Attributes["provider"]; exists {
		var providerDiags hcl.Diagnostics
		r.ProviderConfigRef, providerDiags = decodeProviderConfigRef(attr.Expr, "provider")
		diags = append(diags, providerDiags...)
	}

	if attr, exists := content.Attributes["depends_on"]; exists {
		deps, depsDiags := decodeDependsOn(attr)
		diags = append(diags, depsDiags...)
		r.DependsOn = append(r.DependsOn, deps...)
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
						"The special block type \"_\" can be used to force particular arguments to be interpreted as resource-type-specific rather than as meta-arguments, but each data block can have only one such block. The first escaping block was at %s.",
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
			r.Config = hcl.MergeBodies([]hcl.Body{r.Config, block.Body})

		// The rest of these are just here to reserve block type names for future use.
		case "lifecycle":
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsupported lifecycle block",
				Detail:   "Data resources do not have lifecycle settings, so a lifecycle block is not allowed.",
				Subject:  &block.DefRange,
			})

		default:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reserved block type name in data block",
				Detail:   fmt.Sprintf("The block type name %q is reserved for use by Terraform in a future version.", block.Type),
				Subject:  &block.TypeRange,
			})
		}
	}

	return r, diags
}

type ProviderConfigRef struct {
	Name       string
	NameRange  hcl.Range
	Alias      string
	AliasRange *hcl.Range // nil if alias not set

	// TODO: this may not be set in some cases, so it is not yet suitable for
	// use outside of this package. We currently only use it for internal
	// validation, but once we verify that this can be set in all cases, we can
	// export this so providers don't need to be re-resolved.
	// This same field is also added to the Provider struct.
	providerType addrs.Provider
}

func decodeProviderConfigRef(expr hcl.Expression, argName string) (*ProviderConfigRef, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	var shimDiags hcl.Diagnostics
	expr, shimDiags = shimTraversalInString(expr, false)
	diags = append(diags, shimDiags...)

	traversal, travDiags := hcl.AbsTraversalForExpr(expr)

	// AbsTraversalForExpr produces only generic errors, so we'll discard
	// the errors given and produce our own with extra context. If we didn't
	// get any errors then we might still have warnings, though.
	if !travDiags.HasErrors() {
		diags = append(diags, travDiags...)
	}

	if len(traversal) < 1 || len(traversal) > 2 {
		// A provider reference was given as a string literal in the legacy
		// configuration language and there are lots of examples out there
		// showing that usage, so we'll sniff for that situation here and
		// produce a specialized error message for it to help users find
		// the new correct form.
		if exprIsNativeQuotedString(expr) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider configuration reference",
				Detail:   "A provider configuration reference must not be given in quotes.",
				Subject:  expr.Range().Ptr(),
			})
			return nil, diags
		}

		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider configuration reference",
			Detail:   fmt.Sprintf("The %s argument requires a provider type name, optionally followed by a period and then a configuration alias.", argName),
			Subject:  expr.Range().Ptr(),
		})
		return nil, diags
	}

	// verify that the provider local name is normalized
	name := traversal.RootName()
	nameDiags := checkProviderNameNormalized(name, traversal[0].SourceRange())
	diags = append(diags, nameDiags...)
	if diags.HasErrors() {
		return nil, diags
	}

	ret := &ProviderConfigRef{
		Name:      name,
		NameRange: traversal[0].SourceRange(),
	}

	if len(traversal) > 1 {
		aliasStep, ok := traversal[1].(hcl.TraverseAttr)
		if !ok {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider configuration reference",
				Detail:   "Provider name must either stand alone or be followed by a period and then a configuration alias.",
				Subject:  traversal[1].SourceRange().Ptr(),
			})
			return ret, diags
		}

		ret.Alias = aliasStep.Name
		ret.AliasRange = aliasStep.SourceRange().Ptr()
	}

	return ret, diags
}

// Addr returns the provider config address corresponding to the receiving
// config reference.
//
// This is a trivial conversion, essentially just discarding the source
// location information and keeping just the addressing information.
func (r *ProviderConfigRef) Addr() addrs.LocalProviderConfig {
	return addrs.LocalProviderConfig{
		LocalName: r.Name,
		Alias:     r.Alias,
	}
}

func (r *ProviderConfigRef) String() string {
	if r == nil {
		return "<nil>"
	}
	if r.Alias != "" {
		return fmt.Sprintf("%s.%s", r.Name, r.Alias)
	}
	return r.Name
}

var commonResourceAttributes = []hcl.AttributeSchema{
	{
		Name: "count",
	},
	{
		Name: "for_each",
	},
	{
		Name: "provider",
	},
	{
		Name: "depends_on",
	},
}

var resourceBlockSchema = &hcl.BodySchema{
	Attributes: commonResourceAttributes,
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "locals"}, // reserved for future use
		{Type: "lifecycle"},
		{Type: "connection"},
		{Type: "provisioner", LabelNames: []string{"type"}},
		{Type: "_"}, // meta-argument escaping block
	},
}

var dataBlockSchema = &hcl.BodySchema{
	Attributes: commonResourceAttributes,
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "lifecycle"}, // reserved for future use
		{Type: "locals"},    // reserved for future use
		{Type: "_"},         // meta-argument escaping block
	},
}

var resourceLifecycleBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "create_before_destroy",
		},
		{
			Name: "prevent_destroy",
		},
		{
			Name: "ignore_changes",
		},
	},
}
