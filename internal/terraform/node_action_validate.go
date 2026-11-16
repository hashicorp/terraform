// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// NodeValidatableAction represents an action that is used for validation only.
type NodeValidatableAction struct {
	*NodeActionConfig
}

var (
	_ GraphNodeModuleInstance     = (*NodeValidatableAction)(nil)
	_ GraphNodeExecutable         = (*NodeValidatableAction)(nil)
	_ GraphNodeReferenceable      = (*NodeValidatableAction)(nil)
	_ GraphNodeReferencer         = (*NodeValidatableAction)(nil)
	_ GraphNodeConfigAction       = (*NodeValidatableAction)(nil)
	_ GraphNodeAttachActionSchema = (*NodeValidatableAction)(nil)
)

func (n *NodeValidatableAction) Path() addrs.ModuleInstance {
	// There is no expansion during validation, so we evaluate everything as
	// single module instances.
	return n.Addr.Module.UnkeyedInstanceShim()
}

func (n *NodeValidatableAction) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	return n.Validate(ctx, nil, cty.DynamicVal)
}
