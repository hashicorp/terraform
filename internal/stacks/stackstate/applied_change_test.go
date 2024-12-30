// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackstate

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans/planproto"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
	"github.com/hashicorp/terraform/internal/states"
)

func TestAppliedChangeAsProto(t *testing.T) {
	tests := map[string]struct {
		Receiver AppliedChange
		Want     *stacks.AppliedChange
	}{
		"resource instance": {
			Receiver: &AppliedChangeResourceInstanceObject{
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
							Name: "thingamajig",
						}.Instance(addrs.IntKey(1)).Absolute(
							addrs.RootModuleInstance.Child("pizza", addrs.StringKey("chicken")),
						),
					},
				},
				ProviderConfigAddr: addrs.AbsProviderConfig{
					Module:   addrs.RootModule,
					Provider: addrs.MustParseProviderSourceString("example.com/thingers/thingy"),
				},
				Schema: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
						"secret": {
							Type:      cty.String,
							Sensitive: true,
						},
					},
				},
				NewStateSrc: &states.ResourceInstanceObjectSrc{
					Status:    states.ObjectReady,
					AttrsJSON: []byte(`{"id":"bar","secret":"top"}`),
					AttrSensitivePaths: []cty.Path{
						cty.GetAttrPath("secret"),
					},
				},
			},
			Want: &stacks.AppliedChange{
				Raw: []*stacks.AppliedChange_RawChange{
					{
						Key: `RSRCstack.a["boop"].component.foo["beep"],module.pizza["chicken"].thingy.thingamajig[1],cur`,
						Value: mustMarshalAnyPb(t, &tfstackdata1.StateResourceInstanceObjectV1{
							ValueJson: json.RawMessage(`{"id":"bar","secret":"top"}`),
							SensitivePaths: []*planproto.Path{
								{
									Steps: []*planproto.Path_Step{{
										Selector: &planproto.Path_Step_AttributeName{AttributeName: "secret"}}},
								},
							},
							ProviderConfigAddr: `provider["example.com/thingers/thingy"]`,
							Status:             tfstackdata1.StateResourceInstanceObjectV1_READY,
						}),
					},
				},
				Descriptions: []*stacks.AppliedChange_ChangeDescription{
					{
						Key: `RSRCstack.a["boop"].component.foo["beep"],module.pizza["chicken"].thingy.thingamajig[1],cur`,
						Description: &stacks.AppliedChange_ChangeDescription_ResourceInstance{
							ResourceInstance: &stacks.AppliedChange_ResourceInstance{
								Addr: &stacks.ResourceInstanceObjectInStackAddr{
									ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
									ResourceInstanceAddr:  `module.pizza["chicken"].thingy.thingamajig[1]`,
								},
								NewValue: &stacks.DynamicValue{
									Msgpack: mustMsgpack(t, cty.ObjectVal(map[string]cty.Value{
										"id":     cty.StringVal("bar"),
										"secret": cty.StringVal("top"),
									}), cty.Object(map[string]cty.Type{"id": cty.String, "secret": cty.String})),
									Sensitive: []*stacks.AttributePath{{
										Steps: []*stacks.AttributePath_Step{{
											Selector: &stacks.AttributePath_Step_AttributeName{AttributeName: "secret"},
										}}},
									},
								},
								ResourceMode: stacks.ResourceMode_MANAGED,
								ResourceType: "thingy",
								ProviderAddr: "example.com/thingers/thingy",
							},
						},
					},
				},
			},
		},
		"moved_resource instance": {
			Receiver: &AppliedChangeResourceInstanceObject{
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
							Name: "thingamajig",
						}.Instance(addrs.IntKey(1)).Absolute(
							addrs.RootModuleInstance.Child("pizza", addrs.StringKey("chicken")),
						),
					},
				},
				PreviousResourceInstanceObjectAddr: &stackaddrs.AbsResourceInstanceObject{
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
							Name: "previous_thingamajig",
						}.Instance(addrs.IntKey(1)).Absolute(
							addrs.RootModuleInstance.Child("pizza", addrs.StringKey("chicken")),
						),
					},
				},
				ProviderConfigAddr: addrs.AbsProviderConfig{
					Module:   addrs.RootModule,
					Provider: addrs.MustParseProviderSourceString("example.com/thingers/thingy"),
				},
				Schema: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
						"secret": {
							Type:      cty.String,
							Sensitive: true,
						},
					},
				},
				NewStateSrc: &states.ResourceInstanceObjectSrc{
					Status:    states.ObjectReady,
					AttrsJSON: []byte(`{"id":"bar","secret":"top"}`),
					AttrSensitivePaths: []cty.Path{
						cty.GetAttrPath("secret"),
					},
				},
			},
			Want: &stacks.AppliedChange{
				Raw: []*stacks.AppliedChange_RawChange{
					{
						Key:   `RSRCstack.a["boop"].component.foo["beep"],module.pizza["chicken"].thingy.previous_thingamajig[1],cur`,
						Value: nil,
					},
					{
						Key: `RSRCstack.a["boop"].component.foo["beep"],module.pizza["chicken"].thingy.thingamajig[1],cur`,
						Value: mustMarshalAnyPb(t, &tfstackdata1.StateResourceInstanceObjectV1{
							ValueJson: json.RawMessage(`{"id":"bar","secret":"top"}`),
							SensitivePaths: []*planproto.Path{
								{
									Steps: []*planproto.Path_Step{{
										Selector: &planproto.Path_Step_AttributeName{AttributeName: "secret"}}},
								},
							},
							ProviderConfigAddr: `provider["example.com/thingers/thingy"]`,
							Status:             tfstackdata1.StateResourceInstanceObjectV1_READY,
						}),
					},
				},
				Descriptions: []*stacks.AppliedChange_ChangeDescription{
					{
						Key: `RSRCstack.a["boop"].component.foo["beep"],module.pizza["chicken"].thingy.previous_thingamajig[1],cur`,
						Description: &stacks.AppliedChange_ChangeDescription_Moved{
							Moved: &stacks.AppliedChange_Nothing{},
						},
					},
					{
						Key: `RSRCstack.a["boop"].component.foo["beep"],module.pizza["chicken"].thingy.thingamajig[1],cur`,
						Description: &stacks.AppliedChange_ChangeDescription_ResourceInstance{
							ResourceInstance: &stacks.AppliedChange_ResourceInstance{
								Addr: &stacks.ResourceInstanceObjectInStackAddr{
									ComponentInstanceAddr: `stack.a["boop"].component.foo["beep"]`,
									ResourceInstanceAddr:  `module.pizza["chicken"].thingy.thingamajig[1]`,
								},
								NewValue: &stacks.DynamicValue{
									Msgpack: mustMsgpack(t, cty.ObjectVal(map[string]cty.Value{
										"id":     cty.StringVal("bar"),
										"secret": cty.StringVal("top"),
									}), cty.Object(map[string]cty.Type{"id": cty.String, "secret": cty.String})),
									Sensitive: []*stacks.AttributePath{{
										Steps: []*stacks.AttributePath_Step{{
											Selector: &stacks.AttributePath_Step_AttributeName{AttributeName: "secret"},
										}}},
									},
								},
								ResourceMode: stacks.ResourceMode_MANAGED,
								ResourceType: "thingy",
								ProviderAddr: "example.com/thingers/thingy",
							},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := test.Receiver.AppliedChangeProto()
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.Want, got, protocmp.Transform()); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

func mustMarshalAnyPb(t *testing.T, msg proto.Message) *anypb.Any {
	var ret anypb.Any
	err := anypb.MarshalFrom(&ret, msg, proto.MarshalOptions{})
	if err != nil {
		t.Fatalf("error marshalling anypb: %q", err)
	}
	return &ret
}

func mustMsgpack(t *testing.T, v cty.Value, ty cty.Type) []byte {
	t.Helper()

	ret, err := ctymsgpack.Marshal(v, ty)
	if err != nil {
		t.Fatalf("error marshalling %#v: %s", v, err)
	}

	return ret
}
