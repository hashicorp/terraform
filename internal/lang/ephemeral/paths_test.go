// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

func TestEphemeralValuePaths(t *testing.T) {
	// This test is intentionally not a thorough wringing of all possible cases
	// because EphemeralValuePaths is really just a thing wrapper around a
	// more general function in package marks, and that function already has
	// its own tests. That also in turn wraps a more-general-again function in
	// upstream cty that also has its own tests.
	v := cty.ObjectVal(map[string]cty.Value{
		"unmarked":  cty.StringVal("unmarked"),
		"sensitive": cty.StringVal("sensitive").Mark(marks.Sensitive),
		"ephemeral": cty.StringVal("ephemeral").Mark(marks.Ephemeral),
		"both":      cty.StringVal("both").Mark(marks.Ephemeral).Mark(marks.Sensitive),
		"nested": cty.ListVal([]cty.Value{
			cty.StringVal("unmarked"),
			cty.StringVal("sensitive").Mark(marks.Sensitive),
			cty.StringVal("ephemeral").Mark(marks.Ephemeral),
			cty.StringVal("both").Mark(marks.Ephemeral).Mark(marks.Sensitive),
		}),
	})
	got := cty.NewPathSet(EphemeralValuePaths(v)...)
	want := cty.NewPathSet(
		cty.GetAttrPath("ephemeral"),
		cty.GetAttrPath("both"),
		cty.GetAttrPath("nested").IndexInt(2),
		cty.GetAttrPath("nested").IndexInt(3),
	)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}
