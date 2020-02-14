package langserver

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
)

func refAtPos(pos hcl.Pos, hclFile *hcl.File) *addrs.Reference {
	block := hclFile.OutermostBlockAtPos(pos)
	if block == nil {
		return nil
	}

	return refForBlock(block)
}

func refForBlock(block *hcl.Block) *addrs.Reference {
	rng := tfdiags.SourceRangeFromHCL(block.DefRange)
	labels := block.Labels

	switch block.Type {
	case "resource", "data":
		if len(labels) != 2 {
			return nil // invalid labels
		}
		mode := addrs.ManagedResourceMode
		if block.Type == "data" {
			mode = addrs.DataResourceMode
		}
		return &addrs.Reference{
			Subject: addrs.Resource{
				Mode: mode,
				Type: labels[0],
				Name: labels[1],
			},
			SourceRange: rng,
		}

	case "provider":
		if len(labels) != 1 {
			return nil // invalid labels
		}
		provider := addrs.NewLegacyProvider(labels[0])

		// TODO: new-style provider

		findAliasInBody := func(body hcl.Body) string {
			attrs, diags := body.JustAttributes()
			if diags.HasErrors() {
				return ""
			}

			v, ok := attrs["alias"]
			if !ok {
				return ""
			}

			val, diags := v.Expr.Value(nil)
			if diags.HasErrors() {
				return ""
			}
			return val.AsString()
		}

		provider.Alias = findAliasInBody(block.Body)
		return &addrs.Reference{
			Subject:     provider,
			SourceRange: rng,
		}

	case "variable":
		if len(labels) != 1 {
			return nil // invalid labels
		}
		return &addrs.Reference{
			Subject: addrs.InputVariable{
				Name: labels[0],
			},
			SourceRange: rng,
		}

	case "module":
		if len(labels) != 1 {
			return nil // invalid labels
		}
		return &addrs.Reference{
			Subject: addrs.ModuleCall{
				Name: labels[0],
			},
			SourceRange: rng,
		}
	}

	return nil
}
