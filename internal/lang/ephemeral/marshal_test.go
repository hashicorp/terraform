// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"testing"

	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

func TestEphemeral_removeEphemeralValues(t *testing.T) {
	for name, tc := range map[string]struct {
		input cty.Value
		want  cty.Value
	}{
		"empty case": {
			input: cty.NullVal(cty.DynamicPseudoType),
			want:  cty.NullVal(cty.DynamicPseudoType),
		},
		"ephemeral marks case": {
			input: cty.ObjectVal(map[string]cty.Value{
				"ephemeral": cty.StringVal("ephemeral_value").Mark(marks.Ephemeral),
				"normal":    cty.StringVal("normal_value"),
			}),
			want: cty.ObjectVal(map[string]cty.Value{
				"ephemeral": cty.NullVal(cty.String),
				"normal":    cty.StringVal("normal_value"),
			}),
		},
		"sensitive marks case": {
			input: cty.ObjectVal(map[string]cty.Value{
				"sensitive": cty.StringVal("sensitive_value").Mark(marks.Sensitive),
				"normal":    cty.StringVal("normal_value"),
			}),
			want: cty.ObjectVal(map[string]cty.Value{
				"sensitive": cty.StringVal("sensitive_value").Mark(marks.Sensitive),
				"normal":    cty.StringVal("normal_value"),
			}),
		},
		"sensitive and ephemeral marks case": {
			input: cty.ObjectVal(map[string]cty.Value{
				"sensitive_and_ephemeral": cty.StringVal("sensitive_and_ephemeral_value").Mark(marks.Sensitive).Mark(marks.Ephemeral),
				"normal":                  cty.StringVal("normal_value"),
			}),
			want: cty.ObjectVal(map[string]cty.Value{
				"sensitive_and_ephemeral": cty.NullVal(cty.String).Mark(marks.Sensitive),
				"normal":                  cty.StringVal("normal_value"),
			}),
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := RemoveEphemeralValues(tc.input)

			if !got.RawEquals(tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}
