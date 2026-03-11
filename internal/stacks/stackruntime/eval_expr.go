// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// EvalExpr evaluates the given expression in a specified evaluation
// environment and scope.
//
// This is intended for situations like the "terraform console" command which
// need to evaluate arbitrary expressions against a configuration and
// previously-established state snapshot.
func EvalExpr(ctx context.Context, expr hcl.Expression, req *EvalExprRequest) (cty.Value, tfdiags.Diagnostics) {
	main := stackeval.NewForInspecting(req.Config, req.State, stackeval.InspectOpts{
		InputVariableValues: req.InputValues,
		ProviderFactories:   req.ProviderFactories,
	})
	main.AllowLanguageExperiments(req.ExperimentsAllowed)
	return main.EvalExpr(ctx, expr, req.EvalStackInstance, stackeval.InspectPhase)
}

// EvalExprRequest represents the inputs to an [EvalExpr] call.
type EvalExprRequest struct {
	// Config and State together provide the global environment in which
	// the expression will be evaluated.
	Config *stackconfig.Config
	State  *stackstate.State

	// EvalStackInstance is the address of the stack instance where the
	// expression is to be evaluated. If unspecified, the default is
	// to evaluate in the root stack instance.
	EvalStackInstance stackaddrs.StackInstance

	// InputValues and ProviderFactories are both optional extras to
	// provide a more complete evaluation environment, although neither
	// needs to be provided if the expression to be evaluated doesn't
	// (directly or indirectly) make use of input variables or provider
	// configurations corresponding to these.
	InputValues       map[stackaddrs.InputVariable]ExternalInputValue
	ProviderFactories map[addrs.Provider]providers.Factory

	ExperimentsAllowed bool
}
