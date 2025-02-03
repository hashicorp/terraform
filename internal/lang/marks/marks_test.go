// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package marks

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

func TestDeprecatedPaths(t *testing.T) {
	tests := []struct {
		name string
		val  cty.Value
		want []cty.PathValueMarks
	}{
		{
			name: "single deprecated path",
			val: cty.ObjectVal(map[string]cty.Value{
				"deprecated": cty.StringVal("value").Mark(DeprecationMark{
					AttrPath: cty.GetAttrPath("deprecated"),
					Message:  "This attribute is deprecated",
				}),
			}),
			want: []cty.PathValueMarks{
				{
					Path: cty.GetAttrPath("deprecated"),
					Marks: map[interface{}]struct{}{
						DeprecationMark{
							AttrPath: cty.GetAttrPath("deprecated"),
							Message:  "This attribute is deprecated",
						}: {},
					},
				},
			},
		},
		{
			name: "nested deprecated path",
			val: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"deprecated": cty.StringVal("value").Mark(DeprecationMark{
						AttrPath: cty.GetAttrPath("nested").GetAttr("deprecated"),
						Message:  "This attribute is deprecated",
					}),
				}),
			}),
			want: []cty.PathValueMarks{
				{
					Path: cty.GetAttrPath("nested").GetAttr("deprecated"),
					Marks: map[interface{}]struct{}{
						DeprecationMark{
							AttrPath: cty.GetAttrPath("nested").GetAttr("deprecated"),
							Message:  "This attribute is deprecated",
						}: {},
					},
				},
			},
		},
		{
			name: "list with deprecated path",
			val: cty.ListVal([]cty.Value{
				cty.StringVal("value1"),
				cty.StringVal("value2").Mark(DeprecationMark{
					AttrPath: cty.IndexIntPath(1),
					Message:  "This element is deprecated",
				}),
			}),
			want: []cty.PathValueMarks{
				{
					Path: cty.IndexIntPath(1),
					Marks: map[interface{}]struct{}{
						DeprecationMark{
							AttrPath: cty.IndexIntPath(1),
							Message:  "This element is deprecated",
						}: {},
					},
				},
			},
		},
		{
			name: "deeply nested deprecated path",
			val: cty.ObjectVal(map[string]cty.Value{
				"level1": cty.ObjectVal(map[string]cty.Value{
					"level2": cty.ObjectVal(map[string]cty.Value{
						"deprecated": cty.StringVal("value").Mark(DeprecationMark{
							AttrPath: cty.GetAttrPath("level1").GetAttr("level2").GetAttr("deprecated"),
							Message:  "This attribute is deprecated",
						}),
					}),
				}),
			}),
			want: []cty.PathValueMarks{
				{
					Path: cty.GetAttrPath("level1").GetAttr("level2").GetAttr("deprecated"),
					Marks: map[interface{}]struct{}{
						DeprecationMark{
							AttrPath: cty.GetAttrPath("level1").GetAttr("level2").GetAttr("deprecated"),
							Message:  "This attribute is deprecated",
						}: {},
					},
				},
			},
		},
		{
			name: "no deprecated paths",
			val: cty.ObjectVal(map[string]cty.Value{
				"normal": cty.StringVal("value"),
			}),
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeprecatedPaths(tt.val)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("DeprecatedPaths() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
