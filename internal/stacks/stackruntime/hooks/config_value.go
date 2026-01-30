// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hooks

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

// ConfigValueHookData is the argument type for hook callbacks which
// signal configuration values becoming available during progressive resolution.
type ConfigValueHookData struct {
	// Addr is the string address of the configuration value (e.g., stack output)
	Addr string

	// Value is the computed cty.Value that became available
	Value cty.Value

	// ComponentInstance is the component instance address where this value originated
	ComponentInstance *stackaddrs.AbsComponentInstance

	// Phase indicates when this value became available ("pre-apply", "post-apply", etc.)
	Phase string
}
