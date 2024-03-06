// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package backendbase

import (
	"fmt"
	"math/big"
	"os"

	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// GetPathDefault traverses the steps of the given path through the given
// value, and then returns either that value or the value given in def,
// if the found value was null.
//
// This function expects the given path to be valid for the given value, and
// will panic if not. This should be used only for values that have already
// been coerced into a known-good data type, which is typically achieved by
// passing the value that was returned by [Base.PrepareConfig], which is also
// the value passed to [Backend.Configure].
func GetPathDefault(v cty.Value, path cty.Path, def cty.Value) cty.Value {
	v, err := path.Apply(v)
	if err != nil {
		panic(fmt.Sprintf("invalid path: %s", tfdiags.FormatError(err)))
	}
	if v.IsNull() {
		return def
	}
	return v
}

// GetAttrDefault is like [GetPathDefault] but more convenient for the common
// case of looking up a single top-level attribute.
func GetAttrDefault(v cty.Value, attrName string, def cty.Value) cty.Value {
	return GetPathDefault(v, cty.GetAttrPath(attrName), def)
}

// GetPathEnvDefault is like [GetPathDefault] except that the default value
// is taken from an environment variable of the name given in defEnv, returned
// as a string value.
//
// If that environment variable is unset or has an empty-string value then
// the result is null, as a convenience to callers so that they don't need to
// handle both null-ness and empty-string-ness as variants of "unset".
//
// This function panics in the same situations as [GetPathDefault].
func GetPathEnvDefault(v cty.Value, path cty.Path, defEnv string) cty.Value {
	v, err := path.Apply(v)
	if err != nil {
		panic(fmt.Sprintf("invalid path: %s", tfdiags.FormatError(err)))
	}
	if v.IsNull() {
		if defStr := os.Getenv(defEnv); defStr != "" {
			return cty.StringVal(defStr)
		}
	}
	return v
}

// GetAttrEnvDefault is like [GetPathEnvDefault] but more convenient for the
// common case of looking up a single top-level attribute.
func GetAttrEnvDefault(v cty.Value, attrName string, defEnv string) cty.Value {
	return GetPathEnvDefault(v, cty.GetAttrPath(attrName), defEnv)
}

// GetPathEnvDefaultFallback is like [GetPathEnvDefault] except that if
// neither the attribute nor the environment variable are set then instead
// of returning null it will return the given fallback value.
//
// Unless the fallback value is null itself, this function guarantees to never
// return null.
func GetPathEnvDefaultFallback(v cty.Value, path cty.Path, defEnv string, fallback cty.Value) cty.Value {
	ret := GetPathEnvDefault(v, path, defEnv)
	if ret.IsNull() {
		return fallback
	}
	return ret
}

// GetAttrEnvDefaultFallback is like [GetPathEnvDefault] except that if
// neither the attribute nor the environment variable are set then instead
// of returning null it will return the given fallback value.
//
// Unless the fallback value is null itself, this function guarantees to never
// return null.
func GetAttrEnvDefaultFallback(v cty.Value, attrName string, defEnv string, fallback cty.Value) cty.Value {
	ret := GetAttrEnvDefault(v, attrName, defEnv)
	if ret.IsNull() {
		return fallback
	}
	return ret
}

// IntValue converts a cty value into a Go int64, or returns an error if that's
// not possible.
func IntValue(v cty.Value) (int64, error) {
	v, err := convert.Convert(v, cty.Number)
	if err != nil {
		return 0, err
	}
	if v.IsNull() {
		return 0, fmt.Errorf("must not be null")
	}
	bf := v.AsBigFloat()
	ret, acc := bf.Int64()
	if acc != big.Exact {
		return 0, fmt.Errorf("must not be a whole number")
	}
	return ret, nil
}

// BoolValue converts a cty value Go bool, or returns an error if that's not
// possible.
func BoolValue(v cty.Value) (bool, error) {
	v, err := convert.Convert(v, cty.Bool)
	if err != nil {
		return false, err
	}
	if v.IsNull() {
		return false, fmt.Errorf("must not be null")
	}
	return v.True(), nil
}

// MustBoolValue converts a cty value Go bool, or panics if that's not possible.
func MustBoolValue(v cty.Value) bool {
	ret, err := BoolValue(v)
	if err != nil {
		panic(fmt.Sprintf("MustBoolValue: %s", err))
	}
	return ret
}

// ErrorAsDiagnostics wraps a non-nil error as a tfdiags.Diagnostics.
//
// Panics if the given error is nil, since the caller should only be using
// this if they've encountered a non-nil error.
//
// This is here just as a temporary measure to preserve the old treatment of
// errors returned from legacy helper/schema-based backend implementations,
// so that we can minimize the churn caused in the first iteration of adopting
// backendbase.
//
// In the long run backends should produce higher-quality diagnostics directly
// themselves, but we wanted to first complete the deprecation of the
// legacy/helper/schema package with only mechanical code updates and then
// save diagnostic quality improvements for a later time.
func ErrorAsDiagnostics(err error) tfdiags.Diagnostics {
	if err == nil {
		panic("ErrorAsDiagnostics with nil error")
	}
	var diags tfdiags.Diagnostics
	// This produces a very low-quality diagnostic message, but it matches
	// how legacy helper/schema dealt with the same situation.
	diags = diags.Append(err)
	return diags
}
