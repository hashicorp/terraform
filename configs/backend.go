package configs

import (
	"github.com/hashicorp/hcl2/hcl"
)

// Backend represents a "backend" block inside a "terraform" block in a module
// or file.
type Backend struct {
	Type   string
	Config hcl.Body

	TypeRange hcl.Range
	DeclRange hcl.Range
}

func decodeBackendBlock(block *hcl.Block) (*Backend, hcl.Diagnostics) {
	return &Backend{
		Type:      block.Labels[0],
		TypeRange: block.LabelRanges[0],
		Config:    block.Body,
		DeclRange: block.DefRange,
	}, nil
}
