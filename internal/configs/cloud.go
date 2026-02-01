// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/configs/definitions"
)

// CloudConfig is a type alias for the definition in the definitions package.
type CloudConfig = definitions.CloudConfig

func decodeCloudBlock(block *hcl.Block) (*CloudConfig, hcl.Diagnostics) {
	return &CloudConfig{
		Config:    block.Body,
		DeclRange: block.DefRange,
	}, nil
}

