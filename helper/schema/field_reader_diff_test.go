package schema

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestDiffFieldReader_impl(t *testing.T) {
	var _ FieldReader = new(DiffFieldReader)
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
			t.Fatalf("Case %q:\ngiven: %#v\nexpected: %#v", name, out, tc.Result)
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

func TestDiffFieldReader_SetInSet(t *testing.T) {
	schema := map[string]*Schema{
		"main_set": &Schema{
			Type:     TypeSet,
			Optional: true,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"inner_string_set": &Schema{
						Type:     TypeSet,
						Required: true,
						Set:      HashString,
						Elem:     &Schema{Type: TypeString},
					},
					"inner_int": &Schema{
						Type:     TypeInt,
						Required: true,
					},
				},
			},
		},
		"main_int": &Schema{
			Type:     TypeInt,
			Optional: true,
		},
	}

	var readers = make(map[string]FieldReader)
	readers["state"] = &MapFieldReader{
		Schema: schema,
		Map: BasicMapReader(map[string]string{
			"id":                                              "8395051352714003426",
			"main_int":                                        "9",
			"main_set.#":                                      "1",
			"main_set.2476980464.inner_string_set.#":          "2",
			"main_set.2476980464.inner_string_set.2654390964": "blue",
			"main_set.2476980464.inner_string_set.3499814433": "green",
			"main_set.2476980464.inner_int":                   "4",
		}),
	}

	// If we're only changing main_int
	dfr := &DiffFieldReader{
		Schema: schema,
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"main_int": &terraform.ResourceAttrDiff{
					Old: "9",
					New: "2",
				},
			},
		},
		Source: &MultiLevelFieldReader{
			Levels:  []string{"state", "config"},
			Readers: readers,
		},
	}

	// main_list should NOT be in the diff at all
	result, err := dfr.ReadField([]string{"main_set"})
	if err != nil {
		t.Fatalf("ReadField failed: %s", err)
	}
	expectedResult := NewSet(HashString, []interface{}{})
	if !expectedResult.Equal(result.Value) {
		t.Fatalf("ReadField returned unexpected result.\nGiven: %#v\nexpected: %#v",
			result, expectedResult)
	}
}

func TestDiffFieldReader_SetInList(t *testing.T) {
	schema := map[string]*Schema{
		"main_list": &Schema{
			Type:     TypeList,
			Optional: true,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"inner_string_set": &Schema{
						Type:     TypeSet,
						Required: true,
						Set:      HashString,
						Elem:     &Schema{Type: TypeString},
					},
					"inner_int": &Schema{
						Type:     TypeInt,
						Required: true,
					},
				},
			},
		},
		"main_int": &Schema{
			Type:     TypeInt,
			Optional: true,
		},
	}

	var readers = make(map[string]FieldReader)
	readers["state"] = &MapFieldReader{
		Schema: schema,
		Map: BasicMapReader(map[string]string{
			"id":                                      "8395051352714003426",
			"main_int":                                "9",
			"main_list.#":                             "1",
			"main_list.0.inner_string_set.#":          "2",
			"main_list.0.inner_string_set.2654390964": "blue",
			"main_list.0.inner_string_set.3499814433": "green",
			"main_list.0.inner_int":                   "4",
		}),
	}

	// If we're only changing main_int
	dfr := &DiffFieldReader{
		Schema: schema,
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"main_int": &terraform.ResourceAttrDiff{
					Old: "9",
					New: "2",
				},
			},
		},
		Source: &MultiLevelFieldReader{
			Levels:  []string{"state", "config"},
			Readers: readers,
		},
	}

	// main_list should NOT be in the diff at all
	result, err := dfr.ReadField([]string{"main_list"})
	if err != nil {
		t.Fatalf("ReadField failed: %s", err)
	}
	expectedResult := FieldReadResult{
		Value:          []interface{}{},
		ValueProcessed: nil,
		Exists:         false,
		Computed:       false,
	}
	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("ReadField returned unexpected result.\nGiven: %#v\nexpected: %#v",
			result, expectedResult)
	}
}

func TestDiffFieldReader_SetInList_singleInstance(t *testing.T) {
	schema := map[string]*Schema{
		"main_list": &Schema{
			Type:     TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"inner_string_set": &Schema{
						Type:     TypeSet,
						Required: true,
						Set:      HashString,
						Elem:     &Schema{Type: TypeString},
					},
					"inner_int": &Schema{
						Type:     TypeInt,
						Required: true,
					},
				},
			},
		},
		"main_int": &Schema{
			Type:     TypeInt,
			Optional: true,
		},
	}

	var readers = make(map[string]FieldReader)
	readers["state"] = &MapFieldReader{
		Schema: schema,
		Map: BasicMapReader(map[string]string{
			"id":                                      "8395051352714003426",
			"main_int":                                "9",
			"main_list.#":                             "1",
			"main_list.0.inner_string_set.#":          "2",
			"main_list.0.inner_string_set.2654390964": "blue",
			"main_list.0.inner_string_set.3499814433": "green",
			"main_list.0.inner_int":                   "4",
		}),
	}

	// 1. NEGATIVE (diff doesn't contain list)
	// If we're only changing main_int
	dfrNegative := &DiffFieldReader{
		Schema: schema,
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"main_int": &terraform.ResourceAttrDiff{
					Old: "9",
					New: "2",
				},
			},
		},
		Source: &MultiLevelFieldReader{
			Levels:  []string{"state", "config"},
			Readers: readers,
		},
	}
	// main_list should NOT be in the diff at all
	resultNegative, err := dfrNegative.ReadField([]string{"main_list"})
	if err != nil {
		t.Fatalf("ReadField failed: %s", err)
	}
	expectedNegativeResult := FieldReadResult{
		Value:          []interface{}{},
		ValueProcessed: nil,
		Exists:         false,
		Computed:       false,
	}
	if !reflect.DeepEqual(resultNegative, expectedNegativeResult) {
		t.Fatalf("ReadField returned unexpected resultNegative.\nGiven: %#v\nexpected: %#v",
			resultNegative, expectedNegativeResult)
	}

	// 1. POSITIVE (diff contains list)
	dfrPositive := &DiffFieldReader{
		Schema: schema,
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"main_list.0.inner_int": &terraform.ResourceAttrDiff{
					Old: "4",
					New: "2",
				},
			},
		},
		Source: &MultiLevelFieldReader{
			Levels:  []string{"state", "config"},
			Readers: readers,
		},
	}
	resultPositive, err := dfrPositive.ReadField([]string{"main_list"})
	if err != nil {
		t.Fatalf("ReadField failed: %s", err)
	}
	if !resultPositive.Exists {
		t.Fatal("Expected resultPositive to exist")
	}
	list := resultPositive.Value.([]interface{})
	if len(list) != 1 {
		t.Fatalf("Expected exactly 1 list instance, %d given", len(list))
	}

	m := list[0].(map[string]interface{})

	m_expectedInnerInt := 2
	m_innerInt, ok := m["inner_int"]
	if !ok {
		t.Fatal("Expected inner_int key to exist in map")
	}
	if m_innerInt != m_expectedInnerInt {
		t.Fatalf("Expected inner_int (%d) doesn't match w/ given: %d", m_expectedInnerInt, m_innerInt)
	}

	m_expectedStringSet := NewSet(HashString, []interface{}{"blue", "green"})
	m_StringSet, ok := m["inner_string_set"]
	if !ok {
		t.Fatal("Expected inner_string_set key to exist in map")
	}
	if !m_expectedStringSet.Equal(m_StringSet) {
		t.Fatalf("Expected inner_string_set (%q) doesn't match w/ given: %q",
			m_expectedStringSet.List(), m_StringSet.(*Set).List())
	}
}

func TestDiffFieldReader_SetInList_multipleInstances(t *testing.T) {
	schema := map[string]*Schema{
		"main_list": &Schema{
			Type:     TypeList,
			Optional: true,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"inner_string_set": &Schema{
						Type:     TypeSet,
						Required: true,
						Set:      HashString,
						Elem:     &Schema{Type: TypeString},
					},
					"inner_int": &Schema{
						Type:     TypeInt,
						Required: true,
					},
				},
			},
		},
		"main_int": &Schema{
			Type:     TypeInt,
			Optional: true,
		},
	}

	var readers = make(map[string]FieldReader)
	readers["state"] = &MapFieldReader{
		Schema: schema,
		Map: BasicMapReader(map[string]string{
			"id":                                      "8395051352714003426",
			"main_int":                                "9",
			"main_list.#":                             "3",
			"main_list.0.inner_string_set.#":          "2",
			"main_list.0.inner_string_set.2654390964": "blue",
			"main_list.0.inner_string_set.3499814433": "green",
			"main_list.0.inner_int":                   "4",
			"main_list.1.inner_string_set.#":          "2",
			"main_list.1.inner_string_set.1830392916": "brown",
			"main_list.1.inner_string_set.4200685455": "red",
			"main_list.1.inner_int":                   "4",
			"main_list.2.inner_string_set.#":          "3",
			"main_list.2.inner_string_set.2053932785": "one",
			"main_list.2.inner_string_set.298486374":  "two",
			"main_list.2.inner_string_set.1187371253": "three",
			"main_list.2.inner_int":                   "914",
		}),
	}

	dfr := &DiffFieldReader{
		Schema: schema,
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"main_list.0.inner_int": &terraform.ResourceAttrDiff{
					Old: "4",
					New: "5",
				},
				"main_list.1.inner_int": &terraform.ResourceAttrDiff{
					Old: "4",
					New: "34",
				},
			},
		},
		Source: &MultiLevelFieldReader{
			Levels:  []string{"state", "config"},
			Readers: readers,
		},
	}

	result, err := dfr.ReadField([]string{"main_list"})
	if err != nil {
		t.Fatalf("ReadField 2 failed: %s", err)
	}
	if !result.Exists {
		t.Fatal("Expected result to exist")
	}
	list := result.Value.([]interface{})
	if len(list) != 3 {
		t.Fatalf("Expected exactly 3 list instances, %d given", len(list))
	}

	// First
	m1 := list[0].(map[string]interface{})

	m1_expectedInnerInt := 5
	m1_innerInt, ok := m1["inner_int"]
	if !ok {
		t.Fatal("Expected 1st inner_int key to exist in map")
	}
	if m1_innerInt != m1_expectedInnerInt {
		t.Fatalf("Expected 1st inner_int (%d) doesn't match w/ given: %d", m1_expectedInnerInt, m1_innerInt)
	}

	m1_expectedStringSet := NewSet(HashString, []interface{}{"blue", "green"})
	m1_StringSet, ok := m1["inner_string_set"]
	if !ok {
		t.Fatal("Expected 1st inner_string_set key to exist in map")
	}
	if !m1_expectedStringSet.Equal(m1_StringSet) {
		t.Fatalf("Expected 1st inner_string_set (%q) doesn't match w/ given: %q",
			m1_expectedStringSet.List(), m1_StringSet.(*Set).List())
	}

	// Second
	m2 := list[1].(map[string]interface{})

	m2_expectedInnerInt := 34
	m2_innerInt, ok := m2["inner_int"]
	if !ok {
		t.Fatal("Expected 2nd inner_int key to exist in map")
	}
	if m2_innerInt != m2_expectedInnerInt {
		t.Fatalf("Expected 2nd inner_int (%d) doesn't match w/ given: %d", m2_expectedInnerInt, m2_innerInt)
	}

	m2_expectedStringSet := NewSet(HashString, []interface{}{"brown", "red"})
	m2_StringSet, ok := m2["inner_string_set"].(*Set)
	if !ok {
		t.Fatal("Expected 2nd inner_string_set key to exist in map")
	}
	if !m2_expectedStringSet.Equal(m2_StringSet) {
		t.Fatalf("Expected 2nd inner_string_set (%q) doesn't match w/ given: %q",
			m2_expectedStringSet.List(), m2_StringSet.List())
	}

	// Third
	m3 := list[2].(map[string]interface{})

	m3_expectedInnerInt := 914
	m3_innerInt, ok := m3["inner_int"]
	if !ok {
		t.Fatal("Expected 3rd inner_int key to exist in map")
	}
	if m3_innerInt != m3_expectedInnerInt {
		t.Fatalf("Expected 3rd inner_int (%d) doesn't match w/ given: %d", m3_expectedInnerInt, m3_innerInt)
	}

	m3_expectedStringSet := NewSet(HashString, []interface{}{"one", "two", "three"})
	m3_StringSet, ok := m3["inner_string_set"].(*Set)
	if !ok {
		t.Fatal("Expected 3rd inner_string_set key to exist in map")
	}
	if !m3_expectedStringSet.Equal(m3_StringSet) {
		t.Fatalf("Expected 3rd inner_string_set (%q) doesn't match w/ given: %q",
			m3_expectedStringSet.List(), m3_StringSet.List())
	}
}

func TestDiffFieldReader_SetInList_deeplyNested_singleInstance(t *testing.T) {
	inInnerSetResource := &Resource{
		Schema: map[string]*Schema{
			"in_in_inner_string": &Schema{
				Type:     TypeString,
				Required: true,
			},
			"in_in_inner_list": &Schema{
				Type:     TypeList,
				Optional: true,
				Elem:     &Schema{Type: TypeString},
			},
		},
	}
	innerSetResource := &Resource{
		Schema: map[string]*Schema{
			"in_inner_set": &Schema{
				Type:     TypeSet,
				Required: true,
				MaxItems: 1,
				Elem:     inInnerSetResource,
			},
			"in_inner_string_list": &Schema{
				Type:     TypeList,
				Optional: true,
				Elem:     &Schema{Type: TypeString},
			},
			"in_inner_bool": &Schema{
				Type:     TypeBool,
				Required: true,
			},
		},
	}
	schema := map[string]*Schema{
		"main_list": &Schema{
			Type:     TypeList,
			Optional: true,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"inner_string_set": &Schema{
						Type:     TypeSet,
						Required: true,
						Set:      HashString,
						Elem:     &Schema{Type: TypeString},
					},
					"inner_set": &Schema{
						Type:     TypeSet,
						Required: true,
						MaxItems: 1,
						Elem:     innerSetResource,
					},
					"inner_bool": &Schema{
						Type:     TypeBool,
						Optional: true,
						Default:  false,
					},
					"inner_int": &Schema{
						Type:     TypeInt,
						Required: true,
					},
				},
			},
		},
		"main_int": &Schema{
			Type:     TypeInt,
			Optional: true,
		},
	}

	var readers = make(map[string]FieldReader)
	readers["state"] = &MapFieldReader{
		Schema: schema,
		Map: BasicMapReader(map[string]string{
			"id":                                                                          "8395051352714003426",
			"main_int":                                                                    "5",
			"main_list.#":                                                                 "1",
			"main_list.0.inner_bool":                                                      "true",
			"main_list.0.inner_int":                                                       "2",
			"main_list.0.inner_set.#":                                                     "1",
			"main_list.0.inner_set.2496801729.in_inner_bool":                              "false",
			"main_list.0.inner_set.2496801729.in_inner_set.#":                             "1",
			"main_list.0.inner_set.2496801729.in_inner_set.1989773763.in_in_inner_list.#": "1",
			"main_list.0.inner_set.2496801729.in_inner_set.1989773763.in_in_inner_list.0": "alpha",
			"main_list.0.inner_set.2496801729.in_inner_set.1989773763.in_in_inner_string": "delta",
			"main_list.0.inner_set.2496801729.in_inner_string_list.#":                     "3",
			"main_list.0.inner_set.2496801729.in_inner_string_list.0":                     "one",
			"main_list.0.inner_set.2496801729.in_inner_string_list.1":                     "two",
			"main_list.0.inner_set.2496801729.in_inner_string_list.2":                     "three",
			"main_list.0.inner_string_set.#":                                              "2",
			"main_list.0.inner_string_set.1830392916":                                     "brown",
			"main_list.0.inner_string_set.4200685455":                                     "red",
		}),
	}

	// If we're only changing main_int
	dfr := &DiffFieldReader{
		Schema: schema,
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"main_list.0.inner_int": &terraform.ResourceAttrDiff{
					Old: "2",
					New: "78",
				},
			},
		},
		Source: &MultiLevelFieldReader{
			Levels:  []string{"state", "config"},
			Readers: readers,
		},
	}

	// main_list should NOT be in the diff at all
	result, err := dfr.ReadField([]string{"main_list"})
	if err != nil {
		t.Fatalf("ReadField failed: %s", err)
	}
	if !result.Exists {
		t.Fatal("Expected result to exist")
	}
	list := result.Value.([]interface{})
	if len(list) != 1 {
		t.Fatalf("Expected exactly 1 list instance, %d given", len(list))
	}

	// One day we may have a custom comparison function for nested sets
	// Until that day comes it will look as ridiculous as below
	m1 := list[0].(map[string]interface{})

	m1_expectedInnerInt := 78
	m1_innerInt, ok := m1["inner_int"]
	if !ok {
		t.Fatal("Expected inner_int key to exist in map")
	}
	if m1_innerInt != m1_expectedInnerInt {
		t.Fatalf("Expected inner_int (%d) doesn't match w/ given: %d", m1_expectedInnerInt, m1_innerInt)
	}

	m1_expectedInnerBool := true
	m1_innerBool, ok := m1["inner_bool"]
	if !ok {
		t.Fatal("Expected inner_bool key to exist in map")
	}
	if m1_innerBool != m1_expectedInnerBool {
		t.Fatalf("Expected inner_bool (%t) doesn't match w/ given: %t", m1_expectedInnerBool, m1_innerBool)
	}

	m1_expectedStringSet := NewSet(HashString, []interface{}{"brown", "red"})
	m1_StringSet, ok := m1["inner_string_set"]
	if !ok {
		t.Fatal("Expected inner_string_set key to exist in map")
	}
	if !m1_expectedStringSet.Equal(m1_StringSet) {
		t.Fatalf("Expected inner_string_set (%q) doesn't match w/ given: %q",
			m1_expectedStringSet.List(), m1_StringSet.(*Set).List())
	}

	m1_InnerSet, ok := m1["inner_set"]
	if !ok {
		t.Fatal("Expected inner_set key to exist in map")
	}
	m1_InnerSet_list := m1_InnerSet.(*Set).List()
	m := m1_InnerSet_list[0].(map[string]interface{})

	expectedInInnerBool := false
	inInnerBool, ok := m["in_inner_bool"]
	if !ok {
		t.Fatal("Expected in_inner_bool key to exist in map")
	}
	if inInnerBool != expectedInInnerBool {
		t.Fatalf("Expected inner_set[0].in_inner_bool (%#v) doesn't match w/ given: %#v",
			expectedInInnerBool, inInnerBool)
	}
	expectedInInnerStringList := []interface{}{"one", "two", "three"}
	inInnerStringList, ok := m["in_inner_string_list"]
	if !ok {
		t.Fatal("Expected in_inner_string_list key to exist in map")
	}
	if !reflect.DeepEqual(inInnerStringList, expectedInInnerStringList) {
		t.Fatalf("Expected inner_set[0].in_inner_string_list (%#v) doesn't match w/ given: %#v",
			expectedInInnerStringList, inInnerStringList)
	}

	expectedInInnerSet := map[string]interface{}{
		"in_in_inner_string": "delta",
		"in_in_inner_list":   []interface{}{"alpha"},
	}
	inInnerSet, ok := m["in_inner_set"]
	if !ok {
		t.Fatal("Expected in_inner_set key to exist in map")
	}
	inInnerSet_list := inInnerSet.(*Set).List()
	m2 := inInnerSet_list[0].(map[string]interface{})
	if !reflect.DeepEqual(expectedInInnerSet, m2) {
		t.Fatalf("Expected in_inner_set to match:\nGiven: %#v\nExpected: %#v\n",
			m2, expectedInInnerSet)
	}
}
