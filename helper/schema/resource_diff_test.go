package schema

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/terraform"
)

// testSetFunc is a very simple function we use to test a foo/bar complex set.
// Both "foo" and "bar" are int values.
//
// This is not foolproof as since it performs sums, you can run into
// collisions. Spec tests accordingly. :P
func testSetFunc(v interface{}) int {
	m := v.(map[string]interface{})
	return m["foo"].(int) + m["bar"].(int)
}

func TestSetNew(t *testing.T) {
	testCases := []struct {
		Name          string
		Schema        map[string]*Schema
		State         *terraform.InstanceState
		Config        *terraform.ResourceConfig
		Diff          *terraform.InstanceDiff
		Key           string
		NewValue      interface{}
		Expected      *terraform.InstanceDiff
		ExpectedError bool
	}{
		{
			Name: "basic primitive diff",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo": "bar",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"foo": "baz",
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old: "bar",
						New: "baz",
					},
				},
			},
			Key:      "foo",
			NewValue: "qux",
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old: "bar",
						New: "qux",
					},
				},
			},
		},
		{
			Name: "basic set diff",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeString},
					Set:      HashString,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo.#":          "1",
					"foo.1996459178": "bar",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"foo": []interface{}{"baz"},
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo.1996459178": &terraform.ResourceAttrDiff{
						Old:        "bar",
						New:        "",
						NewRemoved: true,
					},
					"foo.2015626392": &terraform.ResourceAttrDiff{
						Old: "",
						New: "baz",
					},
				},
			},
			Key:      "foo",
			NewValue: []interface{}{"qux"},
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo.1996459178": &terraform.ResourceAttrDiff{
						Old:        "bar",
						New:        "",
						NewRemoved: true,
					},
					"foo.2800005064": &terraform.ResourceAttrDiff{
						Old: "",
						New: "qux",
					},
				},
			},
		},
		{
			Name: "basic list diff",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeList,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeString},
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo.#": "1",
					"foo.0": "bar",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"foo": []interface{}{"baz"},
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo.0": &terraform.ResourceAttrDiff{
						Old: "bar",
						New: "baz",
					},
				},
			},
			Key:      "foo",
			NewValue: []interface{}{"qux"},
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo.0": &terraform.ResourceAttrDiff{
						Old: "bar",
						New: "qux",
					},
				},
			},
		},
		{
			Name: "basic map diff",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeMap,
					Optional: true,
					Computed: true,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo.%":   "1",
					"foo.bar": "baz",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"foo": map[string]interface{}{"bar": "qux"},
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo.bar": &terraform.ResourceAttrDiff{
						Old: "baz",
						New: "qux",
					},
				},
			},
			Key:      "foo",
			NewValue: map[string]interface{}{"bar": "quux"},
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo.bar": &terraform.ResourceAttrDiff{
						Old: "baz",
						New: "quux",
					},
				},
			},
		},
		{
			Name: "additional diff with primitive",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeString,
					Optional: true,
				},
				"one": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo": "bar",
					"one": "two",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"foo": "baz",
				"one": "three",
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old: "bar",
						New: "baz",
					},
					"one": &terraform.ResourceAttrDiff{
						Old: "two",
						New: "three",
					},
				},
			},
			Key:      "one",
			NewValue: "four",
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old: "bar",
						New: "baz",
					},
					"one": &terraform.ResourceAttrDiff{
						Old: "two",
						New: "four",
					},
				},
			},
		},
		{
			Name: "additional diff with primitive computed only",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeString,
					Optional: true,
				},
				"one": &Schema{
					Type:     TypeString,
					Computed: true,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo": "bar",
					"one": "two",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"foo": "baz",
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old: "bar",
						New: "baz",
					},
				},
			},
			Key:      "one",
			NewValue: "three",
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old: "bar",
						New: "baz",
					},
					"one": &terraform.ResourceAttrDiff{
						Old: "two",
						New: "three",
					},
				},
			},
		},
		{
			Name: "complex-ish set diff",
			Schema: map[string]*Schema{
				"top": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"foo": &Schema{
								Type:     TypeInt,
								Optional: true,
								Computed: true,
							},
							"bar": &Schema{
								Type:     TypeInt,
								Optional: true,
								Computed: true,
							},
						},
					},
					Set: testSetFunc,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"top.#":      "2",
					"top.3.foo":  "1",
					"top.3.bar":  "2",
					"top.23.foo": "11",
					"top.23.bar": "12",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"top": []interface{}{
					map[string]interface{}{
						"foo": 1,
						"bar": 3,
					},
					map[string]interface{}{
						"foo": 12,
						"bar": 12,
					},
				},
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"top.4.foo": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"top.4.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "3",
					},
					"top.24.foo": &terraform.ResourceAttrDiff{
						Old: "",
						New: "12",
					},
					"top.24.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "12",
					},
				},
			},
			Key: "top",
			NewValue: NewSet(testSetFunc, []interface{}{
				map[string]interface{}{
					"foo": 1,
					"bar": 4,
				},
				map[string]interface{}{
					"foo": 13,
					"bar": 12,
				},
				map[string]interface{}{
					"foo": 21,
					"bar": 22,
				},
			}),
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"top.#": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "3",
					},
					"top.5.foo": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"top.5.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "4",
					},
					"top.25.foo": &terraform.ResourceAttrDiff{
						Old: "",
						New: "13",
					},
					"top.25.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "12",
					},
					"top.43.foo": &terraform.ResourceAttrDiff{
						Old: "",
						New: "21",
					},
					"top.43.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "22",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s", tc.Name), func(t *testing.T) {
			m := schemaMap(tc.Schema)
			d := newResourceDiff(tc.Schema, nil, tc.State, tc.Diff)
			if err := d.SetNew(tc.Key, tc.NewValue); err != nil {
				t.Fatalf("bad: %s", err)
			}
			for _, k := range d.UpdatedKeys() {
				if err := m.diff(k, m[k], tc.Diff, d, false); err != nil {
					t.Fatalf("bad: %s", err)
				}
			}
			if !reflect.DeepEqual(tc.Expected, tc.Diff) {
				t.Fatalf("Expected %s, got %s", spew.Sdump(tc.Expected), spew.Sdump(tc.Diff))
			}
		})
	}
}
