package langserver

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
)

func refAtPos(pos hcl.Pos, hclFile *hcl.File, cfgFile *configs.File) *addrs.Reference {
	// We'll use the low-level HCL file first to narrow down where we are
	// looking, and then we'll use the higher-level concepts in the
	// Terraform-specific file object for additional context if needed.

	block := hclFile.OutermostBlockAtPos(pos)
	if block == nil {
		return nil // not in a block at all
	}

	if block.DefRange.ContainsPos(pos) {
		// We're inside a block header, so we might be able to synthesize
		// a ref for the object being declared by it.
		return refForBlock(block)
	}

	return nil
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

	default:
		return nil
	}
}

func findDefinition(ref *addrs.Reference, hclFile *hcl.File, cfgFile *configs.File) hcl.Range {
	// This is a temporary implementation of findDefinition that works only
	// on a single file. It ought to instead take a *configs.Module and
	// look for possible resolutions across the whole module, but this is
	// just a proof-of-concept.

	switch addr := ref.Subject.(type) {

	case addrs.Resource:
		var rcs []*configs.Resource
		switch addr.Mode {
		case addrs.ManagedResourceMode:
			rcs = cfgFile.ManagedResources
		case addrs.DataResourceMode:
			rcs = cfgFile.DataResources
		}
		for _, rc := range rcs {
			if rc.Type == addr.Type && rc.Name == addr.Name {
				return rc.DeclRange
			}
		}
		return hcl.Range{}

	case addrs.InputVariable:
		for _, vc := range cfgFile.Variables {
			if vc.Name == addr.Name {
				return vc.DeclRange
			}
		}
		return hcl.Range{}

	case addrs.ModuleCall:
		for _, mc := range cfgFile.ModuleCalls {
			if mc.Name == addr.Name {
				return mc.DeclRange
			}
		}
		return hcl.Range{}

	default:
		return hcl.Range{}
	}
}
