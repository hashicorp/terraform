package terraform

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/mitchellh/reflectwalk"
)

func TestInstanceInfoResourceAddress(t *testing.T) {
	tests := []struct {
		Input *InstanceInfo
		Want  string
	}{
		{
			&InstanceInfo{
				Id: "test_resource.baz",
			},
			"test_resource.baz",
		},
		{
			&InstanceInfo{
				Id:         "test_resource.baz",
				ModulePath: rootModulePath,
			},
			"test_resource.baz",
		},
		{
			&InstanceInfo{
				Id:         "test_resource.baz",
				ModulePath: []string{"root", "foo"},
			},
			"module.foo.test_resource.baz",
		},
		{
			&InstanceInfo{
				Id:         "test_resource.baz",
				ModulePath: []string{"root", "foo", "bar"},
			},
			"module.foo.module.bar.test_resource.baz",
		},
		{
			&InstanceInfo{
				Id: "test_resource.baz (tainted)",
			},
			"test_resource.baz.tainted",
		},
		{
			&InstanceInfo{
				Id: "test_resource.baz (deposed #0)",
			},
			"test_resource.baz.deposed",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			gotAddr := test.Input.ResourceAddress()
			got := gotAddr.String()
			if got != test.Want {
				t.Fatalf("wrong result\ngot:  %s\nwant: %s", got, test.Want)
			}
		})
	}
}

func TestResourceConfigGet(t *testing.T) {
	fooStringSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {Type: cty.String, Optional: true},
		},
	}
	fooListSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {Type: cty.List(cty.Number), Optional: true},
		},
	}

	cases := []struct {
		Config cty.Value
		Schema *configschema.Block
		Key    string
		Value  interface{}
	}{
		{
			Config: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			}),
			Schema: fooStringSchema,
			Key:    "foo",
			Value:  "bar",
		},

		{
			Config: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.UnknownVal(cty.String),
			}),
			Schema: fooStringSchema,
			Key:    "foo",
			Value:  hcl2shim.UnknownVariableValue,
		},

		{
			Config: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
					cty.NumberIntVal(5),
				}),
			}),
			Schema: fooListSchema,
			Key:    "foo.0",
			Value:  1,
		},

		{
			Config: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
					cty.NumberIntVal(5),
				}),
			}),
			Schema: fooListSchema,
			Key:    "foo.5",
			Value:  nil,
		},

		{
			Config: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
					cty.NumberIntVal(5),
				}),
			}),
			Schema: fooListSchema,
			Key:    "foo.-1",
			Value:  nil,
		},

		// get from map
		{
			Config: cty.ObjectVal(map[string]cty.Value{
				"mapname": cty.ListVal([]cty.Value{
					cty.MapVal(map[string]cty.Value{
						"key": cty.NumberIntVal(1),
					}),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"mapname": {Type: cty.List(cty.Map(cty.Number)), Optional: true},
				},
			},
			Key:   "mapname.0.key",
			Value: 1,
		},

		// get from map with dot in key
		{
			Config: cty.ObjectVal(map[string]cty.Value{
				"mapname": cty.ListVal([]cty.Value{
					cty.MapVal(map[string]cty.Value{
						"key.name": cty.NumberIntVal(1),
					}),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"mapname": {Type: cty.List(cty.Map(cty.Number)), Optional: true},
				},
			},
			Key:   "mapname.0.key.name",
			Value: 1,
		},

		// get from map with overlapping key names
		{
			Config: cty.ObjectVal(map[string]cty.Value{
				"mapname": cty.ListVal([]cty.Value{
					cty.MapVal(map[string]cty.Value{
						"key.name":   cty.NumberIntVal(1),
						"key.name.2": cty.NumberIntVal(2),
					}),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"mapname": {Type: cty.List(cty.Map(cty.Number)), Optional: true},
				},
			},
			Key:   "mapname.0.key.name.2",
			Value: 2,
		},
		{
			Config: cty.ObjectVal(map[string]cty.Value{
				"mapname": cty.ListVal([]cty.Value{
					cty.MapVal(map[string]cty.Value{
						"key.name":     cty.NumberIntVal(1),
						"key.name.foo": cty.NumberIntVal(2),
					}),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"mapname": {Type: cty.List(cty.Map(cty.Number)), Optional: true},
				},
			},
			Key:   "mapname.0.key.name",
			Value: 1,
		},
		{
			Config: cty.ObjectVal(map[string]cty.Value{
				"mapname": cty.ListVal([]cty.Value{
					cty.MapVal(map[string]cty.Value{
						"listkey": cty.ListVal([]cty.Value{
							cty.MapVal(map[string]cty.Value{
								"key": cty.NumberIntVal(3),
							}),
						}),
					}),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"mapname": {Type: cty.List(cty.Map(cty.List(cty.Map(cty.Number)))), Optional: true},
				},
			},
			Key:   "mapname.0.listkey.0.key",
			Value: 3,
		},
	}

	for i, tc := range cases {
		rc := NewResourceConfigShimmed(tc.Config, tc.Schema)

		// Test getting a key
		t.Run(fmt.Sprintf("get-%d", i), func(t *testing.T) {
			v, ok := rc.Get(tc.Key)
			if ok && v == nil {
				t.Fatal("(nil, true) returned from Get")
			}

			if !reflect.DeepEqual(v, tc.Value) {
				t.Fatalf("%d bad: %#v", i, v)
			}
		})

		// Test copying and equality
		t.Run(fmt.Sprintf("copy-and-equal-%d", i), func(t *testing.T) {
			copy := rc.DeepCopy()
			if !reflect.DeepEqual(copy, rc) {
				t.Fatalf("bad:\n\n%#v\n\n%#v", copy, rc)
			}

			if !copy.Equal(rc) {
				t.Fatalf("copy != rc:\n\n%#v\n\n%#v", copy, rc)
			}
			if !rc.Equal(copy) {
				t.Fatalf("rc != copy:\n\n%#v\n\n%#v", copy, rc)
			}
		})
	}
}

func TestResourceConfigDeepCopy_nil(t *testing.T) {
	var nilRc *ResourceConfig
	actual := nilRc.DeepCopy()
	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceConfigDeepCopy_nilComputed(t *testing.T) {
	rc := &ResourceConfig{}
	actual := rc.DeepCopy()
	if actual.ComputedKeys != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceConfigEqual_nil(t *testing.T) {
	var nilRc *ResourceConfig
	notNil := NewResourceConfigShimmed(cty.EmptyObjectVal, &configschema.Block{})

	if nilRc.Equal(notNil) {
		t.Fatal("should not be equal")
	}

	if notNil.Equal(nilRc) {
		t.Fatal("should not be equal")
	}
}

func TestResourceConfigEqual_computedKeyOrder(t *testing.T) {
	v := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.UnknownVal(cty.String),
	})
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {Type: cty.String, Optional: true},
		},
	}
	rc := NewResourceConfigShimmed(v, schema)
	rc2 := NewResourceConfigShimmed(v, schema)

	// Set the computed keys manually to force ordering to differ
	rc.ComputedKeys = []string{"foo", "bar"}
	rc2.ComputedKeys = []string{"bar", "foo"}

	if !rc.Equal(rc2) {
		t.Fatal("should be equal")
	}
}

func TestUnknownCheckWalker(t *testing.T) {
	cases := []struct {
		Name   string
		Input  interface{}
		Result bool
	}{
		{
			"primitive",
			42,
			false,
		},

		{
			"primitive computed",
			hcl2shim.UnknownVariableValue,
			true,
		},

		{
			"list",
			[]interface{}{"foo", hcl2shim.UnknownVariableValue},
			true,
		},

		{
			"nested list",
			[]interface{}{
				"foo",
				[]interface{}{hcl2shim.UnknownVariableValue},
			},
			true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			var w unknownCheckWalker
			if err := reflectwalk.Walk(tc.Input, &w); err != nil {
				t.Fatalf("err: %s", err)
			}

			if w.Unknown != tc.Result {
				t.Fatalf("bad: %v", w.Unknown)
			}
		})
	}
}

func TestNewResourceConfigShimmed(t *testing.T) {
	for _, tc := range []struct {
		Name     string
		Val      cty.Value
		Schema   *configschema.Block
		Expected *ResourceConfig
	}{
		{
			Name: "empty object",
			Val:  cty.NullVal(cty.EmptyObject),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			Expected: &ResourceConfig{
				Raw:    map[string]interface{}{},
				Config: map[string]interface{}{},
			},
		},
		{
			Name: "basic",
			Val: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			Expected: &ResourceConfig{
				Raw: map[string]interface{}{
					"foo": "bar",
				},
				Config: map[string]interface{}{
					"foo": "bar",
				},
			},
		},
		{
			Name: "null string",
			Val: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			Expected: &ResourceConfig{
				Raw:    map[string]interface{}{},
				Config: map[string]interface{}{},
			},
		},
		{
			Name: "unknown string",
			Val: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.UnknownVal(cty.String),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			Expected: &ResourceConfig{
				ComputedKeys: []string{"foo"},
				Raw: map[string]interface{}{
					"foo": hcl2shim.UnknownVariableValue,
				},
				Config: map[string]interface{}{
					"foo": hcl2shim.UnknownVariableValue,
				},
			},
		},
		{
			Name: "unknown collections",
			Val: cty.ObjectVal(map[string]cty.Value{
				"bar": cty.UnknownVal(cty.Map(cty.String)),
				"baz": cty.UnknownVal(cty.List(cty.String)),
			}),
			Schema: &configschema.Block{
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
			Expected: &ResourceConfig{
				ComputedKeys: []string{"bar", "baz"},
				Raw: map[string]interface{}{
					"bar": hcl2shim.UnknownVariableValue,
					"baz": hcl2shim.UnknownVariableValue,
				},
				Config: map[string]interface{}{
					"bar": hcl2shim.UnknownVariableValue,
					"baz": hcl2shim.UnknownVariableValue,
				},
			},
		},
		{
			Name: "null collections",
			Val: cty.ObjectVal(map[string]cty.Value{
				"bar": cty.NullVal(cty.Map(cty.String)),
				"baz": cty.NullVal(cty.List(cty.String)),
			}),
			Schema: &configschema.Block{
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
			Expected: &ResourceConfig{
				Raw:    map[string]interface{}{},
				Config: map[string]interface{}{},
			},
		},
		{
			Name: "unknown blocks",
			Val: cty.ObjectVal(map[string]cty.Value{
				"bar": cty.UnknownVal(cty.Map(cty.String)),
				"baz": cty.UnknownVal(cty.List(cty.String)),
			}),
			Schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"bar": {
						Block:   configschema.Block{},
						Nesting: configschema.NestingList,
					},
					"baz": {
						Block:   configschema.Block{},
						Nesting: configschema.NestingSet,
					},
				},
			},
			Expected: &ResourceConfig{
				ComputedKeys: []string{"bar", "baz"},
				Raw: map[string]interface{}{
					"bar": hcl2shim.UnknownVariableValue,
					"baz": hcl2shim.UnknownVariableValue,
				},
				Config: map[string]interface{}{
					"bar": hcl2shim.UnknownVariableValue,
					"baz": hcl2shim.UnknownVariableValue,
				},
			},
		},
		{
			Name: "unknown in nested blocks",
			Val: cty.ObjectVal(map[string]cty.Value{
				"bar": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"baz": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"list": cty.UnknownVal(cty.List(cty.String)),
							}),
						}),
					}),
				}),
			}),
			Schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"bar": {
						Block: configschema.Block{
							BlockTypes: map[string]*configschema.NestedBlock{
								"baz": {
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"list": {Type: cty.List(cty.String),
												Optional: true,
											},
										},
									},
									Nesting: configschema.NestingList,
								},
							},
						},
						Nesting: configschema.NestingList,
					},
				},
			},
			Expected: &ResourceConfig{
				ComputedKeys: []string{"bar.0.baz.0.list"},
				Raw: map[string]interface{}{
					"bar": []interface{}{map[string]interface{}{
						"baz": []interface{}{map[string]interface{}{
							"list": "74D93920-ED26-11E3-AC10-0800200C9A66",
						}},
					}},
				},
				Config: map[string]interface{}{
					"bar": []interface{}{map[string]interface{}{
						"baz": []interface{}{map[string]interface{}{
							"list": "74D93920-ED26-11E3-AC10-0800200C9A66",
						}},
					}},
				},
			},
		},
		{
			Name: "unknown in set",
			Val: cty.ObjectVal(map[string]cty.Value{
				"bar": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"val": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			Schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"bar": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"val": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
						Nesting: configschema.NestingSet,
					},
				},
			},
			Expected: &ResourceConfig{
				ComputedKeys: []string{"bar.0.val"},
				Raw: map[string]interface{}{
					"bar": []interface{}{map[string]interface{}{
						"val": "74D93920-ED26-11E3-AC10-0800200C9A66",
					}},
				},
				Config: map[string]interface{}{
					"bar": []interface{}{map[string]interface{}{
						"val": "74D93920-ED26-11E3-AC10-0800200C9A66",
					}},
				},
			},
		},
		{
			Name: "unknown in attribute sets",
			Val: cty.ObjectVal(map[string]cty.Value{
				"bar": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"val": cty.UnknownVal(cty.String),
					}),
				}),
				"baz": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.UnknownVal(cty.Object(map[string]cty.Type{
							"attr": cty.List(cty.String),
						})),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"obj": cty.ObjectVal(map[string]cty.Value{
							"attr": cty.UnknownVal(cty.List(cty.String)),
						}),
					}),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"bar": &configschema.Attribute{
						Type: cty.Set(cty.Object(map[string]cty.Type{
							"val": cty.String,
						})),
					},
					"baz": &configschema.Attribute{
						Type: cty.Set(cty.Object(map[string]cty.Type{
							"obj": cty.Object(map[string]cty.Type{
								"attr": cty.List(cty.String),
							}),
						})),
					},
				},
			},
			Expected: &ResourceConfig{
				ComputedKeys: []string{"bar.0.val", "baz.0.obj.attr", "baz.1.obj"},
				Raw: map[string]interface{}{
					"bar": []interface{}{map[string]interface{}{
						"val": "74D93920-ED26-11E3-AC10-0800200C9A66",
					}},
					"baz": []interface{}{
						map[string]interface{}{
							"obj": map[string]interface{}{
								"attr": "74D93920-ED26-11E3-AC10-0800200C9A66",
							},
						},
						map[string]interface{}{
							"obj": "74D93920-ED26-11E3-AC10-0800200C9A66",
						},
					},
				},
				Config: map[string]interface{}{
					"bar": []interface{}{map[string]interface{}{
						"val": "74D93920-ED26-11E3-AC10-0800200C9A66",
					}},
					"baz": []interface{}{
						map[string]interface{}{
							"obj": map[string]interface{}{
								"attr": "74D93920-ED26-11E3-AC10-0800200C9A66",
							},
						},
						map[string]interface{}{
							"obj": "74D93920-ED26-11E3-AC10-0800200C9A66",
						},
					},
				},
			},
		},
		{
			Name: "null blocks",
			Val: cty.ObjectVal(map[string]cty.Value{
				"bar": cty.NullVal(cty.Map(cty.String)),
				"baz": cty.NullVal(cty.List(cty.String)),
			}),
			Schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"bar": {
						Block:   configschema.Block{},
						Nesting: configschema.NestingMap,
					},
					"baz": {
						Block:   configschema.Block{},
						Nesting: configschema.NestingSingle,
					},
				},
			},
			Expected: &ResourceConfig{
				Raw:    map[string]interface{}{},
				Config: map[string]interface{}{},
			},
		},
	} {
		t.Run(tc.Name, func(*testing.T) {
			cfg := NewResourceConfigShimmed(tc.Val, tc.Schema)
			if !tc.Expected.Equal(cfg) {
				t.Fatalf("expected:\n%#v\ngot:\n%#v", tc.Expected, cfg)
			}
		})
	}
}
