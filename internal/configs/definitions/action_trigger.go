// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"
)

// ActionTrigger represents a configured "action_trigger" inside the lifecycle
// block of a managed resource.
type ActionTrigger struct {
	Condition hcl.Expression
	Events    []ActionTriggerEvent
	Actions   []ActionRef // References to actions

	DeclRange hcl.Range
}
