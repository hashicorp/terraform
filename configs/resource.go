package configs

import (
	"fmt"

	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
)

// ManagedResource represents a "resource" block in a module or file.
type ManagedResource struct {
	Name    string
	Type    string
	Config  hcl.Body
	Count   hcl.Expression
	ForEach hcl.Expression

	ProviderConfigRef *ProviderConfigRef

	DependsOn []hcl.Traversal

	Connection   *Connection
	Provisioners []*Provisioner

	CreateBeforeDestroy bool
	PreventDestroy      bool
	IgnoreChanges       []hcl.Traversal

	CreateBeforeDestroySet bool
	PreventDestroySet      bool

	DeclRange hcl.Range
	TypeRange hcl.Range
}

func (r *ManagedResource) moduleUniqueKey() string {
	return fmt.Sprintf("%s.%s", r.Name, r.Type)
}

func decodeResourceBlock(block *hcl.Block) (*ManagedResource, hcl.Diagnostics) {
	r := &ManagedResource{
		Type:      block.Labels[0],
		Name:      block.Labels[1],
		DeclRange: block.DefRange,
		TypeRange: block.LabelRanges[0],
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
			Subject:  &block.LabelRanges[0],
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
		r.ProviderConfigRef, providerDiags = decodeProviderConfigRef(attr)
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
				valDiags := gohcl.DecodeExpression(attr.Expr, nil, &r.CreateBeforeDestroy)
				diags = append(diags, valDiags...)
				r.CreateBeforeDestroySet = true
			}

			if attr, exists := lcContent.Attributes["prevent_destroy"]; exists {
				valDiags := gohcl.DecodeExpression(attr.Expr, nil, &r.PreventDestroy)
				diags = append(diags, valDiags...)
				r.PreventDestroySet = true
			}

			if attr, exists := lcContent.Attributes["ignore_changes"]; exists {
				exprs, listDiags := hcl.ExprList(attr.Expr)
				diags = append(diags, listDiags...)

				for _, expr := range exprs {
					traversal, travDiags := hcl.RelTraversalForExpr(expr)
					diags = append(diags, travDiags...)
					if len(traversal) != 0 {
						r.IgnoreChanges = append(r.IgnoreChanges, traversal)
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
			r.Connection = conn

		case "provisioner":
			pv, pvDiags := decodeProvisionerBlock(block)
			diags = append(diags, pvDiags...)
			if pv != nil {
				r.Provisioners = append(r.Provisioners, pv)
			}

		default:
			// Should never happen, because the above cases should always be
			// exhaustive for all the types specified in our schema.
			continue
		}
	}

	return r, diags
}

// DataResource represents a "data" block in a module or file.
type DataResource struct {
	Name    string
	Type    string
	Config  hcl.Body
	Count   hcl.Expression
	ForEach hcl.Expression

	ProviderConfigRef *ProviderConfigRef

	DependsOn []hcl.Traversal

	DeclRange hcl.Range
	TypeRange hcl.Range
}

func (r *DataResource) moduleUniqueKey() string {
	return fmt.Sprintf("data.%s.%s", r.Name, r.Type)
}

func decodeDataBlock(block *hcl.Block) (*DataResource, hcl.Diagnostics) {
	r := &DataResource{
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
			Subject:  &block.LabelRanges[0],
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
		r.ProviderConfigRef, providerDiags = decodeProviderConfigRef(attr)
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

func decodeProviderConfigRef(attr *hcl.Attribute) (*ProviderConfigRef, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	traversal, travDiags := hcl.AbsTraversalForExpr(attr.Expr)

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
		if exprIsNativeQuotedString(attr.Expr) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider configuration reference",
				Detail:   "A provider configuration reference must not be given in quotes.",
				Subject:  attr.Expr.Range().Ptr(),
			})
			return nil, diags
		}

		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider configuration reference",
			Detail:   fmt.Sprintf("The %s argument requires a provider type name, optionally followed by a period and then a configuration alias.", attr.Name),
			Subject:  attr.Expr.Range().Ptr(),
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
