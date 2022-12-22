package testconfigs

import (
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/moduletest/providermocks"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type MockProvider struct {
	Addr addrs.LocalProviderConfig

	DefDir string
	Config *providermocks.Config

	DeclRange hcl.Range
}

func decodeMockProviderBlock(block *hcl.Block) (*MockProvider, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &MockProvider{
		Addr: addrs.LocalProviderConfig{
			LocalName: block.Labels[0],
		},
		DeclRange: block.DefRange,
	}

	content, hclDiags := block.Body.Content(&mockProviderBlockSchema)
	diags = diags.Append(hclDiags)

	sourceFilename := block.DefRange.Filename
	sourceDir := filepath.Dir(sourceFilename)

	if attr, ok := content.Attributes["config"]; ok {
		hclDiags = gohcl.DecodeExpression(attr.Expr, nil, &ret.DefDir)
		diags = diags.Append(hclDiags)
		// The definition directory is resolved relative to the file
		// it's declared in.
		ret.DefDir = filepath.Join(sourceDir, ret.DefDir)

		// NOTE: We don't actually populate ret.Config here, because at
		// the time of decoding a mock provider block we only know the
		// local name of the provider. The main scenario decoder function
		// (our caller) must populate Config before returning this to
		// any external callers.
	}
	if attr, ok := content.Attributes["alias"]; ok {
		hclDiags = gohcl.DecodeExpression(attr.Expr, nil, &ret.Addr.Alias)
		diags = diags.Append(hclDiags)

		if !hclsyntax.ValidIdentifier(ret.Addr.Alias) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider alias",
				Detail:   "A provider alias name must be a valid identifier.",
				Subject:  attr.Expr.Range().Ptr(),
			})
		}
	}

	if !hclsyntax.ValidIdentifier(ret.Addr.LocalName) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider local name",
			Detail:   "A provider local name must be a valid identifier.",
			Subject:  block.LabelRanges[0].Ptr(),
		})
	}

	return ret, diags
}

var mockProviderBlockSchema = hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "config", Required: true},
		{Name: "alias"},
	},
}
