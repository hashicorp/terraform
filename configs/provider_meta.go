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
	// verify that the local name is already localized or produce an error.
	diags := checkProviderNameNormalized(block.Labels[0], block.DefRange)

	return &ProviderMeta{
		Provider:      block.Labels[0],
		ProviderRange: block.LabelRanges[0],
		Config:        block.Body,
		DeclRange:     block.DefRange,
	}, diags
}
