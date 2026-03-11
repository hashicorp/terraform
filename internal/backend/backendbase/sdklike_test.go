// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package backendbase

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestSDKLikePath(t *testing.T) {
	tests := []struct {
		Input string
		Want  cty.Path
	}{
		{
			"foo",
			cty.GetAttrPath("foo"),
		},
		{
			"foo.bar",
			cty.GetAttrPath("foo").GetAttr("bar"),
		},
		{
			"foo.bar.baz",
			cty.GetAttrPath("foo").GetAttr("bar").GetAttr("baz"),
		},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			got := SDKLikePath(test.Input)
			if !test.Want.Equals(got) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestSDKLikeEnvDefault(t *testing.T) {
	t.Setenv("FALLBACK_A", "fallback a")
	t.Setenv("FALLBACK_B", "fallback b")
	t.Setenv("FALLBACK_UNSET", "")
	t.Setenv("FALLBACK_UNSET_1", "")
	t.Setenv("FALLBACK_UNSET_2", "")

	tests := map[string]struct {
		Value    string
		EnvNames []string
		Want     string
	}{
		"value is set": {
			"hello",
			[]string{"FALLBACK_A", "FALLBACK_B"},
			"hello",
		},
		"value is not set, but both fallbacks are": {
			"",
			[]string{"FALLBACK_A", "FALLBACK_B"},
			"fallback a",
		},
		"value is not set, and first callback isn't set": {
			"",
			[]string{"FALLBACK_UNSET", "FALLBACK_B"},
			"fallback b",
		},
		"value is not set, and second callback isn't set": {
			"",
			[]string{"FALLBACK_A", "FALLBACK_UNSET"},
			"fallback a",
		},
		"nothing is set": {
			"",
			[]string{"FALLBACK_UNSET_1", "FALLBACK_UNSET_2"},
			"",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := SDKLikeEnvDefault(test.Value, test.EnvNames...)
			if got != test.Want {
				t.Errorf("wrong result\nvalue: %s\nenvs:  %s\n\ngot:  %s\nwant: %s", test.Value, test.EnvNames, got, test.Want)
			}
		})
	}
}

func TestSDKLikeRequiredWithEnvDefault(t *testing.T) {
	// This intentionally doesn't duplicate all of the test cases from
	// TestSDKLikeEnvDefault, since SDKLikeRequiredWithEnvDefault is
	// just a thin wrapper which adds an error check.

	t.Setenv("FALLBACK_UNSET", "")
	_, err := SDKLikeRequiredWithEnvDefault("attr_name", "", "FALLBACK_UNSET")
	if err == nil {
		t.Fatalf("unexpected success; want error")
	}
	if got, want := err.Error(), `attribute "attr_name" is required`; got != want {
		t.Errorf("wrong error\ngot:  %s\nwant: %s", got, want)
	}
}

func TestSDKLikeData(t *testing.T) {
	d := NewSDKLikeData(cty.ObjectVal(map[string]cty.Value{
		"string": cty.StringVal("hello"),
		"int":    cty.NumberIntVal(5),
		"float":  cty.NumberFloatVal(0.5),
		"bool":   cty.True,

		"null_string": cty.NullVal(cty.String),
		"null_number": cty.NullVal(cty.Number),
		"null_bool":   cty.NullVal(cty.Bool),
	}))

	t.Run("string", func(t *testing.T) {
		got := d.String("string")
		want := "hello"
		if got != want {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})
	t.Run("null string", func(t *testing.T) {
		got := d.String("null_string")
		want := ""
		if got != want {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})
	t.Run("int as string", func(t *testing.T) {
		// This is allowed as a convenience for backends that want to
		// allow environment-based default values for integer values,
		// since environment variables are always strings and so they'd
		// need to do their own parsing afterwards anyway.
		got := d.String("int")
		want := "5"
		if got != want {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})
	t.Run("bool as string", func(t *testing.T) {
		// This is allowed as a convenience for backends that want to
		// allow environment-based default values for bool values,
		// since environment variables are always strings and so they'd
		// need to do their own parsing afterwards anyway.
		got := d.String("bool")
		want := "true"
		if got != want {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})

	t.Run("int", func(t *testing.T) {
		got, err := d.Int64("int")
		want := int64(5)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if got != want {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})
	t.Run("int with fractional part", func(t *testing.T) {
		got, err := d.Int64("float")
		if err == nil {
			t.Fatalf("unexpected success; want error\ngot: %#v", got)
		}
		// Legacy SDK exposed the strconv.ParseInt implementation detail in
		// its error message, and so for now we do the same. Maybe we'll
		// improve this later, but it would probably be better to wean
		// the backends off using the "SDKLike" helper altogether instead.
		if got, want := err.Error(), `strconv.ParseInt: parsing "0.5": invalid syntax`; got != want {
			t.Errorf("wrong error\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("null number as int", func(t *testing.T) {
		got, err := d.Int64("null_number")
		want := int64(0)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if got != want {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})

	t.Run("bool", func(t *testing.T) {
		got := d.Bool("bool")
		want := true
		if got != want {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})
	t.Run("null bool", func(t *testing.T) {
		// Assuming false for a null is quite questionable, but it's what
		// the legacy SDK did and so we'll follow its lead.
		got := d.Bool("null_bool")
		want := false
		if got != want {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})
}

func TestSDKLikeApplyEnvDefaults(t *testing.T) {
	t.Setenv("FALLBACK_BEEP", "beep from environment")
	t.Setenv("FALLBACK_UNUSED", "unused from environment")
	t.Setenv("FALLBACK_EMPTY", "")

	t.Run("success", func(t *testing.T) {
		defs := SDKLikeDefaults{
			"string_set_fallback": {
				Fallback: "fallback not used",
			},
			"string_set_env": {
				EnvVars: []string{"FALLBACK_UNUSED"},
			},
			"string_fallback_null": {
				Fallback: "boop from fallback",
			},
			"string_fallback_empty": {
				Fallback: "boop from fallback",
			},
			"string_env_null": {
				EnvVars:  []string{"FALLBACK_BEEP", "FALLBACK_UNUSED"},
				Fallback: "unused",
			},
			"string_env_empty": {
				EnvVars:  []string{"FALLBACK_BEEP", "FALLBACK_UNUSED"},
				Fallback: "unused",
			},
			"string_env_unsetfirst": {
				EnvVars:  []string{"FALLBACK_EMPTY", "FALLBACK_BEEP"},
				Fallback: "unused",
			},
			"string_env_unsetsecond": {
				EnvVars:  []string{"FALLBACK_BEEP", "FALLBACK_EMPTY"},
				Fallback: "unused",
			},
			"string_nothing_null": {
				EnvVars: []string{"FALLBACK_EMPTY"},
			},
			"string_nothing_empty": {
				EnvVars: []string{"FALLBACK_EMPTY"},
			},
		}
		got, err := defs.ApplyTo(cty.ObjectVal(map[string]cty.Value{
			"string_set_fallback":    cty.StringVal("set in config"),
			"string_set_env":         cty.StringVal("set in config"),
			"string_fallback_null":   cty.NullVal(cty.String),
			"string_fallback_empty":  cty.StringVal(""),
			"string_env_null":        cty.NullVal(cty.String),
			"string_env_empty":       cty.StringVal(""),
			"string_env_unsetfirst":  cty.NullVal(cty.String),
			"string_env_unsetsecond": cty.NullVal(cty.String),
			"string_nothing_null":    cty.NullVal(cty.String),
			"string_nothing_empty":   cty.StringVal(""),
			"passthru":               cty.EmptyObjectVal,
		}))
		want := cty.ObjectVal(map[string]cty.Value{
			"string_set_fallback":    cty.StringVal("set in config"),
			"string_set_env":         cty.StringVal("set in config"),
			"string_fallback_null":   cty.StringVal("boop from fallback"),
			"string_fallback_empty":  cty.StringVal("boop from fallback"),
			"string_env_null":        cty.StringVal("beep from environment"),
			"string_env_empty":       cty.StringVal("beep from environment"),
			"string_env_unsetfirst":  cty.StringVal("beep from environment"),
			"string_env_unsetsecond": cty.StringVal("beep from environment"),
			"string_nothing_null":    cty.NullVal(cty.String),
			"string_nothing_empty":   cty.StringVal(""),
			"passthru":               cty.EmptyObjectVal,
		})
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
}
