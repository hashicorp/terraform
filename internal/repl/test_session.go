// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package repl

import (
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/lang/types"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestSession represents the state for a single REPL session.
type TestSession struct {
	// Scope is the evaluation scope where expressions will be evaluated.
	Scope   *lang.Scope
	Current *moduletest.Run

	// Handlers is a map of command names to functions that handle
	// those commands. If a command is not found in this map, the
	// expression will be evaluated as a normal expression.
	Handlers map[string]HandleFunc

	// Evaluator is the evaluator used to evaluate expressions.
	Evaluator Evaluator
}

type Evaluator interface {
	EvalExpr(s *lang.Scope, expr hcl.Expression, wantType cty.Type) (cty.Value, tfdiags.Diagnostics)
}

// func (s *TestSession) Code() string {
// 	// if s.Current == nil {
// 	// 	return ""
// 	// }
// 	// return s.Current.Code
// }

// Handle handles a single line of input from the REPL.
//
// This is a stateful operation if a command is given (such as setting
// a variable). This function should not be called in parallel.
//
// The return value is the output and the error to show.
func (s *TestSession) Handle(line string) (ret string, exit bool, diags tfdiags.Diagnostics) {
	if handler := s.Handlers[strings.TrimSpace(line)]; handler != nil {
		ret, exit, diags = handler(line)
		return
	}
	switch {
	case strings.TrimSpace(line) == "":
		return "", false, nil
	case strings.TrimSpace(line) == "exit":
		return "", true, nil
	case strings.TrimSpace(line) == "help":
		ret, diags := s.handleHelp()
		return ret, false, diags
	default:
		ret, diags = s.handleEval(line)
		return ret, false, diags
	}
}

func (s *TestSession) handleEval(line string) (string, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Parse the given line as an expression
	expr, parseDiags := hclsyntax.ParseExpression([]byte(line), "<console-input>", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return "", diags
	}

	val, valDiags := s.Evaluator.EvalExpr(s.Scope, expr, cty.DynamicPseudoType)
	diags = diags.Append(valDiags)
	if valDiags.HasErrors() {
		return "", diags
	}

	// The TypeType mark is used only by the console-only `type` function, in
	// order to smuggle the type of a given value back here. We can then
	// display a representation of the type directly.
	if marks.Contains(val, marks.TypeType) {
		val, _ = val.UnmarkDeep()

		valType := val.Type()
		switch {
		case valType.Equals(types.TypeType):
			// An encapsulated type value, which should be displayed directly.
			valType := val.EncapsulatedValue().(*cty.Type)
			return typeString(*valType), diags
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid use of type function",
				"The console-only \"type\" function cannot be used as part of an expression.",
			))
			return "", diags
		}
	}

	return FormatValue(val, 0), diags
}

func (s *TestSession) handleHelp() (string, tfdiags.Diagnostics) {
	text := `
The Terraform test console allows you to experiment with Terraform interpolations.
You may access runs in the test suite just as you would
from a configuration. For example: "run.foo.id" would evaluate
to the Output ID of run "run.foo".

Type in the interpolation to test and hit <enter> to see the result.

To exit the console, type "exit" and hit <enter>, or use Control-C or
Control-D.
`

	return strings.TrimSpace(text), nil
}
