// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package backendbase

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// SDKLikeData offers an approximation of the legack SDK "ResourceData" API
// as a stopgap measure to help migrate all of the remote state backend
// implementations away from the legacy SDK.
//
// It's designed to wrap an object returned by [Base.PrepareConfig] which
// should therefore already have a fixed, known data type. Therefore the
// methods assume that the caller already knows what type each attribute
// should have and will panic if a caller asks for an incompatible type.
type SDKLikeData struct {
	v cty.Value
}

func NewSDKLikeData(v cty.Value) SDKLikeData {
	return SDKLikeData{v}
}

// String extracts a string attribute from a configuration object
// in a similar way to how the legacy SDK would interpret an attribute
// of type schema.TypeString, or panics if the wrapped object isn't of a
// suitable type.
func (d SDKLikeData) String(attrPath string) string {
	v := d.GetAttr(attrPath, cty.String)
	if v.IsNull() {
		return ""
	}
	return v.AsString()
}

// Int extracts a string attribute from a configuration object
// in a similar way to how the legacy SDK would interpret an attribute
// of type schema.TypeInt, or panics if the wrapped object isn't of a
// suitable type.
//
// Since the Terraform language does not have an integers-only type, this
// can fail dynamically (returning an error) if the given value has a
// fractional component.
func (d SDKLikeData) Int64(attrPath string) (int64, error) {
	// Legacy SDK used strconv.ParseInt to interpret values, so we'll
	// follow its lead here for maximal compatibility.
	v := d.GetAttr(attrPath, cty.String)
	if v.IsNull() {
		return 0, nil
	}
	return strconv.ParseInt(v.AsString(), 0, 0)
}

// Bool extracts a string attribute from a configuration object
// in a similar way to how the legacy SDK would interpret an attribute
// of type schema.TypeBool, or panics if the wrapped object isn't of a
// suitable type.
func (d SDKLikeData) Bool(attrPath string) bool {
	// Legacy SDK used strconv.ParseBool to interpret values, but it
	// did so only after the configuration was interpreted by HCL and
	// thus HCL's more constrained definition of bool still "won",
	// and we follow that tradition here.
	v := d.GetAttr(attrPath, cty.Bool)
	if v.IsNull() {
		return false
	}
	return v.True()
}

// GetAttr is just a thin wrapper around [cty.Path.Apply] that accepts
// a legacy-SDK-like dot-separated string as attribute path, instead of
// a [cty.Path] directly.
//
// It uses [SDKLikePath] to interpret the given path, and so the limitations
// of that function apply equally to this function.
//
// This function will panic if asked to extract a path that isn't compatible
// with the object type of the enclosed value.
func (d SDKLikeData) GetAttr(attrPath string, wantType cty.Type) cty.Value {
	path := SDKLikePath(attrPath)
	v, err := path.Apply(d.v)
	if err != nil {
		panic("invalid attribute path: " + err.Error())
	}
	v, err = convert.Convert(v, wantType)
	if err != nil {
		panic("incorrect attribute type: " + err.Error())
	}
	return v
}

// SDKLikePath interprets a subset of the legacy SDK attribute path syntax --
// identifiers separated by dots -- into a cty.Path.
//
// This is designed only for migrating historical remote system backends that
// were originally written using the SDK, and so it's limited only to the
// simple cases they use. It's not suitable for the more complex legacy SDK
// uses made by Terraform providers.
func SDKLikePath(rawPath string) cty.Path {
	var ret cty.Path
	remain := rawPath
	for {
		dot := strings.IndexByte(remain, '.')
		last := false
		if dot == -1 {
			dot = len(remain)
			last = true
		}

		attrName := remain[:dot]
		ret = append(ret, cty.GetAttrStep{Name: attrName})
		if last {
			return ret
		}
		remain = remain[dot+1:]
	}
}

// SDKLikeEnvDefault emulates an SDK-style "EnvDefaultFunc" by taking the
// result of [SDKLikeData.String] and a series of environment variable names.
//
// If the given string is already non-empty then it just returns it directly.
// Otherwise it returns the value of the first environment variable that has
// a non-empty value. If everything turns out empty, the result is an empty
// string.
func SDKLikeEnvDefault(v string, envNames ...string) string {
	if v == "" {
		for _, envName := range envNames {
			v = os.Getenv(envName)
			if v != "" {
				return v
			}
		}
	}
	return v
}

// SDKLikeRequiredWithEnvDefault is a convenience wrapper around
// [SDKLikeEnvDefault] which returns an error if the result is still the
// empty string even after trying all of the fallback environment variables.
//
// This wrapper requires an additional argument specifying the attribute name
// just because that becomes part of the returned error message.
func SDKLikeRequiredWithEnvDefault(attrPath string, v string, envNames ...string) (string, error) {
	ret := SDKLikeEnvDefault(v, envNames...)
	if ret == "" {
		return "", fmt.Errorf("attribute %q is required", attrPath)
	}
	return ret, nil
}
