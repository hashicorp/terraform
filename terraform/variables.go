package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
)

// InputValue represents a value for a variable in the root module, provided
// as part of the definition of an operation.
type InputValue struct {
	Value      cty.Value
	SourceType ValueSourceType

	// SourceRange provides source location information for values whose
	// SourceType is either ValueFromConfig or ValueFromFile. It is not
	// populated for other source types, and so should not be used.
	SourceRange tfdiags.SourceRange
}

// ValueSourceType describes what broad category of source location provided
// a particular value.
type ValueSourceType rune

const (
	// ValueFromUnknown is the zero value of ValueSourceType and is not valid.
	ValueFromUnknown ValueSourceType = 0

	// ValueFromConfig indicates that a value came from a .tf or .tf.json file,
	// e.g. the default value defined for a variable.
	ValueFromConfig ValueSourceType = 'C'

	// ValueFromAutoFile indicates that a value came from a "values file", like
	// a .tfvars file, that was implicitly loaded by naming convention.
	ValueFromAutoFile ValueSourceType = 'F'

	// ValueFromNamedFile indicates that a value came from a named "values file",
	// like a .tfvars file, that was passed explicitly on the command line (e.g.
	// -var-file=foo.tfvars).
	ValueFromNamedFile ValueSourceType = 'N'

	// ValueFromCLIArg indicates that the value was provided directly in
	// a CLI argument. The name of this argument is not recorded and so it must
	// be inferred from context.
	ValueFromCLIArg ValueSourceType = 'A'

	// ValueFromEnvVar indicates that the value was provided via an environment
	// variable. The name of the variable is not recorded and so it must be
	// inferred from context.
	ValueFromEnvVar ValueSourceType = 'E'

	// ValueFromInput indicates that the value was provided at an interactive
	// input prompt.
	ValueFromInput ValueSourceType = 'I'

	// ValueFromPlan indicates that the value was retrieved from a stored plan.
	ValueFromPlan ValueSourceType = 'P'

	// ValueFromCaller indicates that the value was explicitly overridden by
	// a caller to Context.SetVariable after the context was constructed.
	ValueFromCaller ValueSourceType = 'S'
)

func (v *InputValue) GoString() string {
	if (v.SourceRange != tfdiags.SourceRange{}) {
		return fmt.Sprintf("&terraform.InputValue{Value: %#v, SourceType: %#v, SourceRange: %#v}", v.Value, v.SourceType, v.SourceRange)
	} else {
		return fmt.Sprintf("&terraform.InputValue{Value: %#v, SourceType: %#v}", v.Value, v.SourceType)
	}
}

func (v ValueSourceType) GoString() string {
	return fmt.Sprintf("terraform.%s", v)
}

//go:generate go run golang.org/x/tools/cmd/stringer -type ValueSourceType

// InputValues is a map of InputValue instances.
type InputValues map[string]*InputValue

// InputValuesFromCaller turns the given map of naked values into an
// InputValues that attributes each value to "a caller", using the source
// type ValueFromCaller. This is primarily useful for testing purposes.
//
// This should not be used as a general way to convert map[string]cty.Value
// into InputValues, since in most real cases we want to set a suitable
// other SourceType and possibly SourceRange value.
func InputValuesFromCaller(vals map[string]cty.Value) InputValues {
	ret := make(InputValues, len(vals))
	for k, v := range vals {
		ret[k] = &InputValue{
			Value:      v,
			SourceType: ValueFromCaller,
		}
	}
	return ret
}

// Override merges the given value maps with the receiver, overriding any
// conflicting keys so that the latest definition wins.
func (vv InputValues) Override(others ...InputValues) InputValues {
	// FIXME: This should check to see if any of the values are maps and
	// merge them if so, in order to preserve the behavior from prior to
	// Terraform 0.12.
	ret := make(InputValues)
	for k, v := range vv {
		ret[k] = v
	}
	for _, other := range others {
		for k, v := range other {
			ret[k] = v
		}
	}
	return ret
}

// JustValues returns a map that just includes the values, discarding the
// source information.
func (vv InputValues) JustValues() map[string]cty.Value {
	ret := make(map[string]cty.Value, len(vv))
	for k, v := range vv {
		ret[k] = v.Value
	}
	return ret
}

// DefaultVariableValues returns an InputValues map representing the default
// values specified for variables in the given configuration map.
func DefaultVariableValues(configs map[string]*configs.Variable) InputValues {
	ret := make(InputValues)
	for k, c := range configs {
		if c.Default == cty.NilVal {
			continue
		}
		ret[k] = &InputValue{
			Value:       c.Default,
			SourceType:  ValueFromConfig,
			SourceRange: tfdiags.SourceRangeFromHCL(c.DeclRange),
		}
	}
	return ret
}

// SameValues returns true if the given InputValues has the same values as
// the receiever, disregarding the source types and source ranges.
//
// Values are compared using the cty "RawEquals" method, which means that
// unknown values can be considered equal to one another if they are of the
// same type.
func (vv InputValues) SameValues(other InputValues) bool {
	if len(vv) != len(other) {
		return false
	}

	for k, v := range vv {
		ov, exists := other[k]
		if !exists {
			return false
		}
		if !v.Value.RawEquals(ov.Value) {
			return false
		}
	}

	return true
}

// HasValues returns true if the reciever has the same values as in the given
// map, disregarding the source types and source ranges.
//
// Values are compared using the cty "RawEquals" method, which means that
// unknown values can be considered equal to one another if they are of the
// same type.
func (vv InputValues) HasValues(vals map[string]cty.Value) bool {
	if len(vv) != len(vals) {
		return false
	}

	for k, v := range vv {
		oVal, exists := vals[k]
		if !exists {
			return false
		}
		if !v.Value.RawEquals(oVal) {
			return false
		}
	}

	return true
}

// Identical returns true if the given InputValues has the same values,
// source types, and source ranges as the receiver.
//
// Values are compared using the cty "RawEquals" method, which means that
// unknown values can be considered equal to one another if they are of the
// same type.
//
// This method is primarily for testing. For most practical purposes, it's
// better to use SameValues or HasValues.
func (vv InputValues) Identical(other InputValues) bool {
	if len(vv) != len(other) {
		return false
	}

	for k, v := range vv {
		ov, exists := other[k]
		if !exists {
			return false
		}
		if !v.Value.RawEquals(ov.Value) {
			return false
		}
		if v.SourceType != ov.SourceType {
			return false
		}
		if v.SourceRange != ov.SourceRange {
			return false
		}
	}

	return true
}

// checkInputVariables ensures that variable values supplied at the UI conform
// to their corresponding declarations in configuration.
//
// The set of values is considered valid only if the returned diagnostics
// does not contain errors. A valid set of values may still produce warnings,
// which should be returned to the user.
func checkInputVariables(vcs map[string]*configs.Variable, vs InputValues) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for name, vc := range vcs {
		val, isSet := vs[name]
		if !isSet {
			// Always an error, since the caller should already have included
			// default values from the configuration in the values map.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unassigned variable",
				fmt.Sprintf("The input variable %q has not been assigned a value. This is a bug in Terraform; please report it in a GitHub issue.", name),
			))
			continue
		}

		wantType := vc.Type

		// A given value is valid if it can convert to the desired type.
		_, err := convert.Convert(val.Value, wantType)
		if err != nil {
			switch val.SourceType {
			case ValueFromConfig, ValueFromAutoFile, ValueFromNamedFile:
				// We have source location information for these.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid value for input variable",
					Detail:   fmt.Sprintf("The given value is not valid for variable %q: %s.", name, err),
					Subject:  val.SourceRange.ToHCL().Ptr(),
				})
			case ValueFromEnvVar:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid value for input variable",
					fmt.Sprintf("The environment variable TF_VAR_%s does not contain a valid value for variable %q: %s.", name, name, err),
				))
			case ValueFromCLIArg:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid value for input variable",
					fmt.Sprintf("The argument -var=\"%s=...\" does not contain a valid value for variable %q: %s.", name, name, err),
				))
			case ValueFromInput:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid value for input variable",
					fmt.Sprintf("The value entered for variable %q is not valid: %s.", name, err),
				))
			default:
				// The above gets us good coverage for the situations users
				// are likely to encounter with their own inputs. The other
				// cases are generally implementation bugs, so we'll just
				// use a generic error for these.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid value for input variable",
					fmt.Sprintf("The value provided for variable %q is not valid: %s.", name, err),
				))
			}
		}
	}

	// Check for any variables that are assigned without being configured.
	// This is always an implementation error in the caller, because we
	// expect undefined variables to be caught during context construction
	// where there is better context to report it well.
	for name := range vs {
		if _, defined := vcs[name]; !defined {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Value assigned to undeclared variable",
				fmt.Sprintf("A value was assigned to an undeclared input variable %q.", name),
			))
		}
	}

	return diags
}
