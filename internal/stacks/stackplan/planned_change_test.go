// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackplan

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planproto"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
)

func TestPlannedChangeAsProto(t *testing.T) {
	emptyObjectForPlan, err := plans.NewDynamicValue(cty.EmptyObjectVal, cty.EmptyObject)
	if err != nil {
		t.Fatal(err)
	}
	nonEmptyType := cty.Map(cty.String)
	beforeObjectForPlan, err := plans.NewDynamicValue(cty.MapVal(map[string]cty.Value{
		"foo": cty.StringVal("bar"),
	}), nonEmptyType)
	if err != nil {
		t.Fatal(err)
	}
	afterObjectForPlan, err := plans.NewDynamicValue(cty.MapVal(map[string]cty.Value{
		"foo": cty.StringVal("baz"),
	}), nonEmptyType)
	if err != nil {
		t.Fatal(err)
	}
	nullObjectForPlan, err := plans.NewDynamicValue(cty.NullVal(cty.EmptyObject), cty.EmptyObject)
	if err != nil {
		t.Fatal(err)
	}
	fakePlanTimestamp, err := time.Parse(time.RFC3339, "2017-03-27T10:00:00-08:00")
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		Receiver PlannedChange
		Want     *terraform1.PlannedChange
	}{
		"header": {
			Receiver: &PlannedChangeHeader{
				TerraformVersion: version.Must(version.NewSemver("1.2.3-beta4")),
			},
			Want: &terraform1.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanHeader{
						TerraformVersion: "1.2.3-beta4",
					}),
				},
			},
		},
		"applyable true": {
			Receiver: &PlannedChangeApplyable{
				Applyable: true,
			},
			Want: &terraform1.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanApplyable{
						Applyable: true,
					}),
				},
				Descriptions: []*terraform1.PlannedChange_ChangeDescription{
					{
						Description: &terraform1.PlannedChange_ChangeDescription_PlanApplyable{
							PlanApplyable: true,
						},
					},
				},
			},
		},
		"applyable false": {
			Receiver: &PlannedChangeApplyable{
				Applyable: false,
			},
			Want: &terraform1.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanApplyable{
						// false is the default
					}),
				},
				Descriptions: []*terraform1.PlannedChange_ChangeDescription{
					{
						Description: &terraform1.PlannedChange_ChangeDescription_PlanApplyable{
							PlanApplyable: false,
						},
					},
				},
			},
		},
		"component instance create": {
			Receiver: &PlannedChangeComponentInstance{
				Addr: stackaddrs.AbsComponentInstance{
					Stack: stackaddrs.RootStackInstance,
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{Name: "foo"},
					},
				},
				Action:        plans.Create,
				PlanTimestamp: fakePlanTimestamp,
			},
			Want: &terraform1.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanComponentInstance{
						ComponentInstanceAddr: "component.foo",
						PlanTimestamp:         "2017-03-27T10:00:00-08:00",
						PlannedAction:         planproto.Action_CREATE,
					}),
				},
				Descriptions: []*terraform1.PlannedChange_ChangeDescription{
					{
						Description: &terraform1.PlannedChange_ChangeDescription_ComponentInstancePlanned{
							ComponentInstancePlanned: &terraform1.PlannedChange_ComponentInstance{
								Addr: &terraform1.ComponentInstanceInStackAddr{
									ComponentAddr:         "component.foo",
									ComponentInstanceAddr: "component.foo",
								},
								Actions: []terraform1.ChangeType{terraform1.ChangeType_CREATE},
							},
						},
					},
				},
			},
		},
		"component instance noop": {
			Receiver: &PlannedChangeComponentInstance{
				Addr: stackaddrs.AbsComponentInstance{
					Stack: stackaddrs.RootStackInstance,
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{Name: "foo"},
						Key:       addrs.StringKey("bar"),
					},
				},
				Action:        plans.NoOp,
				PlanTimestamp: fakePlanTimestamp,
			},
			Want: &terraform1.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanComponentInstance{
						ComponentInstanceAddr: `component.foo["bar"]`,
						PlanTimestamp:         "2017-03-27T10:00:00-08:00",
					}),
				},
				Descriptions: []*terraform1.PlannedChange_ChangeDescription{
					{
						Description: &terraform1.PlannedChange_ChangeDescription_ComponentInstancePlanned{
							ComponentInstancePlanned: &terraform1.PlannedChange_ComponentInstance{
								Actions: []terraform1.ChangeType{terraform1.ChangeType_NOOP},
								Addr: &terraform1.ComponentInstanceInStackAddr{
									ComponentAddr:         "component.foo",
									ComponentInstanceAddr: `component.foo["bar"]`,
								},
							},
						},
					},
				},
			},
		},
		"component instance delete": {
			Receiver: &PlannedChangeComponentInstance{
				Addr: stackaddrs.AbsComponentInstance{
					Stack: stackaddrs.RootStackInstance.Child("a", addrs.StringKey("boop")),
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{Name: "foo"},
					},
				},
				Action: plans.Delete,
			},
			Want: &terraform1.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanComponentInstance{
						ComponentInstanceAddr: `stack.a["boop"].component.foo`,
						PlannedAction:         planproto.Action_DELETE,
					}),
				},
				Descriptions: []*terraform1.PlannedChange_ChangeDescription{
					{
						Description: &terraform1.PlannedChange_ChangeDescription_ComponentInstancePlanned{
							ComponentInstancePlanned: &terraform1.PlannedChange_ComponentInstance{
								Addr: &terraform1.ComponentInstanceInStackAddr{
									ComponentAddr:         "stack.a.component.foo",
									ComponentInstanceAddr: `stack.a["boop"].component.foo`,
								},
								Actions: []terraform1.ChangeType{terraform1.ChangeType_DELETE},
							},
						},
					},
				},
			},
		},
		"resource instance planned create": {
			Receiver: &PlannedChangeResourceInstancePlanned{
				ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
					Component: stackaddrs.AbsComponentInstance{
						Stack: stackaddrs.RootStackInstance.Child("a", addrs.StringKey("boop")),
						Item: stackaddrs.ComponentInstance{
							Component: stackaddrs.Component{Name: "foo"},
							Key:       addrs.StringKey("beep"),
						},
					},
					Item: addrs.AbsResourceInstanceObject{
						ResourceInstance: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "thingy",
							Name: "wotsit",
						}.Instance(addrs.IntKey(1)).Absolute(
							addrs.RootModuleInstance.Child("pizza", addrs.StringKey("chicken")),
						),
						DeposedKey: addrs.DeposedKey("aaaaaaaa"),
					},
				},
				ProviderConfigAddr: addrs.AbsProviderConfig{
					Module:   addrs.RootModule,
					Provider: addrs.MustParseProviderSourceString("example.com/thingers/thingy"),
				},
				ChangeSrc: &plans.ResourceInstanceChangeSrc{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "thingy",
						Name: "wotsit",
					}.Instance(addrs.IntKey(1)).Absolute(
						addrs.RootModuleInstance.Child("pizza", addrs.StringKey("chicken")),
					),
					DeposedKey: addrs.DeposedKey("aaaaaaaa"),
					ProviderAddr: addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.MustParseProviderSourceString("example.com/thingers/thingy"),
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Create,
						Before: nullObjectForPlan,
						After:  emptyObjectForPlan,
					},
				},
			},
			Want: &terraform1.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanResourceInstanceChangePlanned{
						ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
						ResourceInstanceAddr:  `module.pizza["chicken"].thingy.wotsit[1]`,
						DeposedKey:            "aaaaaaaa",
						ProviderConfigAddr:    `provider["example.com/thingers/thingy"]`,
						Change: &planproto.ResourceInstanceChange{
							Addr:       `module.pizza["chicken"].thingy.wotsit[1]`,
							DeposedKey: "aaaaaaaa",
							Change: &planproto.Change{
								Action: planproto.Action_CREATE,
								Values: []*planproto.DynamicValue{
									{Msgpack: []byte{'\x80'}}, // zero-length mapping
								},
							},
							Provider: `provider["example.com/thingers/thingy"]`,
						},
					}),
				},
				Descriptions: []*terraform1.PlannedChange_ChangeDescription{
					{
						Description: &terraform1.PlannedChange_ChangeDescription_ResourceInstancePlanned{
							ResourceInstancePlanned: &terraform1.PlannedChange_ResourceInstance{
								Addr: &terraform1.ResourceInstanceObjectInStackAddr{
									ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
									ResourceInstanceAddr:  `module.pizza["chicken"].thingy.wotsit[1]`,
									DeposedKey:            "aaaaaaaa",
								},
								ResourceMode: terraform1.ResourceMode_MANAGED,
								ResourceType: "thingy",
								ProviderAddr: "example.com/thingers/thingy",
								Actions:      []terraform1.ChangeType{terraform1.ChangeType_CREATE},
								Values: &terraform1.DynamicValueChange{
									Old: &terraform1.DynamicValue{
										Msgpack: []byte{'\xc0'}, // null
									},
									New: &terraform1.DynamicValue{
										Msgpack: []byte{'\x80'}, // zero-length mapping
									},
								},
							},
						},
					},
				},
			},
		},
		"resource instance planned replace": {
			Receiver: &PlannedChangeResourceInstancePlanned{
				ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
					Component: stackaddrs.AbsComponentInstance{
						Stack: stackaddrs.RootStackInstance.Child("a", addrs.StringKey("boop")),
						Item: stackaddrs.ComponentInstance{
							Component: stackaddrs.Component{Name: "foo"},
							Key:       addrs.StringKey("beep"),
						},
					},
					Item: addrs.AbsResourceInstanceObject{
						ResourceInstance: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "thingy",
							Name: "wotsit",
						}.Instance(addrs.IntKey(1)).Absolute(
							addrs.RootModuleInstance.Child("pizza", addrs.StringKey("chicken")),
						),
						DeposedKey: addrs.DeposedKey("aaaaaaaa"),
					},
				},
				ProviderConfigAddr: addrs.AbsProviderConfig{
					Module:   addrs.RootModule,
					Provider: addrs.MustParseProviderSourceString("example.com/thingers/thingy"),
				},
				ChangeSrc: &plans.ResourceInstanceChangeSrc{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "thingy",
						Name: "wotsit",
					}.Instance(addrs.IntKey(1)).Absolute(
						addrs.RootModuleInstance.Child("pizza", addrs.StringKey("chicken")),
					),
					DeposedKey: addrs.DeposedKey("aaaaaaaa"),
					ProviderAddr: addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.MustParseProviderSourceString("example.com/thingers/thingy"),
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.DeleteThenCreate,
						Before: beforeObjectForPlan,
						After:  afterObjectForPlan,
					},
					RequiredReplace: cty.NewPathSet(cty.GetAttrPath("foo")),
				},
			},
			Want: &terraform1.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanResourceInstanceChangePlanned{
						ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
						ResourceInstanceAddr:  `module.pizza["chicken"].thingy.wotsit[1]`,
						DeposedKey:            "aaaaaaaa",
						ProviderConfigAddr:    `provider["example.com/thingers/thingy"]`,
						Change: &planproto.ResourceInstanceChange{
							Addr:       `module.pizza["chicken"].thingy.wotsit[1]`,
							DeposedKey: "aaaaaaaa",
							Change: &planproto.Change{
								Action: planproto.Action_DELETE_THEN_CREATE,
								Values: []*planproto.DynamicValue{
									{Msgpack: []byte("\x81\xa3foo\xa3bar")},
									{Msgpack: []byte("\x81\xa3foo\xa3baz")},
								},
							},
							Provider: `provider["example.com/thingers/thingy"]`,
							RequiredReplace: []*planproto.Path{
								{
									Steps: []*planproto.Path_Step{
										{
											Selector: &planproto.Path_Step_AttributeName{AttributeName: "foo"},
										},
									},
								},
							},
						},
					}),
				},
				Descriptions: []*terraform1.PlannedChange_ChangeDescription{
					{
						Description: &terraform1.PlannedChange_ChangeDescription_ResourceInstancePlanned{
							ResourceInstancePlanned: &terraform1.PlannedChange_ResourceInstance{
								Addr: &terraform1.ResourceInstanceObjectInStackAddr{
									ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
									ResourceInstanceAddr:  `module.pizza["chicken"].thingy.wotsit[1]`,
									DeposedKey:            "aaaaaaaa",
								},
								ResourceMode: terraform1.ResourceMode_MANAGED,
								ResourceType: "thingy",
								ProviderAddr: "example.com/thingers/thingy",
								Actions:      []terraform1.ChangeType{terraform1.ChangeType_DELETE, terraform1.ChangeType_CREATE},
								Values: &terraform1.DynamicValueChange{
									Old: &terraform1.DynamicValue{
										Msgpack: []byte("\x81\xa3foo\xa3bar"),
									},
									New: &terraform1.DynamicValue{
										Msgpack: []byte("\x81\xa3foo\xa3baz"),
									},
								},
								ReplacePaths: []*terraform1.AttributePath{
									{
										Steps: []*terraform1.AttributePath_Step{
											{
												Selector: &terraform1.AttributePath_Step_AttributeName{
													AttributeName: "foo",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"output value updated": {
			Receiver: &PlannedChangeOutputValue{
				Addr:   stackaddrs.OutputValue{Name: "thingy_id"},
				Action: plans.Update,

				// NOTE: This is a bit unrealistic since we're reporting an
				// update but there's no difference between these two values.
				// In a real planned change this situation would be a "no-op".
				OldValue: emptyObjectForPlan,
				NewValue: emptyObjectForPlan,
			},
			Want: &terraform1.PlannedChange{
				// Output value changes don't generate any raw representation;
				// the diff is only for the benefit of the operator and
				// other subsystems operating on their behalf.
				Descriptions: []*terraform1.PlannedChange_ChangeDescription{
					{
						Description: &terraform1.PlannedChange_ChangeDescription_OutputValuePlanned{
							OutputValuePlanned: &terraform1.PlannedChange_OutputValue{
								Name:    "thingy_id",
								Actions: []terraform1.ChangeType{terraform1.ChangeType_UPDATE},
								Values: &terraform1.DynamicValueChange{
									Old: &terraform1.DynamicValue{
										Msgpack: []byte{'\x80'}, // zero-length mapping
									},
									New: &terraform1.DynamicValue{
										Msgpack: []byte{'\x80'}, // zero-length mapping
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := test.Receiver.PlannedChangeProto()
			if err != nil {
				// All errors this can generate are caused by bugs in Terraform
				// because we're serializing content that we created, and so
				// there are no _expected_ error cases.
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.Want, got, protocmp.Transform()); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

func mustMarshalAnyPb(msg proto.Message) *anypb.Any {
	var ret anypb.Any
	err := anypb.MarshalFrom(&ret, msg, proto.MarshalOptions{})
	if err != nil {
		panic(err)
	}
	return &ret
}
