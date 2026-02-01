// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/configs/definitions"
)

// Backend is a type alias for the definition in the definitions package.
type Backend = definitions.Backend

func decodeBackendBlock(block *hcl.Block) (*Backend, hcl.Diagnostics) {
	return &Backend{
		Type:      block.Labels[0],
		TypeRange: block.LabelRanges[0],
		Config:    block.Body,
		DeclRange: block.DefRange,
	}, nil
}
