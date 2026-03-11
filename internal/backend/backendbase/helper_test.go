// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package backendbase

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestGetPathDefault(t *testing.T) {
	tests := map[string]struct {
		Value   cty.Value
		Path    cty.Path
		Default cty.Value
		Want    cty.Value
	}{
		// The test cases here don't aim to exhaustively test all possible
		// cty.Path values, because we're just delegating to cty.Path.Apply
		// and that's already tested upstream.

		"attribute is set": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
			}),
			cty.GetAttrPath("a"),
			cty.StringVal("default"),
			cty.StringVal("a value"),
		},
		"attribute is not set": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
			cty.GetAttrPath("a"),
			cty.StringVal("default"),
			cty.StringVal("default"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := GetPathDefault(test.Value, test.Path, test.Default)
			if !test.Want.RawEquals(got) {
				t.Errorf(
					"wrong result\nvalue:   %#v\npath:    %#v\ndefault: %#v\n\ngot:  %#v\nwant: %#v",
					test.Value,
					test.Path,
					test.Default,
					got,
					test.Want,
				)
			}
		})
	}
}

func TestGetAttrDefault(t *testing.T) {
	tests := map[string]struct {
		Value   cty.Value
		Attr    string
		Default cty.Value
		Want    cty.Value
	}{
		"attribute is set": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
			}),
			"a",
			cty.StringVal("default"),
			cty.StringVal("a value"),
		},
		"attribute is not set": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
			"a",
			cty.StringVal("default"),
			cty.StringVal("default"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := GetAttrDefault(test.Value, test.Attr, test.Default)
			if !test.Want.RawEquals(got) {
				t.Errorf(
					"wrong result\nvalue:   %#v\nattr:    %#v\ndefault: %#v\n\ngot:  %#v\nwant: %#v",
					test.Value,
					test.Attr,
					test.Default,
					got,
					test.Want,
				)
			}
		})
	}
}

func TestGetPathEnvDefault(t *testing.T) {
	// This one is actually testing both GetPathEnvDefault and GetAttrEnvDefault
	// together, since they are both really just the same functionality exposed
	// in two different ways.

	t.Setenv("DEFAULT_VALUE_SET", "default")
	t.Setenv("DEFAULT_VALUE_EMPTY", "")

	tests := map[string]struct {
		Value  cty.Value
		Attr   string
		EnvVar string
		Want   cty.Value
	}{
		// The test cases here don't aim to exhaustively test all possible
		// cty.Path values, because we're just delegating to cty.Path.Apply
		// and that's already tested upstream.

		"attribute is set": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
			}),
			"a",
			"DEFAULT_VALUE_SET",
			cty.StringVal("a value"),
		},
		"attribute is not set, but environment is": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
			"a",
			"DEFAULT_VALUE_SET",
			cty.StringVal("default"),
		},
		"neither attribute or environment are set": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
			"a",
			"DEFAULT_VALUE_UNSET",
			cty.NullVal(cty.String),
		},
		"attribute is not set, and environment variable is empty": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
			"a",
			"DEFAULT_VALUE_EMPTY",
			cty.NullVal(cty.String),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Run("by attr", func(t *testing.T) {
				got := GetAttrEnvDefault(test.Value, test.Attr, test.EnvVar)
				if !test.Want.RawEquals(got) {
					t.Errorf(
						"wrong result\nvalue:    %#v\nattr:     %#v\nvariable: %#v\n\ngot:  %#v\nwant: %#v",
						test.Value,
						test.Attr,
						test.EnvVar,
						got,
						test.Want,
					)
				}
			})
			t.Run("by path", func(t *testing.T) {
				path := cty.GetAttrPath(test.Attr)
				got := GetPathEnvDefault(test.Value, path, test.EnvVar)
				if !test.Want.RawEquals(got) {
					t.Errorf(
						"wrong result\nvalue:    %#v\npath:     %#v\nvariable: %#v\n\ngot:  %#v\nwant: %#v",
						test.Value,
						path,
						test.EnvVar,
						got,
						test.Want,
					)
				}
			})
		})
	}
}
