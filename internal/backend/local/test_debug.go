// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/moduletest/graph"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// DebugContext is a context used for debugging tests in the local backend.
type DebugContext struct {
	Suite *moduletest.Suite

	// RunCh is the current run whose evaluation scope is where expressions will be evaluated.
	// This channel is used to communicate the results of the run back to the caller.
	RunCh chan *moduletest.Run

	ErrCh chan tfdiags.Diagnostics

	activeEvalContext *graph.EvalContext
}

func (ctx *DebugContext) Resume() {
	ctx.activeEvalContext.Pause(false)
}

func (ctx *DebugContext) EvalExpr(s *lang.Scope, expr hcl.Expression, wantType cty.Type) (cty.Value, tfdiags.Diagnostics) {
	return ctx.activeEvalContext.EvalExpr(s, expr, wantType)
}
