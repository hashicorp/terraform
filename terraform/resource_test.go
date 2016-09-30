package terraform

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/config"
)

func TestInstanceInfo(t *testing.T) {
	cases := []struct {
		Info   *InstanceInfo
		Result string
	}{
		{
			&InstanceInfo{
				Id: "foo",
			},
			"foo",
		},
		{
			&InstanceInfo{
				Id:         "foo",
				ModulePath: rootModulePath,
			},
			"foo",
		},
		{
			&InstanceInfo{
				Id:         "foo",
				ModulePath: []string{"root", "consul"},
			},
			"module.consul.foo",
		},
	}

	for i, tc := range cases {
		actual := tc.Info.HumanId()
		if actual != tc.Result {
			t.Fatalf("%d: %s", i, actual)
		}
	}
}

func TestResourceConfigGet(t *testing.T) {
	cases := []struct {
		Config map[string]interface{}
		Vars   map[string]string
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
				"foo": "${var.foo}",
			},
			Key:   "foo",
			Value: "${var.foo}",
		},

		{
			Config: map[string]interface{}{
				"foo": "${var.foo}",
			},
			Vars:  map[string]string{"foo": "bar"},
			Key:   "foo",
			Value: "bar",
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
				vs["var."+k] = ast.Variable{Value: v, Type: ast.TypeString}
			}

			if err := rawC.Interpolate(vs); err != nil {
				t.Fatalf("err: %s", err)
			}
		}

		rc := NewResourceConfig(rawC)
		rc.interpolateForce()

		// Test getting a key
		t.Run(fmt.Sprintf("get-%d", i), func(t *testing.T) {
			v, _ := rc.Get(tc.Key)
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

func TestResourceConfigDeepCopy_nil(t *testing.T) {
	var nilRc *ResourceConfig
	actual := nilRc.DeepCopy()
	if actual != nil {
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

func testResourceConfig(
	t *testing.T, c map[string]interface{}) *ResourceConfig {
	raw, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return NewResourceConfig(raw)
}
