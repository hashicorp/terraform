// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestParserLoadValuesFile(t *testing.T) {
	tests := map[string]struct {
		Source    string
		Want      map[string]cty.Value
		DiagCount int
	}{
		"empty.tfvars": {
			"",
			map[string]cty.Value{},
			0,
		},
		"empty.json": {
			"{}",
			map[string]cty.Value{},
			0,
		},
		"zerolen.json": {
			"",
			map[string]cty.Value{},
			2, // syntax error and missing root object
		},
		"one-number.tfvars": {
			"foo = 1\n",
			map[string]cty.Value{
				"foo": cty.NumberIntVal(1),
			},
			0,
		},
		"one-number.tfvars.json": {
			`{"foo": 1}`,
			map[string]cty.Value{
				"foo": cty.NumberIntVal(1),
			},
			0,
		},
		"two-bools.tfvars": {
			"foo = true\nbar = false\n",
			map[string]cty.Value{
				"foo": cty.True,
				"bar": cty.False,
			},
			0,
		},
		"two-bools.tfvars.json": {
			`{"foo": true, "bar": false}`,
			map[string]cty.Value{
				"foo": cty.True,
				"bar": cty.False,
			},
			0,
		},
		"invalid-syntax.tfvars": {
			"foo bar baz\n",
			map[string]cty.Value{},
			2, // invalid block definition, and unexpected foo block (the latter due to parser recovery behavior)
		},
		"block.tfvars": {
			"foo = true\ninvalid {\n}\n",
			map[string]cty.Value{
				"foo": cty.True,
			},
			1, // blocks are not allowed
		},
		"variables.tfvars": {
			"baz = true\nfoo = var.baz\n",
			map[string]cty.Value{
				"baz": cty.True,
				"foo": cty.DynamicVal,
			},
			1, // variables cannot be referenced here
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := testParser(map[string]string{
				name: test.Source,
			})
			got, diags := p.LoadValuesFile(name)
			if len(diags) != test.DiagCount {
				t.Errorf("wrong number of diagnostics %d; want %d", len(diags), test.DiagCount)
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}

			if len(got) != len(test.Want) {
				t.Errorf("wrong number of result keys %d; want %d", len(got), len(test.Want))
			}

			for name, gotVal := range got {
				wantVal := test.Want[name]
				if wantVal == cty.NilVal {
					t.Errorf("unexpected result key %q", name)
					continue
				}

				if !gotVal.RawEquals(wantVal) {
					t.Errorf("wrong value for %q\ngot:  %#v\nwant: %#v", name, gotVal, wantVal)
					continue
				}
			}
		})
	}
}
