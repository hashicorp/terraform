// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"github.com/hashicorp/hcl/v2"
)

// StateStorage represents a "state_store" block inside a "terraform" block
// in a module or file.
type StateStorage struct {
	Type   string
	Config hcl.Body

	ProviderConfigRef *ProviderConfigRef

	TypeRange hcl.Range
	DeclRange hcl.Range
}

func decodeStateStorageBlock(block *hcl.Block) (*StateStorage, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ss := &StateStorage{
		Type:      block.Labels[0],
		TypeRange: block.LabelRanges[0],
		Config:    block.Body,
		DeclRange: block.DefRange,
	}

	content, remain, moreDiags := block.Body.PartialContent(ResourceBlockSchema)
	diags = append(diags, moreDiags...)
	ss.Config = remain

	if attr, exists := content.Attributes["provider"]; exists {
		var providerDiags hcl.Diagnostics
		ss.ProviderConfigRef, providerDiags = decodeProviderConfigRef(attr.Expr, "provider")
		diags = append(diags, providerDiags...)
	}

	return ss, diags
}

var StateStorageBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "provider",
		},
	},
}
