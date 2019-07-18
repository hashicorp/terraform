package terraform

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hil"
	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/config"
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
	cases := []struct {
		Config map[string]interface{}
		Vars   map[string]interface{}
		Key    string
		Value  interface{}
	}{
		{
			Config: nil,
			Key:    "foo",
			Value:  nil,
		},

		{
			Config: map[string]interface{}{
				"foo": "bar",
			},
			Key:   "foo",
			Value: "bar",
		},

		{
			Config: map[string]interface{}{
				"foo": "${var.foo}",
			},
			Key:   "foo",
			Value: "${var.foo}",
		},

		{
			Config: map[string]interface{}{
				"foo": "${var.foo}",
			},
			Vars:  map[string]interface{}{"foo": unknownValue()},
			Key:   "foo",
			Value: "${var.foo}",
		},

		{
			Config: map[string]interface{}{
				"foo": []interface{}{1, 2, 5},
			},
			Key:   "foo.0",
			Value: 1,
		},

		{
			Config: map[string]interface{}{
				"foo": []interface{}{1, 2, 5},
			},
			Key:   "foo.5",
			Value: nil,
		},

		{
			Config: map[string]interface{}{
				"foo": []interface{}{1, 2, 5},
			},
			Key:   "foo.-1",
			Value: nil,
		},

		// get from map
		{
			Config: map[string]interface{}{
				"mapname": []map[string]interface{}{
					map[string]interface{}{"key": 1},
				},
			},
			Key:   "mapname.0.key",
			Value: 1,
		},

		// get from map with dot in key
		{
			Config: map[string]interface{}{
				"mapname": []map[string]interface{}{
					map[string]interface{}{"key.name": 1},
				},
			},
			Key:   "mapname.0.key.name",
			Value: 1,
		},

		// get from map with overlapping key names
		{
			Config: map[string]interface{}{
				"mapname": []map[string]interface{}{
					map[string]interface{}{
						"key.name":   1,
						"key.name.2": 2,
					},
				},
			},
			Key:   "mapname.0.key.name.2",
			Value: 2,
		},
		{
			Config: map[string]interface{}{
				"mapname": []map[string]interface{}{
					map[string]interface{}{
						"key.name":     1,
						"key.name.foo": 2,
					},
				},
			},
			Key:   "mapname.0.key.name",
			Value: 1,
		},
		{
			Config: map[string]interface{}{
				"mapname": []map[string]interface{}{
					map[string]interface{}{
						"listkey": []map[string]interface{}{
							{"key": 3},
						},
					},
				},
			},
			Key:   "mapname.0.listkey.0.key",
			Value: 3,
		},

		// A map assigned to a list via interpolation should Get a non-existent
		// value. The test code now also checks that Get doesn't return (nil,
		// true), which it previously did for this configuration.
		{
			Config: map[string]interface{}{
				"maplist": "${var.maplist}",
			},
			Key:   "maplist.0",
			Value: nil,
		},

		// Reference list of maps variable.
		// This does not work from GetRaw.
		{
			Vars: map[string]interface{}{
				"maplist": []interface{}{
					map[string]interface{}{
						"key": "a",
					},
					map[string]interface{}{
						"key": "b",
					},
				},
			},
			Config: map[string]interface{}{
				"maplist": "${var.maplist}",
			},
			Key:   "maplist.0",
			Value: map[string]interface{}{"key": "a"},
		},

		// Reference a map-of-lists variable.
		// This does not work from GetRaw.
		{
			Vars: map[string]interface{}{
				"listmap": map[string]interface{}{
					"key1": []interface{}{"a", "b"},
					"key2": []interface{}{"c", "d"},
				},
			},
			Config: map[string]interface{}{
				"listmap": "${var.listmap}",
			},
			Key:   "listmap.key1",
			Value: []interface{}{"a", "b"},
		},

		// FIXME: this is ambiguous, and matches the nested map
		//        leaving here to catch this behaviour if it changes.
		{
			Config: map[string]interface{}{
				"mapname": []map[string]interface{}{
					map[string]interface{}{
						"key.name":   1,
						"key.name.0": 2,
						"key":        map[string]interface{}{"name": 3},
					},
				},
			},
			Key:   "mapname.0.key.name",
			Value: 3,
		},
		/*
			// TODO: can't access this nested list at all.
			// FIXME: key with name matching substring of nested list can panic
			{
				Config: map[string]interface{}{
					"mapname": []map[string]interface{}{
						map[string]interface{}{
							"key.name": []map[string]interface{}{
								{"subkey": 1},
							},
							"key": 3,
						},
					},
				},
				Key:   "mapname.0.key.name.0.subkey",
				Value: 3,
			},
		*/
	}

	for i, tc := range cases {
		var rawC *config.RawConfig
		if tc.Config != nil {
			var err error
			rawC, err = config.NewRawConfig(tc.Config)
			if err != nil {
				t.Fatalf("err: %s", err)
			}
		}

		if tc.Vars != nil {
			vs := make(map[string]ast.Variable)
			for k, v := range tc.Vars {
				hilVar, err := hil.InterfaceToVariable(v)
				if err != nil {
					t.Fatalf("%#v to var: %s", v, err)
				}

				vs["var."+k] = hilVar
			}

			if err := rawC.Interpolate(vs); err != nil {
				t.Fatalf("err: %s", err)
			}
		}

		rc := NewResourceConfig(rawC)
		rc.interpolateForce()

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

		// If we have vars, we don't test copying
		if len(tc.Vars) > 0 {
			continue
		}

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

func TestResourceConfigGetRaw(t *testing.T) {
	cases := []struct {
		Config map[string]interface{}
		Vars   map[string]interface{}
		Key    string
		Value  interface{}
	}{
		// Referencing a list-of-maps variable doesn't work from GetRaw.
		// The ConfigFieldReader currently catches this case and looks up the
		// variable in the config.
		{
			Vars: map[string]interface{}{
				"maplist": []interface{}{
					map[string]interface{}{
						"key": "a",
					},
					map[string]interface{}{
						"key": "b",
					},
				},
			},
			Config: map[string]interface{}{
				"maplist": "${var.maplist}",
			},
			Key:   "maplist.0",
			Value: nil,
		},
		// Reference a map-of-lists variable.
		// The ConfigFieldReader currently catches this case and looks up the
		// variable in the config.
		{
			Vars: map[string]interface{}{
				"listmap": map[string]interface{}{
					"key1": []interface{}{"a", "b"},
					"key2": []interface{}{"c", "d"},
				},
			},
			Config: map[string]interface{}{
				"listmap": "${var.listmap}",
			},
			Key:   "listmap.key1",
			Value: nil,
		},
	}

	for i, tc := range cases {
		var rawC *config.RawConfig
		if tc.Config != nil {
			var err error
			rawC, err = config.NewRawConfig(tc.Config)
			if err != nil {
				t.Fatalf("err: %s", err)
			}
		}

		if tc.Vars != nil {
			vs := make(map[string]ast.Variable)
			for k, v := range tc.Vars {
				hilVar, err := hil.InterfaceToVariable(v)
				if err != nil {
					t.Fatalf("%#v to var: %s", v, err)
				}
				vs["var."+k] = hilVar
			}
			if err := rawC.Interpolate(vs); err != nil {
				t.Fatalf("err: %s", err)
			}
		}

		rc := NewResourceConfig(rawC)
		rc.interpolateForce()

		// Test getting a key
		t.Run(fmt.Sprintf("get-%d", i), func(t *testing.T) {
			v, ok := rc.GetRaw(tc.Key)
			if ok && v == nil {
				t.Fatal("(nil, true) returned from GetRaw")
			}

			if !reflect.DeepEqual(v, tc.Value) {
				t.Fatalf("%d bad: %#v", i, v)
			}
		})
	}
}

func TestResourceConfigIsComputed(t *testing.T) {
	cases := []struct {
		Name   string
		Config map[string]interface{}
		Vars   map[string]interface{}
		Key    string
		Result bool
	}{
		{
			Name: "basic value",
			Config: map[string]interface{}{
				"foo": "${var.foo}",
			},
			Vars: map[string]interface{}{
				"foo": unknownValue(),
			},
			Key:    "foo",
			Result: true,
		},

		{
			Name: "set with a computed element",
			Config: map[string]interface{}{
				"foo": "${var.foo}",
			},
			Vars: map[string]interface{}{
				"foo": []string{
					"a",
					unknownValue(),
				},
			},
			Key:    "foo",
			Result: true,
		},

		{
			Name: "set with no computed elements",
			Config: map[string]interface{}{
				"foo": "${var.foo}",
			},
			Vars: map[string]interface{}{
				"foo": []string{
					"a",
					"b",
				},
			},
			Key:    "foo",
			Result: false,
		},

		/*
			{
				Name: "set count with computed elements",
				Config: map[string]interface{}{
					"foo": "${var.foo}",
				},
				Vars: map[string]interface{}{
					"foo": []string{
						"a",
						unknownValue(),
					},
				},
				Key:    "foo.#",
				Result: true,
			},
		*/

		{
			Name: "set count with computed elements",
			Config: map[string]interface{}{
				"foo": []interface{}{"${var.foo}"},
			},
			Vars: map[string]interface{}{
				"foo": []string{
					"a",
					unknownValue(),
				},
			},
			Key:    "foo.#",
			Result: true,
		},

		{
			Name: "nested set with computed elements",
			Config: map[string]interface{}{
				"route": []map[string]interface{}{
					map[string]interface{}{
						"index":   "1",
						"gateway": []interface{}{"${var.foo}"},
					},
				},
			},
			Vars: map[string]interface{}{
				"foo": unknownValue(),
			},
			Key:    "route.0.gateway",
			Result: true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			var rawC *config.RawConfig
			if tc.Config != nil {
				var err error
				rawC, err = config.NewRawConfig(tc.Config)
				if err != nil {
					t.Fatalf("err: %s", err)
				}
			}

			if tc.Vars != nil {
				vs := make(map[string]ast.Variable)
				for k, v := range tc.Vars {
					hilVar, err := hil.InterfaceToVariable(v)
					if err != nil {
						t.Fatalf("%#v to var: %s", v, err)
					}

					vs["var."+k] = hilVar
				}

				if err := rawC.Interpolate(vs); err != nil {
					t.Fatalf("err: %s", err)
				}
			}

			rc := NewResourceConfig(rawC)
			rc.interpolateForce()

			t.Logf("Config: %#v", rc)

			actual := rc.IsComputed(tc.Key)
			if actual != tc.Result {
				t.Fatalf("bad: %#v", actual)
			}
		})
	}
}

func TestResourceConfigCheckSet(t *testing.T) {
	cases := []struct {
		Name   string
		Config map[string]interface{}
		Vars   map[string]interface{}
		Input  []string
		Errs   bool
	}{
		{
			Name: "computed basic",
			Config: map[string]interface{}{
				"foo": "${var.foo}",
			},
			Vars: map[string]interface{}{
				"foo": unknownValue(),
			},
			Input: []string{"foo"},
			Errs:  false,
		},

		{
			Name: "basic",
			Config: map[string]interface{}{
				"foo": "bar",
			},
			Vars:  nil,
			Input: []string{"foo"},
			Errs:  false,
		},

		{
			Name: "basic with not set",
			Config: map[string]interface{}{
				"foo": "bar",
			},
			Vars:  nil,
			Input: []string{"foo", "bar"},
			Errs:  true,
		},

		{
			Name: "basic with one computed",
			Config: map[string]interface{}{
				"foo": "bar",
				"bar": "${var.foo}",
			},
			Vars: map[string]interface{}{
				"foo": unknownValue(),
			},
			Input: []string{"foo", "bar"},
			Errs:  false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			var rawC *config.RawConfig
			if tc.Config != nil {
				var err error
				rawC, err = config.NewRawConfig(tc.Config)
				if err != nil {
					t.Fatalf("err: %s", err)
				}
			}

			if tc.Vars != nil {
				vs := make(map[string]ast.Variable)
				for k, v := range tc.Vars {
					hilVar, err := hil.InterfaceToVariable(v)
					if err != nil {
						t.Fatalf("%#v to var: %s", v, err)
					}

					vs["var."+k] = hilVar
				}

				if err := rawC.Interpolate(vs); err != nil {
					t.Fatalf("err: %s", err)
				}
			}

			rc := NewResourceConfig(rawC)
			rc.interpolateForce()

			t.Logf("Config: %#v", rc)

			errs := rc.CheckSet(tc.Input)
			if tc.Errs != (len(errs) > 0) {
				t.Fatalf("bad: %#v", errs)
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
	notNil := NewResourceConfig(nil)

	if nilRc.Equal(notNil) {
		t.Fatal("should not be equal")
	}

	if notNil.Equal(nilRc) {
		t.Fatal("should not be equal")
	}
}

func TestResourceConfigEqual_computedKeyOrder(t *testing.T) {
	c := map[string]interface{}{"foo": "${a.b.c}"}
	rc := NewResourceConfig(config.TestRawConfig(t, c))
	rc2 := NewResourceConfig(config.TestRawConfig(t, c))

	// Set the computed keys manual
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
			unknownValue(),
			true,
		},

		{
			"list",
			[]interface{}{"foo", unknownValue()},
			true,
		},

		{
			"nested list",
			[]interface{}{
				"foo",
				[]interface{}{unknownValue()},
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

func testResourceConfig(
	t *testing.T, c map[string]interface{}) *ResourceConfig {
	raw, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return NewResourceConfig(raw)
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
