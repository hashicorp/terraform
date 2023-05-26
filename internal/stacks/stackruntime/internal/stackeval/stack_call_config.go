package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// StackCallConfig represents a "stack" block in a stack configuration,
// representing a call to an embedded stack.
type StackCallConfig struct {
	addr   stackaddrs.ConfigStackCall
	config *stackconfig.EmbeddedStack

	main *Main

	inputVariableValues promising.Once[withDiagnostics[map[stackaddrs.InputVariable]cty.Value]]
}

var _ Validatable = (*InputVariableConfig)(nil)
var _ ExpressionScope = (*StackCallConfig)(nil)

func newStackCallConfig(main *Main, addr stackaddrs.ConfigStackCall, config *stackconfig.EmbeddedStack) *StackCallConfig {
	return &StackCallConfig{
		addr:   addr,
		config: config,
		main:   main,
	}
}

func (s *StackCallConfig) Addr() stackaddrs.ConfigStackCall {
	return s.addr
}

// CallerConfig returns the object representing the stack configuration that this
// stack call was declared within.
func (s *StackCallConfig) CallerConfig(ctx context.Context) *StackConfig {
	return s.main.mustStackConfig(ctx, s.Addr().Stack)
}

// CalleeConfig returns the object representing the child stack configuration
// that this stack call is referring to.
func (s *StackCallConfig) CalleeConfig(ctx context.Context) *StackConfig {
	return s.main.mustStackConfig(ctx, s.Addr().Stack.Child(s.addr.Item.Name))
}

// ValidateInputVariableValues evaluates the "inputs" argument inside the
// configuration block, ensure that it's valid per the expectations of the
// child stack config, and then returns the resulting values.
//
// A [StackCallConfig] represents the not-yet-expanded stack call, so the
// result is an approximation of the input variables for all instances of
// this call. To get the final values for a particular instance, use
// [StackCall.InputVariableValues] instead.
//
// If the returned diagnostics contains errors then the returned values may
// be incomplete, but should at least be of the types specified in the
// variable declarations.
func (s *StackCallConfig) ValidateInputVariableValues(ctx context.Context) (map[stackaddrs.InputVariable]cty.Value, tfdiags.Diagnostics) {
	// FIXME: This once-handling is pretty awkward when there are diagnostics
	// involved. Can we find a better way to handle this common situation?
	ret, err := s.inputVariableValues.Do(ctx, func(ctx context.Context) (withDiagnostics[map[stackaddrs.InputVariable]cty.Value], error) {
		ret, diags := func() (map[stackaddrs.InputVariable]cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics
			callee := s.CalleeConfig(ctx)
			vars := callee.InputVariables(ctx)

			atys := make(map[string]cty.Type, len(vars))
			var optional []string
			defs := make(map[string]cty.Value, len(vars))
			for addr, v := range vars {
				aty := v.TypeConstraint()

				atys[addr.Name] = aty
				if def := v.DefaultValue(ctx); def != cty.NilVal {
					optional = append(optional, addr.Name)
					defs[addr.Name] = def
				}
			}

			oty := cty.ObjectWithOptionalAttrs(atys, optional)

			varsObj, moreDiags := EvalExpr(ctx, s.config.Inputs, ValidatePhase, s)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				varsObj = cty.UnknownVal(oty.WithoutOptionalAttributesDeep())
			}

			// FIXME: TODO: We need to apply the nested optional attribute defaults
			// somewhere in here too, but it isn't clear where we should do that since
			// we're supposed to do that before type conversion but we don't yet have
			// the isolated variable values to apply the defaults to.

			varsObj, err := convert.Convert(varsObj, oty)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid input variable definitions",
					Detail: fmt.Sprintf(
						"Unsuitable input variable definitions: %s.",
						tfdiags.FormatError(err),
					),
					Subject: s.config.Inputs.Range().Ptr(),
				})
				varsObj = cty.UnknownVal(oty.WithoutOptionalAttributesDeep())
			}

			ret := make(map[stackaddrs.InputVariable]cty.Value, len(vars))

			for addr := range vars {
				val := varsObj.GetAttr(addr.Name)
				if val.IsNull() {
					if def, ok := defs[addr.Name]; ok {
						ret[addr] = def
					} else {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Missing definition for required input variable",
							Detail:   fmt.Sprintf("The input variable %q is required, so cannot be omitted.", addr.Name),
							Subject:  s.config.Inputs.Range().Ptr(),
						})
						ret[addr] = cty.UnknownVal(atys[addr.Name])
					}
				} else {
					ret[addr] = val
				}
			}

			return ret, diags
		}()
		return withDiagnostics[map[stackaddrs.InputVariable]cty.Value]{
			Result:      ret,
			Diagnostics: diags,
		}, nil
	})
	if err != nil {
		// TODO: A better error message for promise resolution failures.
		ret.Diagnostics = ret.Diagnostics.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to evaluate input variable definitions",
			Detail:   fmt.Sprintf("Could not evaluate the input variable definitions for this call: %s.", err),
			Subject:  s.config.DeclRange.ToHCL().Ptr(),
		})
	}
	return ret.Result, ret.Diagnostics
}

// InputVariableValues returns the effective input variable values specified
// in this call, or correctly-typed placeholders if any values are invalid.
//
// This is intended to support downstream evaluation of other objects during
// the validate phase, rather than for direct validation of this object. If you
// are intending to report problems directly to the user, use
// [StackCallConfig.ValidateInputVariableValues] instead.
func (s *StackCallConfig) InputVariableValues(ctx context.Context) map[stackaddrs.InputVariable]cty.Value {
	ret, _ := s.ValidateInputVariableValues(ctx)
	return ret
}

// ResolveExpressionReference implements ExpressionScope for evaluating
// expressions within a "stack" block during the validation phase.
//
// Note that the "stack" block lives in the caller scope rather than the
// callee scope, so this scope is not appropriate for evaluating anything
// inside the child variable declarations: they belong to the callee
// scope.
//
// This scope produces an approximation of expression results that is true
// for all instances of the stack call, not final results for a specific
// instance of a stack call. This is not the right scope to use during the
// plan and apply phases.
func (s *StackCallConfig) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	repetition := instances.RepetitionData{}
	if s.config.ForEach != nil {
		// We're producing an approximation across all eventual instances
		// of this call, so we'll set each.key and each.value to unknown
		// values.
		repetition.EachKey = cty.UnknownVal(cty.String).RefineNotNull()
		repetition.EachValue = cty.DynamicVal
	}
	return s.main.
		mustStackConfig(ctx, s.Addr().Stack).
		resolveExpressionReference(ctx, ref, instances.RepetitionData{}, nil)
}

// Validate implements Validatable
func (s *StackCallConfig) Validate(ctx context.Context) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	diags = diags.Append(
		s.ValidateInputVariableValues(ctx),
	)
	return diags
}
