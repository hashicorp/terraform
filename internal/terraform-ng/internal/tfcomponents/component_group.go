package tfcomponents

import (
	"fmt"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/terraform-ng/internal/ngaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ComponentGroup struct {
	Name string

	SourceAddr      addrs.ModuleSource
	SourceAddrRaw   string
	SourceAddrRange tfdiags.SourceRange
	// TODO: Also need to do something about module package versions.
	// A "version" attribute here would be consistent with "module"
	// blocks in Terraform, but maybe authors would prefer something
	// a bit more centralized than each group having its own package
	// version.

	DisplayName hcl.Expression
	Variables   hcl.Expression
	ForEach     hcl.Expression

	DeclRange tfdiags.SourceRange
}

func (c *ComponentGroup) CallAddr() ngaddrs.ComponentGroupCall {
	return ngaddrs.ComponentGroupCall{Name: c.Name}
}

func decodeComponentGroupBlock(block *hcl.Block) (*ComponentGroup, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &ComponentGroup{
		Name:      block.Labels[0],
		DeclRange: tfdiags.SourceRangeFromHCL(block.DefRange),
	}
	if !hclsyntax.ValidIdentifier(block.Labels[0]) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid component group name",
			Detail:   fmt.Sprintf("Cannot use %q as a component group name: must be a valid identifier.", block.Labels[0]),
			Subject:  block.LabelRanges[0].Ptr(),
		})
	}

	content, hclDiags := block.Body.Content(componentGroupBlockSchema)
	diags = diags.Append(hclDiags)

	if attr, ok := content.Attributes["components"]; ok {
		ret.SourceAddrRange = tfdiags.SourceRangeFromHCL(attr.Expr.Range())

		hclDiags := gohcl.DecodeExpression(attr.Expr, nil, &ret.SourceAddrRaw)
		diags = diags.Append(hclDiags)
		if !hclDiags.HasErrors() {
			addr, err := addrs.ParseModuleSource(ret.SourceAddrRaw)
			if err == nil {
				ret.SourceAddr = addr
				if !(strings.HasSuffix(ret.SourceAddrRaw, ".tfcomponents.hcl") || strings.HasSuffix(ret.SourceAddrRaw, "/tfcomponents.hcl")) {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid component group configuration address",
						Detail:   fmt.Sprintf("Failed to parse component group configuration address: selected file must have suffix \".tfcomponents.hcl\"."),
						Subject:  attr.Expr.Range().Ptr(),
					})
				}
			} else {
				// NOTE: We intentionally leave SourceAddr unset here, so that
				// a caller trying to carefully analyze a partially-invalid
				// result can clearly see that the source address wass invalid.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid component group configuration address",
					Detail:   fmt.Sprintf("Failed to parse component group configuration address: %s.", err),
					Subject:  attr.Expr.Range().Ptr(),
				})
			}
		}
	}
	if attr, ok := content.Attributes["display_name"]; ok {
		ret.DisplayName = attr.Expr
	}
	if attr, ok := content.Attributes["for_each"]; ok {
		ret.ForEach = attr.Expr
	}
	if attr, ok := content.Attributes["variables"]; ok {
		ret.Variables = attr.Expr
	}

	return ret, diags
}

var componentGroupBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "components", Required: true},
		{Name: "display_name", Required: true},
		{Name: "for_each"},
		{Name: "variables"},
	},
}
