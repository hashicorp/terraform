package configs

import (
	"fmt"

	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"

	"github.com/hashicorp/terraform/addrs"
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

// ProviderConfigAddr returns the address for the provider configuration
// that should be used for this resource. This function implements the
// default behavior of extracting the type from the resource type name if
// an explicit "provider" argument was not provided.
func (r *Resource) ProviderConfigAddr() addrs.ProviderConfig {
	if r.ProviderConfigRef == nil {
		return r.Addr().DefaultProviderConfig()
	}

	return addrs.ProviderConfig{
		Type:  r.ProviderConfigRef.Name,
		Alias: r.ProviderConfigRef.Alias,
	}
}

func decodeResourceBlock(block *hcl.Block) (*Resource, hcl.Diagnostics) {
	r := &Resource{
		Mode:      addrs.ManagedResourceMode,
		Type:      block.Labels[0],
		Name:      block.Labels[1],
		DeclRange: block.DefRange,
		TypeRange: block.LabelRanges[0],
		Managed:   &ManagedResource{},
	}

	content, remain, diags := block.Body.PartialContent(resourceBlockSchema)
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
		r.Count = attr.Expr
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
								Severity: hcl.DiagWarning,
								Summary:  "Deprecated ignore_changes wildcard",
								Detail:   "The [\"*\"] form of ignore_changes wildcard is deprecated. Use \"ignore_changes = all\" to ignore changes to all attributes.",
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

			conn, connDiags := decodeConnectionBlock(block)
			diags = append(diags, connDiags...)
			r.Managed.Connection = conn

		case "provisioner":
			pv, pvDiags := decodeProvisionerBlock(block)
			diags = append(diags, pvDiags...)
			if pv != nil {
				r.Managed.Provisioners = append(r.Managed.Provisioners, pv)
			}

		default:
			// Should never happen, because the above cases should always be
			// exhaustive for all the types specified in our schema.
			continue
		}
	}

	return r, diags
}

func decodeDataBlock(block *hcl.Block) (*Resource, hcl.Diagnostics) {
	r := &Resource{
		Mode:      addrs.DataResourceMode,
		Type:      block.Labels[0],
		Name:      block.Labels[1],
		DeclRange: block.DefRange,
		TypeRange: block.LabelRanges[0],
	}

	content, remain, diags := block.Body.PartialContent(dataBlockSchema)
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
		r.Count = attr.Expr
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

	for _, block := range content.Blocks {
		// Our schema only allows for "lifecycle" blocks, so we can assume
		// that this is all we will see here. We don't have any lifecycle
		// attributes for data resources currently, so we'll just produce
		// an error.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported lifecycle block",
			Detail:   "Data resources do not have lifecycle settings, so a lifecycle block is not allowed.",
			Subject:  &block.DefRange,
		})
		break
	}

	return r, diags
}

type ProviderConfigRef struct {
	Name       string
	NameRange  hcl.Range
	Alias      string
	AliasRange *hcl.Range // nil if alias not set
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

	if len(traversal) < 1 && len(traversal) > 2 {
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

	ret := &ProviderConfigRef{
		Name:      traversal.RootName(),
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
func (r *ProviderConfigRef) Addr() addrs.ProviderConfig {
	return addrs.ProviderConfig{
		Type:  r.Name,
		Alias: r.Alias,
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
		{
			Type: "lifecycle",
		},
		{
			Type: "connection",
		},
		{
			Type:       "provisioner",
			LabelNames: []string{"type"},
		},
	},
}

var dataBlockSchema = &hcl.BodySchema{
	Attributes: commonResourceAttributes,
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "lifecycle",
		},
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
