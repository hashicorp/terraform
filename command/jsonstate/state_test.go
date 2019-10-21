package jsonstate

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zclconf/go-cty/cty"
)

func TestMarshalOutputs(t *testing.T) {
	tests := []struct {
		Outputs map[string]*states.OutputValue
		Want    map[string]output
		Err     bool
	}{
		{
			nil,
			nil,
			false,
		},
		{
			map[string]*states.OutputValue{
				"test": {
					Sensitive: true,
					Value:     cty.StringVal("sekret"),
				},
			},
			map[string]output{
				"test": {
					Sensitive: true,
					Value:     json.RawMessage(`"sekret"`),
				},
			},
			false,
		},
		{
			map[string]*states.OutputValue{
				"test": {
					Sensitive: false,
					Value:     cty.StringVal("not_so_sekret"),
				},
			},
			map[string]output{
				"test": {
					Sensitive: false,
					Value:     json.RawMessage(`"not_so_sekret"`),
				},
			},
			false,
		},
	}

	for _, test := range tests {
		got, err := marshalOutputs(test.Outputs)
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
			// printing the output isn't terribly useful, but it does help indicate which test case failed
			t.Fatalf("wrong result:\nGot: %#v\nWant: %#v\n", got, test.Want)
		}
	}
}

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

func TestMarshalResources(t *testing.T) {
	deposedKey := states.NewDeposedKey()
	tests := map[string]struct {
		Resources map[string]*states.Resource
		Schemas   *terraform.Schemas
		Want      []resource
		Err       bool
	}{
		"nil": {
			nil,
			nil,
			nil,
			false,
		},
		"single resource": {
			map[string]*states.Resource{
				"test_thing.baz": {
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "bar",
					},
					EachMode: states.EachList,
					Instances: map[addrs.InstanceKey]*states.ResourceInstance{
						addrs.IntKey(0): {
							Current: &states.ResourceInstanceObjectSrc{
								SchemaVersion: 1,
								Status:        states.ObjectReady,
								AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
							},
						},
					},
					ProviderConfig: addrs.ProviderConfig{
						Type: "test",
					}.Absolute(addrs.RootModuleInstance),
				},
			},
			testSchemas(),
			[]resource{
				resource{
					Address:       "test_thing.bar",
					Mode:          "managed",
					Type:          "test_thing",
					Name:          "bar",
					Index:         addrs.IntKey(0),
					ProviderName:  "test",
					SchemaVersion: 1,
					AttributeValues: attributeValues{
						"foozles": json.RawMessage(`null`),
						"woozles": json.RawMessage(`"confuzles"`),
					},
				},
			},
			false,
		},
		"deposed resource": {
			map[string]*states.Resource{
				"test_thing.baz": {
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "bar",
					},
					EachMode: states.EachList,
					Instances: map[addrs.InstanceKey]*states.ResourceInstance{
						addrs.IntKey(0): {
							Deposed: map[states.DeposedKey]*states.ResourceInstanceObjectSrc{
								states.DeposedKey(deposedKey): &states.ResourceInstanceObjectSrc{
									SchemaVersion: 1,
									Status:        states.ObjectReady,
									AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
								},
							},
						},
					},
					ProviderConfig: addrs.ProviderConfig{
						Type: "test",
					}.Absolute(addrs.RootModuleInstance),
				},
			},
			testSchemas(),
			[]resource{
				resource{
					Address:      "test_thing.bar",
					Mode:         "managed",
					Type:         "test_thing",
					Name:         "bar",
					Index:        addrs.IntKey(0),
					ProviderName: "test",
					DeposedKey:   deposedKey.String(),
					AttributeValues: attributeValues{
						"foozles": json.RawMessage(`null`),
						"woozles": json.RawMessage(`"confuzles"`),
					},
				},
			},
			false,
		},
		"deposed and current resource": {
			map[string]*states.Resource{
				"test_thing.baz": {
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "bar",
					},
					EachMode: states.EachList,
					Instances: map[addrs.InstanceKey]*states.ResourceInstance{
						addrs.IntKey(0): {
							Deposed: map[states.DeposedKey]*states.ResourceInstanceObjectSrc{
								states.DeposedKey(deposedKey): &states.ResourceInstanceObjectSrc{
									SchemaVersion: 1,
									Status:        states.ObjectReady,
									AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
								},
							},
							Current: &states.ResourceInstanceObjectSrc{
								SchemaVersion: 1,
								Status:        states.ObjectReady,
								AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
							},
						},
					},
					ProviderConfig: addrs.ProviderConfig{
						Type: "test",
					}.Absolute(addrs.RootModuleInstance),
				},
			},
			testSchemas(),
			[]resource{
				resource{
					Address:       "test_thing.bar",
					Mode:          "managed",
					Type:          "test_thing",
					Name:          "bar",
					Index:         addrs.IntKey(0),
					ProviderName:  "test",
					SchemaVersion: 1,
					AttributeValues: attributeValues{
						"foozles": json.RawMessage(`null`),
						"woozles": json.RawMessage(`"confuzles"`),
					},
				},
				resource{
					Address:      "test_thing.bar",
					Mode:         "managed",
					Type:         "test_thing",
					Name:         "bar",
					Index:        addrs.IntKey(0),
					ProviderName: "test",
					DeposedKey:   deposedKey.String(),
					AttributeValues: attributeValues{
						"foozles": json.RawMessage(`null`),
						"woozles": json.RawMessage(`"confuzles"`),
					},
				},
			},
			false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := marshalResources(test.Resources, test.Schemas)
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

func testSchemas() *terraform.Schemas {
	return &terraform.Schemas{
		Providers: map[string]*terraform.ProviderSchema{
			"test": &terraform.ProviderSchema{
				ResourceTypes: map[string]*configschema.Block{
					"test_thing": {
						Attributes: map[string]*configschema.Attribute{
							"woozles": {Type: cty.String, Optional: true, Computed: true},
							"foozles": {Type: cty.String, Optional: true},
						},
					},
				},
			},
		},
	}
}
