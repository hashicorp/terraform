package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// OutputValueConfig represents an "output" block in a stack configuration.
type OutputValueConfig struct {
	addr   stackaddrs.ConfigOutputValue
	config *stackconfig.OutputValue

	main *Main

	validatedValue promising.Once[withDiagnostics[cty.Value]]
}

var _ Validatable = (*OutputValueConfig)(nil)
var _ namedPromiseReporter = (*OutputValueConfig)(nil)

func newOutputValueConfig(main *Main, addr stackaddrs.ConfigOutputValue, config *stackconfig.OutputValue) *OutputValueConfig {
	if config == nil {
		panic("newOutputValueConfig with nil configuration")
	}
	return &OutputValueConfig{
		addr:   addr,
		config: config,
		main:   main,
	}
}

func (ov *OutputValueConfig) Addr() stackaddrs.ConfigOutputValue {
	return ov.addr
}

func (ov *OutputValueConfig) Declaration(ctx context.Context) *stackconfig.OutputValue {
	return ov.config
}

func (ov *OutputValueConfig) tracingName() string {
	return ov.Addr().String()
}

// StackConfig returns the object representing the stack configuration that
// this output block belongs to.
func (ov *OutputValueConfig) StackConfig(ctx context.Context) *StackConfig {
	stackConfigAddr := ov.Addr().Stack
	return ov.main.StackConfig(ctx, stackConfigAddr)
}

// Value returns the result value for this output value that should be used
// for validating other objects that refer to this output value.
//
// If this output value is itself invalid then the result may be a
// compatibly-typed unknown placeholder value that's suitable for partial
// downstream validation.
func (ov *OutputValueConfig) Value(ctx context.Context) cty.Value {
	v, _ := ov.ValidateValue(ctx)
	return v
}

// ValueTypeConstraint returns the type that the final result of this output
// value is guaranteed to have.
func (ov *OutputValueConfig) ValueTypeConstraint(ctx context.Context) cty.Type {
	return ov.config.Type.Constraint
}

// ValidateValue validates that the value expression is evaluatable and that
// its result can convert to the declared type constraint, returning the
// resulting value.
//
// If the returned diagnostics has errors then the returned value might be
// just an approximation of the result, such as an unknown value with the
// declared type constraint.
func (ov *OutputValueConfig) ValidateValue(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
	return withCtyDynamicValPlaceholder(doOnceWithDiags(
		ctx, &ov.validatedValue, ov.main,
		ov.validateValueInner,
	))
}

// validateValueInner is the real implementation of ValidateValue, which runs
// in the background only once per instance of [OutputValueConfig] and then
// provides the result for all ValidateValue callers simultaneously.
func (ov *OutputValueConfig) validateValueInner(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	result, moreDiags := EvalExprAndEvalContext(ctx, ov.config.Value, ValidatePhase, ov.StackConfig(ctx))
	v := result.Value
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		v = cty.UnknownVal(ov.ValueTypeConstraint(ctx))
	}

	var err error
	v, err = convert.Convert(v, ov.config.Type.Constraint)
	if err != nil {
		v = cty.UnknownVal(ov.ValueTypeConstraint(ctx))
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid result for output value",
			Detail: fmt.Sprintf(
				"The result value does not match the declared type constraint: %s.",
				tfdiags.FormatError(err),
			),
			Subject:     ov.config.Value.Range().Ptr(),
			Expression:  result.Expression,
			EvalContext: result.EvalContext,
		})
	}

	return v, diags
}

// Validate implements Validatable.
func (ov *OutputValueConfig) Validate(ctx context.Context) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	_, moreDiags := ov.ValidateValue(ctx)
	diags = diags.Append(moreDiags)
	return diags
}

// reportNamedPromises implements namedPromiseReporter.
func (ov *OutputValueConfig) reportNamedPromises(report func(id promising.PromiseID, name string)) {
	report(ov.validatedValue.PromiseID(), ov.addr.String()+" value")
}
