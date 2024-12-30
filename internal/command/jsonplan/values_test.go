// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonplan

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
)

func TestMarshalAttributeValues(t *testing.T) {
	tests := []struct {
		Attr   cty.Value
		Schema *configschema.Block
		Want   attributeValues
	}{
		{
			cty.NilVal,
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			nil,
		},
		{
			cty.NullVal(cty.String),
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			nil,
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			}),
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			attributeValues{"foo": json.RawMessage(`"bar"`)},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			attributeValues{"foo": json.RawMessage(`null`)},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"bar": cty.MapVal(map[string]cty.Value{
					"hello": cty.StringVal("world"),
				}),
				"baz": cty.ListVal([]cty.Value{
					cty.StringVal("goodnight"),
					cty.StringVal("moon"),
				}),
			}),
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"bar": {
						Type:     cty.Map(cty.String),
						Required: true,
					},
					"baz": {
						Type:     cty.List(cty.String),
						Optional: true,
					},
				},
			},
			attributeValues{
				"bar": json.RawMessage(`{"hello":"world"}`),
				"baz": json.RawMessage(`["goodnight","moon"]`),
			},
		},
	}

	for _, test := range tests {
		got := marshalAttributeValues(test.Attr, test.Schema)
		eq := reflect.DeepEqual(got, test.Want)
		if !eq {
			t.Fatalf("wrong result:\nGot: %#v\nWant: %#v\n", got, test.Want)
		}
	}
}

func TestMarshalPlannedOutputs(t *testing.T) {
	after, _ := plans.NewDynamicValue(cty.StringVal("after"), cty.DynamicPseudoType)

	tests := []struct {
		Changes *plans.ChangesSrc
		Want    map[string]output
		Err     bool
	}{
		{
			&plans.ChangesSrc{},
			nil,
			false,
		},
		{
			&plans.ChangesSrc{
				Outputs: []*plans.OutputChangeSrc{
					{
						Addr: addrs.OutputValue{Name: "bar"}.Absolute(addrs.RootModuleInstance),
						ChangeSrc: plans.ChangeSrc{
							Action: plans.Create,
							After:  after,
						},
						Sensitive: false,
					},
				},
			},
			map[string]output{
				"bar": {
					Sensitive: false,
					Type:      json.RawMessage(`"string"`),
					Value:     json.RawMessage(`"after"`),
				},
			},
			false,
		},
		{ // Delete action
			&plans.ChangesSrc{
				Outputs: []*plans.OutputChangeSrc{
					{
						Addr: addrs.OutputValue{Name: "bar"}.Absolute(addrs.RootModuleInstance),
						ChangeSrc: plans.ChangeSrc{
							Action: plans.Delete,
						},
						Sensitive: false,
					},
				},
			},
			map[string]output{},
			false,
		},
	}

	for _, test := range tests {
		got, err := marshalPlannedOutputs(test.Changes)
		if test.Err {
			if err == nil {
				t.Fatal("succeeded; want error")
			}
			return
		} else if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		eq := reflect.DeepEqual(got, test.Want)
		if !eq {
			t.Fatalf("wrong result:\nGot: %#v\nWant: %#v\n", got, test.Want)
		}
	}
}

func TestMarshalPlanResources(t *testing.T) {
	tests := map[string]struct {
		Action plans.Action
		Before cty.Value
		After  cty.Value
		Want   []resource
		Err    bool
	}{
		"create with unknowns": {
			Action: plans.Create,
			Before: cty.NullVal(cty.EmptyObject),
			After: cty.ObjectVal(map[string]cty.Value{
				"woozles": cty.UnknownVal(cty.String),
				"foozles": cty.UnknownVal(cty.String),
			}),
			Want: []resource{{
				Address:         "test_thing.example",
				Mode:            "managed",
				Type:            "test_thing",
				Name:            "example",
				Index:           addrs.InstanceKey(nil),
				ProviderName:    "registry.terraform.io/hashicorp/test",
				SchemaVersion:   1,
				AttributeValues: attributeValues{},
				SensitiveValues: json.RawMessage("{}"),
			}},
			Err: false,
		},
		"delete with null and nil": {
			Action: plans.Delete,
			Before: cty.NullVal(cty.EmptyObject),
			After:  cty.NilVal,
			Want:   nil,
			Err:    false,
		},
		"delete": {
			Action: plans.Delete,
			Before: cty.ObjectVal(map[string]cty.Value{
				"woozles": cty.StringVal("foo"),
				"foozles": cty.StringVal("bar"),
			}),
			After: cty.NullVal(cty.Object(map[string]cty.Type{
				"woozles": cty.String,
				"foozles": cty.String,
			})),
			Want: nil,
			Err:  false,
		},
		"update without unknowns": {
			Action: plans.Update,
			Before: cty.ObjectVal(map[string]cty.Value{
				"woozles": cty.StringVal("foo"),
				"foozles": cty.StringVal("bar"),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"woozles": cty.StringVal("baz"),
				"foozles": cty.StringVal("bat"),
			}),
			Want: []resource{{
				Address:       "test_thing.example",
				Mode:          "managed",
				Type:          "test_thing",
				Name:          "example",
				Index:         addrs.InstanceKey(nil),
				ProviderName:  "registry.terraform.io/hashicorp/test",
				SchemaVersion: 1,
				AttributeValues: attributeValues{
					"woozles": json.RawMessage(`"baz"`),
					"foozles": json.RawMessage(`"bat"`),
				},
				SensitiveValues: json.RawMessage("{}"),
			}},
			Err: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			before, err := plans.NewDynamicValue(test.Before, test.Before.Type())
			if err != nil {
				t.Fatal(err)
			}

			after, err := plans.NewDynamicValue(test.After, test.After.Type())
			if err != nil {
				t.Fatal(err)
			}
			testChange := &plans.ChangesSrc{
				Resources: []*plans.ResourceInstanceChangeSrc{
					{
						Addr: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "test_thing",
							Name: "example",
						}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
						ProviderAddr: addrs.AbsProviderConfig{
							Provider: addrs.NewDefaultProvider("test"),
							Module:   addrs.RootModule,
						},
						ChangeSrc: plans.ChangeSrc{
							Action: test.Action,
							Before: before,
							After:  after,
						},
					},
				},
			}

			ris := testResourceAddrs()

			got, err := marshalPlanResources(testChange, ris, testSchemas())
			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			eq := reflect.DeepEqual(got, test.Want)
			if !eq {
				t.Fatalf("wrong result:\nGot: %#v\nWant: %#v\n", got, test.Want)
			}
		})
	}
}

func TestMarshalPlanValuesNoopDeposed(t *testing.T) {
	dynamicNull, err := plans.NewDynamicValue(cty.NullVal(cty.DynamicPseudoType), cty.DynamicPseudoType)
	if err != nil {
		t.Fatal(err)
	}
	testChange := &plans.ChangesSrc{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_thing",
					Name: "example",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				DeposedKey: "12345678",
				ProviderAddr: addrs.AbsProviderConfig{
					Provider: addrs.NewDefaultProvider("test"),
					Module:   addrs.RootModule,
				},
				ChangeSrc: plans.ChangeSrc{
					Action: plans.NoOp,
					Before: dynamicNull,
					After:  dynamicNull,
				},
			},
		},
	}

	_, err = marshalPlannedValues(testChange, testSchemas())
	if err != nil {
		t.Fatal(err)
	}
}

func testSchemas() *terraform.Schemas {
	return &terraform.Schemas{
		Providers: map[addrs.Provider]providers.ProviderSchema{
			addrs.NewDefaultProvider("test"): providers.ProviderSchema{
				ResourceTypes: map[string]providers.Schema{
					"test_thing": {
						Version: 1,
						Block: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"woozles": {Type: cty.String, Optional: true, Computed: true},
								"foozles": {Type: cty.String, Optional: true},
							},
						},
					},
				},
			},
		},
	}
}

func testResourceAddrs() []addrs.AbsResourceInstance {
	return []addrs.AbsResourceInstance{
		mustAddr("test_thing.example"),
	}
}

func mustAddr(str string) addrs.AbsResourceInstance {
	addr, diags := addrs.ParseAbsResourceInstanceStr(str)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return addr
}
