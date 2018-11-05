package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
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

//go:generate stringer -type ValueSourceType

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
