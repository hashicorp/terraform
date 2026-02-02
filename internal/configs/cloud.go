// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/configs/definitions"
)

func decodeCloudBlock(block *hcl.Block) (*definitions.CloudConfig, hcl.Diagnostics) {
	return &definitions.CloudConfig{
		Config:    block.Body,
		DeclRange: block.DefRange,
	}, nil
}

