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
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planproto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
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
		Want     *stacks.PlannedChange
		WantErr  string
	}{
		"header": {
			Receiver: &PlannedChangeHeader{
				TerraformVersion: version.Must(version.NewSemver("1.2.3-beta4")),
			},
			Want: &stacks.PlannedChange{
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
			Want: &stacks.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanApplyable{
						Applyable: true,
					}),
				},
				Descriptions: []*stacks.PlannedChange_ChangeDescription{
					{
						Description: &stacks.PlannedChange_ChangeDescription_PlanApplyable{
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
			Want: &stacks.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanApplyable{
						// false is the default
					}),
				},
				Descriptions: []*stacks.PlannedChange_ChangeDescription{
					{
						Description: &stacks.PlannedChange_ChangeDescription_PlanApplyable{
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
			Want: &stacks.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanComponentInstance{
						ComponentInstanceAddr: "component.foo",
						PlanTimestamp:         "2017-03-27T10:00:00-08:00",
						PlannedAction:         planproto.Action_CREATE,
					}),
				},
				Descriptions: []*stacks.PlannedChange_ChangeDescription{
					{
						Description: &stacks.PlannedChange_ChangeDescription_ComponentInstancePlanned{
							ComponentInstancePlanned: &stacks.PlannedChange_ComponentInstance{
								Addr: &stacks.ComponentInstanceInStackAddr{
									ComponentAddr:         "component.foo",
									ComponentInstanceAddr: "component.foo",
								},
								Actions: []stacks.ChangeType{stacks.ChangeType_CREATE},
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
			Want: &stacks.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanComponentInstance{
						ComponentInstanceAddr: `component.foo["bar"]`,
						PlanTimestamp:         "2017-03-27T10:00:00-08:00",
					}),
				},
				Descriptions: []*stacks.PlannedChange_ChangeDescription{
					{
						Description: &stacks.PlannedChange_ChangeDescription_ComponentInstancePlanned{
							ComponentInstancePlanned: &stacks.PlannedChange_ComponentInstance{
								Actions: []stacks.ChangeType{stacks.ChangeType_NOOP},
								Addr: &stacks.ComponentInstanceInStackAddr{
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
			Want: &stacks.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanComponentInstance{
						ComponentInstanceAddr: `stack.a["boop"].component.foo`,
						PlannedAction:         planproto.Action_DELETE,
					}),
				},
				Descriptions: []*stacks.PlannedChange_ChangeDescription{
					{
						Description: &stacks.PlannedChange_ChangeDescription_ComponentInstancePlanned{
							ComponentInstancePlanned: &stacks.PlannedChange_ComponentInstance{
								Addr: &stacks.ComponentInstanceInStackAddr{
									ComponentAddr:         "stack.a.component.foo",
									ComponentInstanceAddr: `stack.a["boop"].component.foo`,
								},
								Actions: []stacks.ChangeType{stacks.ChangeType_DELETE},
							},
						},
					},
				},
			},
		},
		"resource instance deferred": {
			Receiver: &PlannedChangeDeferredResourceInstancePlanned{
				ResourceInstancePlanned: PlannedChangeResourceInstancePlanned{
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
				DeferredReason: providers.DeferredReasonResourceConfigUnknown,
			},
			Want: &stacks.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanDeferredResourceInstanceChange{
						Change: &tfstackdata1.PlanResourceInstanceChangePlanned{
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
						},
						Deferred: &planproto.Deferred{
							Reason: planproto.DeferredReason_RESOURCE_CONFIG_UNKNOWN,
						},
					}),
				},
				Descriptions: []*stacks.PlannedChange_ChangeDescription{
					{
						Description: &stacks.PlannedChange_ChangeDescription_ResourceInstanceDeferred{
							ResourceInstanceDeferred: &stacks.PlannedChange_ResourceInstanceDeferred{
								ResourceInstance: &stacks.PlannedChange_ResourceInstance{
									Addr: &stacks.ResourceInstanceObjectInStackAddr{
										ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
										ResourceInstanceAddr:  `module.pizza["chicken"].thingy.wotsit[1]`,
										DeposedKey:            "aaaaaaaa",
									},
									ResourceMode: stacks.ResourceMode_MANAGED,
									ResourceType: "thingy",
									ProviderAddr: "example.com/thingers/thingy",
									Actions:      []stacks.ChangeType{stacks.ChangeType_CREATE},
									ActionReason: "ResourceInstanceChangeNoReason",
									Index: &stacks.PlannedChange_ResourceInstance_Index{
										Value: &stacks.DynamicValue{
											Msgpack: []byte{0x92, 0xc4, 0x08, 0x22, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x22, 0x01}, // 1
										},
									},
									ModuleAddr:   `module.pizza["chicken"]`,
									ResourceName: "wotsit",
									Values: &stacks.DynamicValueChange{
										Old: &stacks.DynamicValue{
											Msgpack: []byte{'\xc0'}, // null
										},
										New: &stacks.DynamicValue{
											Msgpack: []byte{'\x80'}, // zero-length mapping
										},
									},
								},
								Deferred: &stacks.Deferred{
									Reason: stacks.Deferred_RESOURCE_CONFIG_UNKNOWN,
								},
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
			Want: &stacks.PlannedChange{
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
				Descriptions: []*stacks.PlannedChange_ChangeDescription{
					{
						Description: &stacks.PlannedChange_ChangeDescription_ResourceInstancePlanned{
							ResourceInstancePlanned: &stacks.PlannedChange_ResourceInstance{
								Addr: &stacks.ResourceInstanceObjectInStackAddr{
									ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
									ResourceInstanceAddr:  `module.pizza["chicken"].thingy.wotsit[1]`,
									DeposedKey:            "aaaaaaaa",
								},
								ResourceMode: stacks.ResourceMode_MANAGED,
								ResourceType: "thingy",
								ProviderAddr: "example.com/thingers/thingy",
								Actions:      []stacks.ChangeType{stacks.ChangeType_CREATE},
								ActionReason: "ResourceInstanceChangeNoReason",
								Index: &stacks.PlannedChange_ResourceInstance_Index{
									Value: &stacks.DynamicValue{
										Msgpack: []byte{0x92, 0xc4, 0x08, 0x22, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x22, 0x01}, // 1
									},
								},
								ModuleAddr:   `module.pizza["chicken"]`,
								ResourceName: "wotsit",
								Values: &stacks.DynamicValueChange{
									Old: &stacks.DynamicValue{
										Msgpack: []byte{'\xc0'}, // null
									},
									New: &stacks.DynamicValue{
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
			Want: &stacks.PlannedChange{
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
				Descriptions: []*stacks.PlannedChange_ChangeDescription{
					{
						Description: &stacks.PlannedChange_ChangeDescription_ResourceInstancePlanned{
							ResourceInstancePlanned: &stacks.PlannedChange_ResourceInstance{
								Addr: &stacks.ResourceInstanceObjectInStackAddr{
									ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
									ResourceInstanceAddr:  `module.pizza["chicken"].thingy.wotsit[1]`,
									DeposedKey:            "aaaaaaaa",
								},
								ResourceMode: stacks.ResourceMode_MANAGED,
								ResourceType: "thingy",
								ProviderAddr: "example.com/thingers/thingy",
								Actions:      []stacks.ChangeType{stacks.ChangeType_DELETE, stacks.ChangeType_CREATE},
								ActionReason: "ResourceInstanceChangeNoReason",
								Index: &stacks.PlannedChange_ResourceInstance_Index{
									Value: &stacks.DynamicValue{
										Msgpack: []byte{0x92, 0xc4, 0x08, 0x22, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x22, 0x01}, // 1
									},
								},
								ModuleAddr:   `module.pizza["chicken"]`,
								ResourceName: "wotsit",
								Values: &stacks.DynamicValueChange{
									Old: &stacks.DynamicValue{
										Msgpack: []byte("\x81\xa3foo\xa3bar"),
									},
									New: &stacks.DynamicValue{
										Msgpack: []byte("\x81\xa3foo\xa3baz"),
									},
								},
								ReplacePaths: []*stacks.AttributePath{
									{
										Steps: []*stacks.AttributePath_Step{
											{
												Selector: &stacks.AttributePath_Step_AttributeName{
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
		"resource instance planned import": {
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
					ProviderAddr: addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.MustParseProviderSourceString("example.com/thingers/thingy"),
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.NoOp,
						Before: emptyObjectForPlan,
						After:  emptyObjectForPlan,
						Importing: &plans.ImportingSrc{
							ID: "bbbbbbb",
						},
					},
				},
			},
			Want: &stacks.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanResourceInstanceChangePlanned{
						ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
						ResourceInstanceAddr:  `module.pizza["chicken"].thingy.wotsit[1]`,
						ProviderConfigAddr:    `provider["example.com/thingers/thingy"]`,
						Change: &planproto.ResourceInstanceChange{
							Addr: `module.pizza["chicken"].thingy.wotsit[1]`,
							Change: &planproto.Change{
								Action: planproto.Action_NOOP,
								Values: []*planproto.DynamicValue{
									{Msgpack: []byte{'\x80'}}, // zero-length mapping
								},
								Importing: &planproto.Importing{
									Id: "bbbbbbb",
								},
							},
							Provider: `provider["example.com/thingers/thingy"]`,
						},
					}),
				},
				Descriptions: []*stacks.PlannedChange_ChangeDescription{
					{
						Description: &stacks.PlannedChange_ChangeDescription_ResourceInstancePlanned{
							ResourceInstancePlanned: &stacks.PlannedChange_ResourceInstance{
								Addr: &stacks.ResourceInstanceObjectInStackAddr{
									ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
									ResourceInstanceAddr:  `module.pizza["chicken"].thingy.wotsit[1]`,
								},
								ResourceMode: stacks.ResourceMode_MANAGED,
								ResourceType: "thingy",
								ProviderAddr: "example.com/thingers/thingy",
								Actions:      []stacks.ChangeType{stacks.ChangeType_NOOP},
								ActionReason: "ResourceInstanceChangeNoReason",
								Index: &stacks.PlannedChange_ResourceInstance_Index{
									Value: &stacks.DynamicValue{
										Msgpack: []byte{0x92, 0xc4, 0x08, 0x22, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x22, 0x01}, // 1
									},
								},
								ModuleAddr:   `module.pizza["chicken"]`,
								ResourceName: "wotsit",
								Values: &stacks.DynamicValueChange{
									Old: &stacks.DynamicValue{
										Msgpack: []byte{'\x80'}, // zero-length mapping
									},
									New: &stacks.DynamicValue{
										Msgpack: []byte{'\x80'}, // zero-length mapping
									},
								},
								Imported: &stacks.PlannedChange_ResourceInstance_Imported{
									ImportId: "bbbbbbb",
								},
							},
						},
					},
				},
			},
		},
		"resource instance planned moved": {
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
					PrevRunAddr: addrs.AbsResourceInstance{
						Resource: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "thingy",
							Name: "wotsit",
						}.Instance(addrs.NoKey),
						Module: addrs.RootModuleInstance.Child("pizza", addrs.StringKey("chicken")),
					},
					ProviderAddr: addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.MustParseProviderSourceString("example.com/thingers/thingy"),
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.NoOp,
						Before: emptyObjectForPlan,
						After:  emptyObjectForPlan,
					},
				},
			},
			Want: &stacks.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanResourceInstanceChangePlanned{
						ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
						ResourceInstanceAddr:  `module.pizza["chicken"].thingy.wotsit[1]`,
						ProviderConfigAddr:    `provider["example.com/thingers/thingy"]`,
						Change: &planproto.ResourceInstanceChange{
							Addr:        `module.pizza["chicken"].thingy.wotsit[1]`,
							PrevRunAddr: `module.pizza["chicken"].thingy.wotsit`,
							Change: &planproto.Change{
								Action: planproto.Action_NOOP,
								Values: []*planproto.DynamicValue{
									{Msgpack: []byte{'\x80'}}, // zero-length mapping
								},
							},
							Provider: `provider["example.com/thingers/thingy"]`,
						},
					}),
				},
				Descriptions: []*stacks.PlannedChange_ChangeDescription{
					{
						Description: &stacks.PlannedChange_ChangeDescription_ResourceInstancePlanned{
							ResourceInstancePlanned: &stacks.PlannedChange_ResourceInstance{
								Addr: &stacks.ResourceInstanceObjectInStackAddr{
									ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
									ResourceInstanceAddr:  `module.pizza["chicken"].thingy.wotsit[1]`,
								},
								ResourceMode: stacks.ResourceMode_MANAGED,
								ResourceType: "thingy",
								ProviderAddr: "example.com/thingers/thingy",
								Actions:      []stacks.ChangeType{stacks.ChangeType_NOOP},
								ActionReason: "ResourceInstanceChangeNoReason",
								Index: &stacks.PlannedChange_ResourceInstance_Index{
									Value: &stacks.DynamicValue{
										Msgpack: []byte{0x92, 0xc4, 0x08, 0x22, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x22, 0x01}, // 1
									},
								},
								ModuleAddr:   `module.pizza["chicken"]`,
								ResourceName: "wotsit",
								Values: &stacks.DynamicValueChange{
									Old: &stacks.DynamicValue{
										Msgpack: []byte{'\x80'}, // zero-length mapping
									},
									New: &stacks.DynamicValue{
										Msgpack: []byte{'\x80'}, // zero-length mapping
									},
								},
								Moved: &stacks.PlannedChange_ResourceInstance_Moved{
									PrevAddr: &stacks.ResourceInstanceInStackAddr{
										ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
										ResourceInstanceAddr:  `module.pizza["chicken"].thingy.wotsit`,
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
			Want: &stacks.PlannedChange{
				// Output value changes don't generate any raw representation;
				// the diff is only for the benefit of the operator and
				// other subsystems operating on their behalf.
				Descriptions: []*stacks.PlannedChange_ChangeDescription{
					{
						Description: &stacks.PlannedChange_ChangeDescription_OutputValuePlanned{
							OutputValuePlanned: &stacks.PlannedChange_OutputValue{
								Name:    "thingy_id",
								Actions: []stacks.ChangeType{stacks.ChangeType_UPDATE},
								Values: &stacks.DynamicValueChange{
									Old: &stacks.DynamicValue{
										Msgpack: []byte{'\x80'}, // zero-length mapping
									},
									New: &stacks.DynamicValue{
										Msgpack: []byte{'\x80'}, // zero-length mapping
									},
								},
							},
						},
					},
				},
			},
		},
		"sensitive root input variable": {
			Receiver: &PlannedChangeRootInputValue{
				Addr:  stackaddrs.InputVariable{Name: "thingy_id"},
				Value: cty.StringVal("boop").Mark(marks.Sensitive),
			},
			Want: &stacks.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanRootInputValue{
						Name: "thingy_id",
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
					}),
				},
			},
		},
		"ephemeral root input variable": {
			Receiver: &PlannedChangeRootInputValue{
				Addr:  stackaddrs.InputVariable{Name: "thingy_id"},
				Value: cty.StringVal("boop").Mark(marks.Ephemeral),
			},
			WantErr: "unexpected marks found on path: Ephemeral", // Ephemeral values should never make it this far.
		},
		"root input variable": {
			Receiver: &PlannedChangeRootInputValue{
				Addr:  stackaddrs.InputVariable{Name: "thingy_id"},
				Value: cty.StringVal("boop"),
			},
			Want: &stacks.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanRootInputValue{
						Name: "thingy_id",
						Value: &tfstackdata1.DynamicValue{
							Value: &planproto.DynamicValue{
								Msgpack: []byte("\x92\xc4\b\"string\"\xa4boop"),
							},
						},
					}),
				},
			},
		},
		"root input variable that must be re-supplied during apply": {
			Receiver: &PlannedChangeRootInputValue{
				Addr:            stackaddrs.InputVariable{Name: "thingy_id"},
				RequiredOnApply: true,
				// No value in this case: the value must be re-supplied during
				// apply specifically so that we can avoid the need to store it.
			},
			Want: &stacks.PlannedChange{
				Raw: []*anypb.Any{
					mustMarshalAnyPb(&tfstackdata1.PlanRootInputValue{
						Name:            "thingy_id",
						RequiredOnApply: true,
					}),
				},
				Descriptions: []*stacks.PlannedChange_ChangeDescription{
					&stacks.PlannedChange_ChangeDescription{
						Description: &stacks.PlannedChange_ChangeDescription_ApplyTimeInputVariable{
							ApplyTimeInputVariable: &stacks.PlannedChange_InputVariableDuringApply{
								Name: "thingy_id",
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
			if len(test.WantErr) > 0 {
				if diff := cmp.Diff(test.WantErr, err.Error()); diff != "" {
					t.Errorf("wrong error\n%s", diff)
				}
				if got != nil {
					t.Errorf("unexpected result: %v", got)
				}
				return
			}

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
