// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// stacksInspector is the backing representation of a "stack inspector handle"
// as exposed in the stacks part of the RPC API, which allows a caller to
// provide what they want to inspect just once and then perform any number
// of subsequent inspection actions against it.
type stacksInspector struct {
	Config             *stackconfig.Config
	State              *stackstate.State
	ProviderFactories  map[addrs.Provider]providers.Factory
	InputValues        map[stackaddrs.InputVariable]stackruntime.ExternalInputValue
	ExperimentsAllowed bool
}

// InspectExpressionResult evaluates a given expression string in the
// inspection environment represented by the receiver.
func (i *stacksInspector) InspectExpressionResult(ctx context.Context, req *stacks.InspectExpressionResult_Request) (*stacks.InspectExpressionResult_Response, error) {
	var diags tfdiags.Diagnostics

	expr, hclDiags := hclsyntax.ParseExpression(req.ExpressionSrc, "<external expression>", hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return &stacks.InspectExpressionResult_Response{
			Diagnostics: diagnosticsToProto(diags),
		}, nil
	}

	stackAddr := stackaddrs.RootStackInstance
	if req.StackAddr != "" {
		// FIXME: Support this later. We don't currently have a stack instance
		// address parser to parse this input with, but we could build one
		// in future.
		return nil, status.Error(codes.InvalidArgument, "the InspectExpressionResult operation currently only supports evaluating in the topmost stack")
	}

	val, moreDiags := stackruntime.EvalExpr(ctx, expr, &stackruntime.EvalExprRequest{
		Config:             i.Config,
		State:              i.State,
		EvalStackInstance:  stackAddr,
		InputValues:        i.InputValues,
		ProviderFactories:  i.ProviderFactories,
		ExperimentsAllowed: i.ExperimentsAllowed,
	})
	diags = diags.Append(moreDiags)
	if val == cty.NilVal {
		// Too invalid to return any value at all, then.
		return &stacks.InspectExpressionResult_Response{
			Diagnostics: diagnosticsToProto(diags),
		}, nil
	}

	val, markses := val.UnmarkDeepWithPaths()
	sensitivePaths, otherMarkses := marks.PathsWithMark(markses, marks.Sensitive)
	if len(otherMarkses) != 0 {
		// Any other marks should've been dealt with by the stacks runtime
		// before getting here, since we only know how to preserve the sensitive
		// marking.
		return nil, fmt.Errorf(
			"%s: unhandled value marks %#v (this is a bug in Terraform)",
			tfdiags.FormatCtyPath(otherMarkses[0].Path), otherMarkses[0].Marks,
		)
	}
	valRaw, err := plans.NewDynamicValue(val, cty.DynamicPseudoType)
	if err != nil {
		// We might get here if the result was of a type we cannot send
		// over the wire, such as a reference to a provider configuration.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Result is not serializable",
			fmt.Sprintf("Cannot return the result of the given expression: %s.", err),
		))
		return &stacks.InspectExpressionResult_Response{
			Diagnostics: diagnosticsToProto(diags),
		}, nil
	}

	return &stacks.InspectExpressionResult_Response{
		Result:      stacks.NewDynamicValue(valRaw, sensitivePaths),
		Diagnostics: diagnosticsToProto(diags),
	}, nil
}
