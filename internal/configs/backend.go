// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/configs/definitions"
)

func decodeBackendBlock(block *hcl.Block) (*definitions.Backend, hcl.Diagnostics) {
	return &definitions.Backend{
		Type:      block.Labels[0],
		TypeRange: block.LabelRanges[0],
		Config:    block.Body,
		DeclRange: block.DefRange,
	}, nil
}
