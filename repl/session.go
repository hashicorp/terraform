package repl

import (
	"errors"
	"fmt"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/terraform/configs/hcl2shim"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/tfdiags"
)

// ErrSessionExit is a special error result that should be checked for
// from Handle to signal a graceful exit.
var ErrSessionExit = errors.New("session exit")

// Session represents the state for a single REPL session.
type Session struct {
	// Scope is the evaluation scope where expressions will be evaluated.
	Scope *lang.Scope
}

// Handle handles a single line of input from the REPL.
//
// This is a stateful operation if a command is given (such as setting
// a variable). This function should not be called in parallel.
//
// The return value is the output and the error to show.
func (s *Session) Handle(line string) (string, bool, tfdiags.Diagnostics) {
	switch {
	case strings.TrimSpace(line) == "":
		return "", false, nil
	case strings.TrimSpace(line) == "exit":
		return "", true, nil
	case strings.TrimSpace(line) == "help":
		ret, diags := s.handleHelp()
		return ret, false, diags
	default:
		ret, diags := s.handleEval(line)
		return ret, false, diags
	}
}

func (s *Session) handleEval(line string) (string, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Parse the given line as an expression
	expr, parseDiags := hclsyntax.ParseExpression([]byte(line), "<console-input>", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return "", diags
	}

	val, valDiags := s.Scope.EvalExpr(expr, cty.DynamicPseudoType)
	diags = diags.Append(valDiags)
	if valDiags.HasErrors() {
		return "", diags
	}

	if !val.IsWhollyKnown() {
		// FIXME: In future, once we've updated the result formatter to be
		// cty-aware, we should just include unknown values as "(not yet known)"
		// in the serialized result, allowing the rest (if any) to be seen.
		diags = diags.Append(fmt.Errorf("Result depends on values that cannot be determined until after \"terraform apply\"."))
		return "", diags
	}

	// Our formatter still wants an old-style raw interface{} value, so
	// for now we'll just shim it.
	// FIXME: Port the formatter to work with cty.Value directly.
	legacyVal := hcl2shim.ConfigValueFromHCL2(val)
	result, err := FormatResult(legacyVal)
	if err != nil {
		diags = diags.Append(err)
		return "", diags
	}

	return result, diags
}

func (s *Session) handleHelp() (string, tfdiags.Diagnostics) {
	text := `
The Terraform console allows you to experiment with Terraform interpolations.
You may access resources in the state (if you have one) just as you would
from a configuration. For example: "aws_instance.foo.id" would evaluate
to the ID of "aws_instance.foo" if it exists in your state.

Type in the interpolation to test and hit <enter> to see the result.

To exit the console, type "exit" and hit <enter>, or use Control-C or
Control-D.
`

	return strings.TrimSpace(text), nil
}
