// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeEphemeralHold is a graph node that passes the root module scope to the provided callback
// and then blocks until the returned channel is closed. This keeps the walk alive until the
// caller no longer needs the evaluation result of the walk, which is mostly useful for using ephemeral
// data that must be continually renewed and then closed. See [ephemeralResourceCloseTransformer] for
// more details on ordering.
type nodeEphemeralHold struct {
	callback func(scope *lang.Scope) <-chan struct{}
}

var (
	_ GraphNodeExecutable     = (*nodeEphemeralHold)(nil)
	_ GraphNodeModuleInstance = (*nodeEphemeralHold)(nil)
)

func (n *nodeEphemeralHold) Name() string {
	return "ephemeral resource hold"
}

func (n *nodeEphemeralHold) Path() addrs.ModuleInstance {
	return addrs.RootModuleInstance
}

func (n *nodeEphemeralHold) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	scope := ctx.EvaluationScope(nil, nil, EvalDataForNoInstanceKey)
	release := n.callback(scope)
	if release == nil {
		return nil
	}

	log.Printf("[TRACE] nodeEphemeralHold: holding the walk until released")
	select {
	case <-release:
	case <-ctx.StopCtx().Done():
		log.Printf("[TRACE] nodeEphemeralHold: stopped while holding the walk")
	}

	return nil
}
