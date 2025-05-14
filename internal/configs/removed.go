// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

// Removed describes the contents of a "removed" block in configuration.
type Removed struct {
	// From is the address of the configuration object being removed.
	From *addrs.RemoveTarget

	// Destroy indicates that the resource should be destroyed, not just removed
	// from state. Defaults to true.
	Destroy bool

	// Managed captures a number of metadata fields that are applicable only
	// for managed resources, and not for other resource modes.
	//
	// "removed" blocks support only a subset of the fields in [ManagedResource].
	Managed *ManagedResource

	DeclRange hcl.Range
}

func decodeRemovedBlock(block *hcl.Block) (*Removed, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	removed := &Removed{
		DeclRange: block.DefRange,
	}

	content, moreDiags := block.Body.Content(removedBlockSchema)
	diags = append(diags, moreDiags...)

	var targetKind addrs.RemoveTargetKind
	var resourceMode addrs.ResourceMode // only valid if targetKind is addrs.RemoveTargetResource
	if attr, exists := content.Attributes["from"]; exists {
		from, traversalDiags := hcl.AbsTraversalForExpr(attr.Expr)
		diags = append(diags, traversalDiags...)
		if !traversalDiags.HasErrors() {
			from, fromDiags := addrs.ParseRemoveTarget(from)
			diags = append(diags, fromDiags.ToHCL()...)
			removed.From = from
			if removed.From != nil {
				targetKind = removed.From.ObjectKind()
				if targetKind == addrs.RemoveTargetResource {
					resourceMode = removed.From.RelSubject.(addrs.ConfigResource).Resource.Mode
				}
			}
		}
	}

	removed.Destroy = true
	if resourceMode == addrs.ManagedResourceMode {
		removed.Managed = &ManagedResource{}
	}

	var seenConnection *hcl.Block
	for _, block := range content.Blocks {
		switch block.Type {
		case "lifecycle":
			lcContent, lcDiags := block.Body.Content(removedLifecycleBlockSchema)
			diags = append(diags, lcDiags...)

			if attr, exists := lcContent.Attributes["destroy"]; exists {
				valDiags := gohcl.DecodeExpression(attr.Expr, nil, &removed.Destroy)
				diags = append(diags, valDiags...)
			}

		case "connection":
			if removed.Managed == nil {
				// target is not a managed resource, then
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid connection block",
					Detail:   "Provisioner connection configuration is valid only when a removed block targets a managed resource.",
					Subject:  &block.DefRange,
				})
				continue
			}

			if seenConnection != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate connection block",
					Detail:   fmt.Sprintf("This \"removed\" block already has a connection block at %s.", seenConnection.DefRange),
					Subject:  &block.DefRange,
				})
				continue
			}
			seenConnection = block

			removed.Managed.Connection = &Connection{
				Config:    block.Body,
				DeclRange: block.DefRange,
			}

		case "provisioner":
			if removed.Managed == nil {
				// target is not a managed resource, then
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid provisioner block",
					Detail:   "Provisioners are valid only when a removed block targets a managed resource.",
					Subject:  &block.DefRange,
				})
				continue
			}

			pv, pvDiags := decodeProvisionerBlock(block)
			diags = append(diags, pvDiags...)
			if pv != nil {
				removed.Managed.Provisioners = append(removed.Managed.Provisioners, pv)

				if pv.When != ProvisionerWhenDestroy {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid provisioner block",
						Detail:   "Only destroy-time provisioners are valid in \"removed\" blocks. To declare a destroy-time provisioner, use:\n    when = destroy",
						Subject:  &block.DefRange,
					})
				}
			}
		}
	}

	return removed, diags
}

var removedBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "from",
			Required: true,
		},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "lifecycle"},
		{Type: "connection"},
		{Type: "provisioner", LabelNames: []string{"type"}},
	},
}

var removedLifecycleBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "destroy",
		},
	},
}
