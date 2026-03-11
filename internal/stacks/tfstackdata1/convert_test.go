// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tfstackdata1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans/planproto"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
)

func TestDynamicValueToTFStackData1(t *testing.T) {
	startVal := cty.ObjectVal(map[string]cty.Value{
		"a": cty.StringVal("a").Mark(marks.Sensitive),
		"b": cty.StringVal("b"),
		"c": cty.ListVal([]cty.Value{
			cty.StringVal("c[0]"),
			cty.StringVal("c[1]").Mark(marks.Sensitive),
		}),
	})
	ty := startVal.Type()

	partial, err := stacks.ToDynamicValue(startVal, ty)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	got := Terraform1ToStackDataDynamicValue(partial)
	want := &DynamicValue{
		Value: &planproto.DynamicValue{
			// The following is cty's canonical MessagePack encoding of
			// the unmarked version of startVal:
			//   - \x83 marks the start of a three-element "fixmap"
			//   - \xa1 and \xa4 mark a one-element and a four-element fixstr respectively
			//   - \x92 marks the start of a two-element "fixarray"
			// cty/msgpack always orders object attribute names lexically when
			// serializing, so we can safely rely on the order of the attrs.
			Msgpack: []byte("\x83\xa1a\xa1a\xa1b\xa1b\xa1c\x92\xa4c[0]\xa4c[1]"),
		},

		SensitivePaths: []*planproto.Path{
			{
				Steps: []*planproto.Path_Step{
					{
						Selector: &planproto.Path_Step_AttributeName{
							AttributeName: "a",
						},
					},
				},
			},
			{
				Steps: []*planproto.Path_Step{
					{
						Selector: &planproto.Path_Step_AttributeName{
							AttributeName: "c",
						},
					},
					{
						Selector: &planproto.Path_Step_ElementKey{
							ElementKey: &planproto.DynamicValue{
								Msgpack: []byte{0b00000001}, // MessagePack-encoded fixint 1
							},
						},
					},
				},
			},
		},
	}

	// DynamicValueToTFStackData1 doesn't guarantee the order of the
	// entries in SensitivePaths, so we'll normalize what we got.
	// We distinguish the two expected paths by their number of steps.
	if len(got.SensitivePaths) == 2 && len(got.SensitivePaths[0].Steps) == 2 {
		got.SensitivePaths[0], got.SensitivePaths[1] = got.SensitivePaths[1], got.SensitivePaths[0]
	}

	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestDynamicValueFromTFStackData1(t *testing.T) {
	startVal := cty.ObjectVal(map[string]cty.Value{
		"a": cty.StringVal("a").Mark(marks.Sensitive),
		"b": cty.StringVal("b"),
		"c": cty.ListVal([]cty.Value{
			cty.StringVal("c[0]"),
			cty.StringVal("c[1]").Mark(marks.Sensitive),
		}),
	})
	ty := startVal.Type()

	// We'll use the MessagePack encoder directly to get the raw bytes
	// representing the above, just for maintainability's sake since it's
	// challenging to read and modify raw MessagePack values.
	unmarkedVal, _ := startVal.UnmarkDeep()
	raw, err := msgpack.Marshal(unmarkedVal, ty)
	if err != nil {
		t.Fatal(err)
	}

	input := &DynamicValue{
		Value: &planproto.DynamicValue{
			Msgpack: raw,
		},
		SensitivePaths: []*planproto.Path{
			{
				Steps: []*planproto.Path_Step{
					{
						Selector: &planproto.Path_Step_AttributeName{
							AttributeName: "a",
						},
					},
				},
			},
			{
				Steps: []*planproto.Path_Step{
					{
						Selector: &planproto.Path_Step_AttributeName{
							AttributeName: "c",
						},
					},
					{
						Selector: &planproto.Path_Step_ElementKey{
							ElementKey: &planproto.DynamicValue{
								Msgpack: []byte{0b00000001}, // MessagePack-encoded fixint 1
							},
						},
					},
				},
			},
		},
	}

	got, err := DynamicValueFromTFStackData1(input, ty)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want := startVal

	if !want.RawEquals(got) {
		t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
	}
}
