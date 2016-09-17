package schema

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestDiffFieldReader_impl(t *testing.T) {
	var _ FieldReader = new(DiffFieldReader)
}

func TestDiffFieldReader_NestedSetUpdate(t *testing.T) {
	schema := map[string]*Schema{
		"list_of_sets_1": &Schema{
			Type: TypeList,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"nested_set": &Schema{
						Type: TypeSet,
						Elem: &Resource{
							Schema: map[string]*Schema{
								"val": &Schema{
									Type: TypeInt,
								},
							},
						},
						Set: func(a interface{}) int {
							m := a.(map[string]interface{})
							return m["val"].(int)
						},
					},
				},
			},
		},
		"list_of_sets_2": &Schema{
			Type: TypeList,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"nested_set": &Schema{
						Type: TypeSet,
						Elem: &Resource{
							Schema: map[string]*Schema{
								"val": &Schema{
									Type: TypeInt,
								},
							},
						},
						Set: func(a interface{}) int {
							m := a.(map[string]interface{})
							return m["val"].(int)
						},
					},
				},
			},
		},
	}

	r := &DiffFieldReader{
		Schema: schema,
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"list_of_sets_1.0.nested_set.1.val": &terraform.ResourceAttrDiff{
					Old:        "1",
					New:        "0",
					NewRemoved: true,
				},
				"list_of_sets_1.0.nested_set.2.val": &terraform.ResourceAttrDiff{
					New: "2",
				},
			},
		},
	}

	r.Source = &MultiLevelFieldReader{
		Readers: map[string]FieldReader{
			"diff": r,
			"set":  &MapFieldReader{Schema: schema},
			"state": &MapFieldReader{
				Map: &BasicMapReader{
					"list_of_sets_1.#":                  "1",
					"list_of_sets_1.0.nested_set.#":     "1",
					"list_of_sets_1.0.nested_set.1.val": "1",
					"list_of_sets_2.#":                  "1",
					"list_of_sets_2.0.nested_set.#":     "1",
					"list_of_sets_2.0.nested_set.1.val": "1",
				},
				Schema: schema,
			},
		},
		Levels: []string{"state", "config"},
	}

	out, err := r.ReadField([]string{"list_of_sets_2"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	list, ok := out.Value.([]interface{})
	if !ok {
		t.Fatalf("nestedSetUpdate: bad: Value type\n\nexpected: []interface{}\n\ngot: %v", reflect.TypeOf(out.Value))
	}
	if len(list) != 1 {
		t.Fatalf("nestedSetUpdate: bad: list length\n\nexpected: 1\n\ngot: %v\n\n", len(list))
	}

	var m map[string]interface{}
	m, ok = list[0].(map[string]interface{})
	if !ok {
		t.Fatalf("nestedSetUpdate: bad: list item type\n\nexpected: map[string]interface{}\n\ngot: %v\n\n", reflect.TypeOf(list[0]))
	}

	rawSet := m["nested_set"]
	if rawSet == nil {
		t.Fatalf("nestedSetUpdate: bad: nested_set\n\nexpected: not nil\n\ngot: nil\n\n")
	}

	var set *Set
	set, ok = rawSet.(*Set)
	if !ok {
		t.Fatalf("nestedSetUpdate: bad: nested set type\n\nexpected: *Set\n\ngot: %v\n\n", reflect.TypeOf(m["nested_set"]))
	}
	if set.Len() != 1 {
		t.Fatalf("nestedSetUpdate: bad: nested set length\n\nexpected: 1\n\ngot: %v\n\n", set.Len())
	}

	setList := set.List()
	var item map[string]interface{}
	item, ok = setList[0].(map[string]interface{})
	if !ok {
		t.Fatalf("nestedSetUpdate: bad: nested set item type\n\nexpected: map[string]interface{}\n\ngot: %v\n\n", reflect.TypeOf(setList[0]))
	}

	val := item["val"].(int)
	if val != 1 {
		t.Fatalf("nestedSetUpdate: bad: val\n\nexpected: 1\n\ngot: %v\n\n", val)
	}
}

// https://github.com/hashicorp/terraform/issues/914
func TestDiffFieldReader_MapHandling(t *testing.T) {
	schema := map[string]*Schema{
		"tags": &Schema{
			Type: TypeMap,
		},
	}
	r := &DiffFieldReader{
		Schema: schema,
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"tags.%": &terraform.ResourceAttrDiff{
					Old: "1",
					New: "2",
				},
				"tags.baz": &terraform.ResourceAttrDiff{
					Old: "",
					New: "qux",
				},
			},
		},
		Source: &MapFieldReader{
			Schema: schema,
			Map: BasicMapReader(map[string]string{
				"tags.%":   "1",
				"tags.foo": "bar",
			}),
		},
	}

	result, err := r.ReadField([]string{"tags"})
	if err != nil {
		t.Fatalf("ReadField failed: %#v", err)
	}

	expected := map[string]interface{}{
		"foo": "bar",
		"baz": "qux",
	}

	if !reflect.DeepEqual(expected, result.Value) {
		t.Fatalf("bad: DiffHandling\n\nexpected: %#v\n\ngot: %#v\n\n", expected, result.Value)
	}
}

func TestDiffFieldReader_extra(t *testing.T) {
	schema := map[string]*Schema{
		"stringComputed": &Schema{Type: TypeString},

		"listMap": &Schema{
			Type: TypeList,
			Elem: &Schema{
				Type: TypeMap,
			},
		},

		"mapRemove": &Schema{Type: TypeMap},

		"setChange": &Schema{
			Type:     TypeSet,
			Optional: true,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"index": &Schema{
						Type:     TypeInt,
						Required: true,
					},

					"value": &Schema{
						Type:     TypeString,
						Required: true,
					},
				},
			},
			Set: func(a interface{}) int {
				m := a.(map[string]interface{})
				return m["index"].(int)
			},
		},

		"setEmpty": &Schema{
			Type:     TypeSet,
			Optional: true,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"index": &Schema{
						Type:     TypeInt,
						Required: true,
					},

					"value": &Schema{
						Type:     TypeString,
						Required: true,
					},
				},
			},
			Set: func(a interface{}) int {
				m := a.(map[string]interface{})
				return m["index"].(int)
			},
		},
	}

	r := &DiffFieldReader{
		Schema: schema,
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"stringComputed": &terraform.ResourceAttrDiff{
					Old:         "foo",
					New:         "bar",
					NewComputed: true,
				},

				"listMap.0.bar": &terraform.ResourceAttrDiff{
					NewRemoved: true,
				},

				"mapRemove.bar": &terraform.ResourceAttrDiff{
					NewRemoved: true,
				},

				"setChange.10.value": &terraform.ResourceAttrDiff{
					Old: "50",
					New: "80",
				},

				"setEmpty.#": &terraform.ResourceAttrDiff{
					Old: "2",
					New: "0",
				},
			},
		},

		Source: &MapFieldReader{
			Schema: schema,
			Map: BasicMapReader(map[string]string{
				"listMap.#":     "2",
				"listMap.0.foo": "bar",
				"listMap.0.bar": "baz",
				"listMap.1.baz": "baz",

				"mapRemove.foo": "bar",
				"mapRemove.bar": "bar",

				"setChange.#":        "1",
				"setChange.10.index": "10",
				"setChange.10.value": "50",

				"setEmpty.#":        "2",
				"setEmpty.10.index": "10",
				"setEmpty.10.value": "50",
				"setEmpty.20.index": "20",
				"setEmpty.20.value": "50",
			}),
		},
	}

	cases := map[string]struct {
		Addr   []string
		Result FieldReadResult
		Err    bool
	}{
		"stringComputed": {
			[]string{"stringComputed"},
			FieldReadResult{
				Value:    "",
				Exists:   true,
				Computed: true,
			},
			false,
		},

		"listMapRemoval": {
			[]string{"listMap"},
			FieldReadResult{
				Value: []interface{}{
					map[string]interface{}{
						"foo": "bar",
					},
					map[string]interface{}{
						"baz": "baz",
					},
				},
				Exists: true,
			},
			false,
		},

		"mapRemove": {
			[]string{"mapRemove"},
			FieldReadResult{
				Value: map[string]interface{}{
					"foo": "bar",
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"setChange": {
			[]string{"setChange"},
			FieldReadResult{
				Value: []interface{}{
					map[string]interface{}{
						"index": 10,
						"value": "80",
					},
				},
				Exists: true,
			},
			false,
		},

		"setEmpty": {
			[]string{"setEmpty"},
			FieldReadResult{
				Value:  []interface{}{},
				Exists: true,
			},
			false,
		},
	}

	for name, tc := range cases {
		out, err := r.ReadField(tc.Addr)
		if err != nil != tc.Err {
			t.Fatalf("%s: err: %s", name, err)
		}
		if s, ok := out.Value.(*Set); ok {
			// If it is a set, convert to a list so its more easily checked.
			out.Value = s.List()
		}
		if !reflect.DeepEqual(tc.Result, out) {
			t.Fatalf("%s: bad: %#v", name, out)
		}
	}
}

func TestDiffFieldReader(t *testing.T) {
	testFieldReader(t, func(s map[string]*Schema) FieldReader {
		return &DiffFieldReader{
			Schema: s,
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"bool": &terraform.ResourceAttrDiff{
						Old: "",
						New: "true",
					},

					"int": &terraform.ResourceAttrDiff{
						Old: "",
						New: "42",
					},

					"float": &terraform.ResourceAttrDiff{
						Old: "",
						New: "3.1415",
					},

					"string": &terraform.ResourceAttrDiff{
						Old: "",
						New: "string",
					},

					"stringComputed": &terraform.ResourceAttrDiff{
						Old:         "foo",
						New:         "bar",
						NewComputed: true,
					},

					"list.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "2",
					},

					"list.0": &terraform.ResourceAttrDiff{
						Old: "",
						New: "foo",
					},

					"list.1": &terraform.ResourceAttrDiff{
						Old: "",
						New: "bar",
					},

					"listInt.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "2",
					},

					"listInt.0": &terraform.ResourceAttrDiff{
						Old: "",
						New: "21",
					},

					"listInt.1": &terraform.ResourceAttrDiff{
						Old: "",
						New: "42",
					},

					"map.foo": &terraform.ResourceAttrDiff{
						Old: "",
						New: "bar",
					},

					"map.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "baz",
					},

					"set.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "2",
					},

					"set.10": &terraform.ResourceAttrDiff{
						Old: "",
						New: "10",
					},

					"set.50": &terraform.ResourceAttrDiff{
						Old: "",
						New: "50",
					},

					"setDeep.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "2",
					},

					"setDeep.10.index": &terraform.ResourceAttrDiff{
						Old: "",
						New: "10",
					},

					"setDeep.10.value": &terraform.ResourceAttrDiff{
						Old: "",
						New: "foo",
					},

					"setDeep.50.index": &terraform.ResourceAttrDiff{
						Old: "",
						New: "50",
					},

					"setDeep.50.value": &terraform.ResourceAttrDiff{
						Old: "",
						New: "bar",
					},
				},
			},

			Source: &MapFieldReader{
				Schema: s,
				Map: BasicMapReader(map[string]string{
					"listMap.#":     "2",
					"listMap.0.foo": "bar",
					"listMap.0.bar": "baz",
					"listMap.1.baz": "baz",
				}),
			},
		}
	})
}
