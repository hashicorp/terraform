// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"
)

// CloudConfig represents a "cloud" block inside a "terraform" block in a module
// or file.
type CloudConfig struct {
	Config hcl.Body

	DeclRange hcl.Range
}

// ToBackendConfig converts the CloudConfig to a Backend struct with type "cloud".
func (c *CloudConfig) ToBackendConfig() Backend {
	return Backend{
		Type:   "cloud",
		Config: c.Config,
	}
}
