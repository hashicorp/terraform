// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import "github.com/hashicorp/hcl/v2"

// ProviderMeta represents a "provider_meta" block inside a "terraform" block
// in a module or file.
type ProviderMeta struct {
	Provider string
	Config   hcl.Body

	ProviderRange hcl.Range
	DeclRange     hcl.Range
}

func decodeProviderMetaBlock(block *hcl.Block) (*ProviderMeta, hcl.Diagnostics) {
	// provider_meta must be a static map. We can verify this by attempting to
	// evaluate the values.
	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}

	for _, attr := range attrs {
		_, d := attr.Expr.Value(nil)
		diags = append(diags, d...)
	}

	// verify that the local name is already localized or produce an error.
	diags = append(diags, checkProviderNameNormalized(block.Labels[0], block.DefRange)...)

	return &ProviderMeta{
		Provider:      block.Labels[0],
		ProviderRange: block.LabelRanges[0],
		Config:        block.Body,
		DeclRange:     block.DefRange,
	}, diags
}
