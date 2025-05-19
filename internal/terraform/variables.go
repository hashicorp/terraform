// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// InputValue represents a raw value for a root module input variable as
// provided by the external caller into a function like terraform.Context.Plan.
//
// InputValue should represent as directly as possible what the user set the
// variable to, without any attempt to convert the value to the variable's
// type constraint or substitute the configured default values for variables
// that wasn't set. Those adjustments will be handled by Terraform Core itself
// as part of performing the requested operation.
//
// A Terraform Core caller must provide an InputValue object for each of the
// variables declared in the root module, even if the end user didn't provide
// an explicit value for some of them. See the Value field documentation for
// how to handle that situation.
//
// Terraform Core also internally uses InputValue to represent the raw value
// provided for a variable in a child module call, following the same
// conventions. However, that's an implementation detail not visible to
// outside callers.
type InputValue struct {
	// Value is the raw value as provided by the user as part of the plan
	// options, or a corresponding similar data structure for non-plan
	// operations.
	//
	// If a particular variable declared in the root module is _not_ set by
	// the user then the caller must still provide an InputValue for it but
	// must set Value to cty.NilVal to represent the absense of a value.
	// This requirement is to help detect situations where the caller isn't
	// correctly detecting and handling all of the declared variables.
	//
	// For historical reasons it's important that callers distinguish the
	// situation of the value not being set at all (cty.NilVal) from the
	// situation of it being explicitly set to null (a cty.NullVal result):
	// for "nullable" input variables that distinction unfortunately decides
	// whether the final value will be the variable's default or will be
	// explicitly null.
	Value cty.Value

	// SourceType is a high-level category for where the value of Value
	// came from, which Terraform Core uses to tailor some of its error
	// messages to be more helpful to the user.
	//
	// Some SourceType values should be accompanied by a populated SourceRange
	// value. See that field's documentation below for more information.
	SourceType ValueSourceType

	// SourceRange provides source location information for values whose
	// SourceType is either ValueFromConfig, ValueFromNamedFile, or
	// ValueForNormalFile. It is not populated for other source types, and so
	// should not be used.
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

// HasSourceRange returns true if the reciever has a source type for which
// we expect the SourceRange field to be populated with a valid range.
func (v *InputValue) HasSourceRange() bool {
	return v.SourceType.HasSourceRange()
}

// HasSourceRange returns true if the reciever is one of the source types
// that is used along with a valid SourceRange field when appearing inside an
// InputValue object.
func (v ValueSourceType) HasSourceRange() bool {
	switch v {
	case ValueFromConfig, ValueFromAutoFile, ValueFromNamedFile:
		return true
	default:
		return false
	}
}

func (v ValueSourceType) GoString() string {
	return fmt.Sprintf("terraform.%s", v)
}

func (v ValueSourceType) DiagnosticLabel() string {
	switch v {
	case ValueFromConfig:
		return "set by the default value in configuration"
	case ValueFromAutoFile:
		return "set by an automatically loaded .tfvars file"
	case ValueFromNamedFile:
		return "set by a .tfvars file passed through -var-file argument"
	case ValueFromCLIArg:
		return "set by a CLI argument"
	case ValueFromEnvVar:
		return "set by an environment variable"
	case ValueFromInput:
		return "set by an interactive input"
	case ValueFromPlan:
		return "set by the plan"
	default:
		return "unknown"
	}
}

//go:generate go tool golang.org/x/tools/cmd/stringer -type ValueSourceType

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

// checkInputVariables ensures that the caller provided an InputValue
// definition for each root module variable declared in the configuration.
// The caller must provide an InputVariables with keys exactly matching
// the declared variables, though some of them may be marked explicitly
// unset by their values being cty.NilVal.
//
// This doesn't perform any type checking, default value substitution, or
// validation checks. Those are all handled during a graph walk when we
// visit the graph nodes representing each root variable.
//
// The set of values is considered valid only if the returned diagnostics
// does not contain errors. A valid set of values may still produce warnings,
// which should be returned to the user.
func checkInputVariables(vcs map[string]*configs.Variable, vs InputValues) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for name := range vcs {
		_, isSet := vs[name]
		if !isSet {
			// Always an error, since the caller should have produced an
			// item with Value: cty.NilVal to be explicit that it offered
			// an opportunity to set this variable.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unassigned variable",
				fmt.Sprintf("The input variable %q has not been assigned a value. This is a bug in Terraform; please report it in a GitHub issue.", name),
			))
			continue
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
