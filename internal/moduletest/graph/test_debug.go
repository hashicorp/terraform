// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"strings"

	"github.com/google/go-dap"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// DebugContext is a context used for debugging tests in the local backend.
type DebugContext struct {
	Suite *moduletest.Suite

	// RunCh is the current run whose evaluation scope is where expressions will be evaluated.
	// This channel is used to communicate the results of the run back to the caller.
	RunCh chan *moduletest.Run

	ErrCh chan tfdiags.Diagnostics

	ActiveEvalContext *EvalContext

	ExecutionPoint string // The current execution point in the test run, e.g., "before", "after", etc.

	BeforeBreakpoints map[string]dap.Breakpoint
	Breakpoints       map[string]dap.Breakpoint
}

func (ctx *DebugContext) Resume() {
	ctx.ActiveEvalContext.Pause(false)
}

func (ctx *DebugContext) Next(prev *moduletest.Run) {
	// Set the "after" breakpoint for the next run in the file.
	fileName := ctx.ActiveEvalContext.File.Name
	runs := ctx.Suite.Files[fileName].Runs
	if prev.Index+1 < len(runs) {
		runs[prev.Index+1].SetBreakPoint("after")
		ctx.ExecutionPoint = "after"
		ctx.Resume()
	}
}

func (ctx *DebugContext) Break(scope *lang.Scope, expr hcl.Expression) (runName string, diags tfdiags.Diagnostics) {
	run, sDiags := scope.EvalExpr(expr, cty.DynamicPseudoType)
	diags = diags.Append(sDiags)
	if diags.HasErrors() {
		return "", diags
	}
	if run.IsNull() {
		return "", diags
	}
	runName = run.AsString()

	fileName := ctx.ActiveEvalContext.File.Name
	runs := ctx.Suite.Files[fileName].Runs
	var found bool
	for _, r := range runs {
		if r.Name == runName {
			found = true
			r.SetBreakPoint("after")
			break
		}
	}

	// no run with that name was found
	if !found {
		return "", diags
	}

	return runName, diags
}

func (ctx *DebugContext) breakFunc() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name:      "run",
				Type:      cty.String,
				AllowNull: true,
			},
		},
		VarParam: &function.Parameter{
			Name:             "condition",
			Type:             cty.Bool,
			AllowNull:        true,
			AllowDynamicType: true,
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			if len(args) < 1 || len(args) > 2 {
				return cty.NullVal(cty.String), fmt.Errorf("break() expects 1 or 2 arguments, got %d", len(args))
			}

			run := args[0].AsString()
			runName := strings.TrimPrefix(run, "run.")

			if !strings.HasPrefix(run, "run.") {
				return cty.NullVal(cty.String), function.NewArgErrorf(0, "expected run name to start with 'run.', got '%s'", run)
			}

			if len(args) == 1 {
				return cty.StringVal(runName), nil
			}
			condition := args[1]
			if condition.True() {
				return cty.StringVal(runName), nil
			}

			return cty.NullVal(cty.String), nil
		},
	})
}

func (ctx *DebugContext) EvalExpr(s *lang.Scope, expr hcl.Expression, wantType cty.Type) (cty.Value, tfdiags.Diagnostics) {
	ctx.parseRef(s)
	return ctx.ActiveEvalContext.EvalExpr(s, expr, wantType)
}

func (ctx *DebugContext) parseRef(s *lang.Scope) {
	prevParseRef := s.ParseRef
	parseObjectRef := func(traversal hcl.Traversal) (refs *addrs.Reference, diags tfdiags.Diagnostics) {
		root := traversal.RootName()
		// allow accessing run blocks and variables directly
		if len(traversal) == 1 {
			switch root {
			case "run":
				ret := &addrs.Reference{
					Subject:     addrs.Run{},
					SourceRange: tfdiags.SourceRangeFromHCL(traversal[0].SourceRange()),
				}
				return ret, diags
			case "var":
				ret := &addrs.Reference{
					Subject:     addrs.InputVariable{},
					SourceRange: tfdiags.SourceRangeFromHCL(traversal[0].SourceRange()),
				}
				return ret, diags
			}
		}
		return prevParseRef(traversal)
	}
	s.ParseRef = parseObjectRef
}

func (ctx *DebugContext) AddBreakpoint(br dap.Breakpoint) tfdiags.Diagnostics {
	file := ctx.Suite.Files[br.Source.Name]
	for _, run := range file.Runs {
		if run.Config.DeclRange.Start.Line-1 == br.Line {
			ctx.BeforeBreakpoints[run.Name] = br
			return nil
		} else if run.Config.DeclRange.Start.Line == br.Line {
			ctx.Breakpoints[run.Name] = br
			return nil
		}
	}
	return nil
}
