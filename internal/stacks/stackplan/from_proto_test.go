// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackplan

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans/planproto"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
)

func TestAddRaw(t *testing.T) {
	tests := map[string]struct {
		Raw  []*anypb.Any
		Want *Plan
	}{
		"empty": {
			Raw: nil,
			Want: &Plan{
				Root:            newStackInstance(stackaddrs.RootStackInstance),
				PrevRunStateRaw: make(map[string]*anypb.Any),
				RootInputValues: make(map[stackaddrs.InputVariable]cty.Value),
			},
		},
		"sensitive input value": {
			Raw: []*anypb.Any{
				mustMarshalAnyPb(&tfstackdata1.PlanRootInputValue{
					Name: "foo",
					Value: &tfstackdata1.DynamicValue{
						Value: &planproto.DynamicValue{
							Msgpack: []byte("\x92\xc4\b\"string\"\xa4boop"),
						},
						SensitivePaths: []*planproto.Path{
							{
								Steps: make([]*planproto.Path_Step, 0), // no steps as it is the root value
							},
						},
					},
					RequiredOnApply: false,
				}),
			},
			Want: &Plan{
				Root:            newStackInstance(stackaddrs.RootStackInstance),
				PrevRunStateRaw: make(map[string]*anypb.Any),
				RootInputValues: map[stackaddrs.InputVariable]cty.Value{
					stackaddrs.InputVariable{Name: "foo"}: cty.StringVal("boop").Mark(marks.Sensitive),
				},
			},
		},
		"input value": {
			Raw: []*anypb.Any{
				mustMarshalAnyPb(&tfstackdata1.PlanRootInputValue{
					Name: "foo",
					Value: &tfstackdata1.DynamicValue{
						Value: &planproto.DynamicValue{
							Msgpack: []byte("\x92\xc4\b\"string\"\xa4boop"),
						},
					},
					RequiredOnApply: false,
				}),
			},
			Want: &Plan{
				Root:            newStackInstance(stackaddrs.RootStackInstance),
				PrevRunStateRaw: make(map[string]*anypb.Any),
				RootInputValues: map[stackaddrs.InputVariable]cty.Value{
					stackaddrs.InputVariable{Name: "foo"}: cty.StringVal("boop"),
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			loader := NewLoader()
			for _, raw := range test.Raw {
				if err := loader.AddRaw(raw); err != nil {
					t.Errorf("AddRaw() error = %v", err)
				}
			}

			if t.Failed() {
				return
			}

			opts := cmp.Options{
				ctydebug.CmpOptions,
				collections.CmpOptions,
			}
			if diff := cmp.Diff(test.Want, loader.ret, opts...); diff != "" {
				t.Errorf("AddRaw() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
