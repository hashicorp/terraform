// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package ctyutil provides helpers for working with cty values.
package ctyutil

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// NormalizePath makes dot and bracket access identify the same named member.
// Index-key marks do not affect identity. Step boundaries are preserved:
// ["a.b"] remains distinct from ["a"]["b"].
func NormalizePath(path cty.Path) cty.Path {
	ret := path.Copy()
	for i, step := range ret {
		switch step := step.(type) {
		case cty.GetAttrStep:
			ret[i] = cty.IndexStep{Key: cty.StringVal(step.Name)}
		case cty.IndexStep:
			key, _ := step.Key.Unmark()
			ret[i] = cty.IndexStep{Key: key}
		}
	}
	return ret
}

// ApplyPath applies the steps in a path, allowing either attribute or string-index
// access to named members of maps and objects.
func ApplyPath(value cty.Value, path cty.Path) (cty.Value, error) {
	var err error
	for i, step := range path {
		switch s := step.(type) {
		case cty.GetAttrStep:
			if value.Type().IsMapType() {
				step = cty.IndexStep{Key: cty.StringVal(s.Name)}
			}
		case cty.IndexStep:
			if value.Type().IsObjectType() && s.Key.Type() == cty.String && s.Key.IsKnown() && !s.Key.IsNull() {
				step = cty.GetAttrStep{Name: s.Key.AsString()}
			}
		}
		value, err = step.Apply(value)
		if err != nil {
			return cty.NilVal, fmt.Errorf("at step %d: %s", i, err)
		}
	}
	return value, nil
}
