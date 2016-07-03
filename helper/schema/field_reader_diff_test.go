package schema

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/hashcode"
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

func TestDiffFieldReader_setItemRemoved(t *testing.T) {
	resourceAwsElbListenerHash := func(v interface{}) int {
		var buf bytes.Buffer
		m := v.(map[string]interface{})
		buf.WriteString(fmt.Sprintf("%d-", m["instance_port"].(int)))
		buf.WriteString(fmt.Sprintf("%s-",
			strings.ToLower(m["instance_protocol"].(string))))
		buf.WriteString(fmt.Sprintf("%d-", m["lb_port"].(int)))
		buf.WriteString(fmt.Sprintf("%s-",
			strings.ToLower(m["lb_protocol"].(string))))

		if v, ok := m["ssl_certificate_id"]; ok {
			buf.WriteString(fmt.Sprintf("%s-", v.(string)))
		}

		return hashcode.String(buf.String())
	}

	schema := map[string]*Schema{
		"listener": &Schema{
			Type:     TypeSet,
			Required: true,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"instance_port": &Schema{
						Type:     TypeInt,
						Required: true,
					},

					"instance_protocol": &Schema{
						Type:     TypeString,
						Required: true,
					},

					"lb_port": &Schema{
						Type:     TypeInt,
						Required: true,
					},

					"lb_protocol": &Schema{
						Type:     TypeString,
						Required: true,
					},

					"ssl_certificate_id": &Schema{
						Type:     TypeString,
						Optional: true,
					},
				},
			},
			Set: resourceAwsElbListenerHash,
		},

		"tags": &Schema{
			Type:     TypeMap,
			Optional: true,
		},
	}

	var readers = make(map[string]FieldReader)
	readers["state"] = &MapFieldReader{
		Schema: schema,
		Map: BasicMapReader(map[string]string{
			"id":                                    "tf-lb-t77qtnbbm5dczcnker4c7boj7m",
			"listener.#":                            "1",
			"listener.206423021.instance_port":      "8000",
			"listener.206423021.instance_protocol":  "http",
			"listener.206423021.lb_port":            "80",
			"listener.206423021.lb_protocol":        "http",
			"listener.206423021.ssl_certificate_id": "",
			"tags.%":   "1",
			"tags.bar": "baz",
		}),
	}

	dfr := &DiffFieldReader{
		Schema: schema,
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"listener.3931999347.ssl_certificate_id": &terraform.ResourceAttrDiff{
					Old: "", New: "", NewRemoved: false,
				},
				"tags.bar": &terraform.ResourceAttrDiff{
					Old: "baz", New: "", NewRemoved: true,
				},
				"listener.206423021.instance_port": &terraform.ResourceAttrDiff{
					Old: "8000", New: "0", NewRemoved: true,
				},
				"listener.206423021.lb_port": &terraform.ResourceAttrDiff{
					Old: "80", New: "0", NewRemoved: true,
				},
				"listener.206423021.lb_protocol": &terraform.ResourceAttrDiff{
					Old: "http", New: "", NewRemoved: true,
				},
				"listener.206423021.ssl_certificate_id": &terraform.ResourceAttrDiff{
					Old: "", New: "", NewRemoved: true,
				},
				"listener.3931999347.instance_protocol": &terraform.ResourceAttrDiff{
					Old: "", New: "http", NewRemoved: false,
				},
				"listener.206423021.instance_protocol": &terraform.ResourceAttrDiff{
					Old: "http", New: "", NewRemoved: true,
				},
				"listener.3931999347.instance_port": &terraform.ResourceAttrDiff{
					Old: "", New: "8080", NewRemoved: false,
				},
				"listener.3931999347.lb_port": &terraform.ResourceAttrDiff{
					Old: "", New: "80", NewRemoved: false,
				},
				"listener.3931999347.lb_protocol": &terraform.ResourceAttrDiff{
					Old: "", New: "http", NewRemoved: false,
				},
				"tags.%": &terraform.ResourceAttrDiff{
					Old: "1", New: "0", NewRemoved: false,
				},
			},
		},
		Source: &MultiLevelFieldReader{
			Levels:  []string{"state", "config"},
			Readers: readers,
		},
	}

	result, err := dfr.ReadField([]string{"listener"})
	if err != nil {
		t.Fatalf("ReadField failed: %s", err)
	}
	expectedResult := NewSet(resourceAwsElbListenerHash, []interface{}{
		map[string]interface{}{
			"lb_port":            80,
			"lb_protocol":        "http",
			"ssl_certificate_id": "",
			"instance_port":      8080,
			"instance_protocol":  "http",
		},
	})
	if !expectedResult.Equal(result.Value) {
		t.Fatalf("ReadField returned unexpected result.\nGiven: %#v\nexpected: %#v",
			result, expectedResult)
	}
}

func TestDiffFieldReader_CodeDeploy_special(t *testing.T) {
	resourceAwsCodeDeployTriggerConfigHash := func(v interface{}) int {
		var buf bytes.Buffer
		m := v.(map[string]interface{})
		if v, ok := m["trigger_name"]; ok {
			buf.WriteString(fmt.Sprintf("%s-", v.(string)))
		}
		if v, ok := m["trigger_target_arn"]; ok {
			buf.WriteString(fmt.Sprintf("%s-", v.(string)))
		}

		if triggerEvents, ok := m["trigger_events"]; ok {
			names := triggerEvents.(*Set).List()
			strings := make([]string, len(names))
			for i, raw := range names {
				strings[i] = raw.(string)
			}
			sort.Strings(strings)

			for _, s := range strings {
				buf.WriteString(fmt.Sprintf("%s-", s))
			}
		}
		return hashcode.String(buf.String())
	}
	schema := map[string]*Schema{
		"trigger_configuration": &Schema{
			Type:     TypeSet,
			Optional: true,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"trigger_events": &Schema{
						Type:     TypeSet,
						Required: true,
						Set:      HashString,
						Elem: &Schema{
							Type: TypeString,
						},
					},

					"trigger_name": &Schema{
						Type:     TypeString,
						Required: true,
					},

					"trigger_target_arn": &Schema{
						Type:     TypeString,
						Required: true,
					},
				},
			},
			Set: resourceAwsCodeDeployTriggerConfigHash,
		},
	}

	var readers = make(map[string]FieldReader)
	readers["state"] = &MapFieldReader{
		Schema: schema,
		Map: BasicMapReader(map[string]string{
			"id": "b62bb84c-2d61-42b2-a50c-b629bf76027d",
			"trigger_configuration.#":                                    "1",
			"trigger_configuration.2509701091.trigger_events.#":          "1",
			"trigger_configuration.2509701091.trigger_events.4157777861": "DeploymentFailure",
			"trigger_configuration.2509701091.trigger_name":              "foo-trigger",
			"trigger_configuration.2509701091.trigger_target_arn":        "arn:aws:sns:us-west-2:123456789012:foo-topic-rsimko-test",
		}),
	}

	dfr := &DiffFieldReader{
		Schema: schema,
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"trigger_configuration.2509701091.trigger_target_arn": &terraform.ResourceAttrDiff{
					Old: "arn:aws:sns:us-west-2:123456789012:foo-topic-rsimko-test", New: "",
					NewComputed: false, NewRemoved: true,
				},
				"trigger_configuration.2509701091.trigger_events.#": &terraform.ResourceAttrDiff{
					Old: "1", New: "0",
					NewComputed: false, NewRemoved: false,
				},
				"trigger_configuration.2509701091.trigger_events.4157777861": &terraform.ResourceAttrDiff{
					Old: "DeploymentFailure", New: "",
					NewComputed: false, NewRemoved: true,
				},
				"trigger_configuration.2509701091.trigger_name": &terraform.ResourceAttrDiff{
					Old: "foo-trigger", New: "",
					NewComputed: false, NewRemoved: true,
				},

				"trigger_configuration.3738386998.trigger_events.#": &terraform.ResourceAttrDiff{
					Old: "0", New: "2",
					NewComputed: false, NewRemoved: false,
				},
				"trigger_configuration.3738386998.trigger_name": &terraform.ResourceAttrDiff{
					Old: "", New: "foo-trigger",
					NewComputed: false, NewRemoved: false,
				},
				"trigger_configuration.3738386998.trigger_events.4157777861": &terraform.ResourceAttrDiff{
					Old: "", New: "DeploymentFailure",
					NewComputed: false, NewRemoved: false,
				},
				"trigger_configuration.3738386998.trigger_events.3108600758": &terraform.ResourceAttrDiff{
					Old: "", New: "DeploymentSuccess",
					NewComputed: false, NewRemoved: false,
				},
				"trigger_configuration.3738386998.trigger_target_arn": &terraform.ResourceAttrDiff{
					Old: "", New: "arn:aws:sns:us-west-2:123456789012:foo-topic-rsimko-test",
					NewComputed: false, NewRemoved: false,
				},
			},
			Destroy:        false,
			DestroyTainted: false,
		},
		Source: &MultiLevelFieldReader{
			Levels:  []string{"state", "config"},
			Readers: readers,
		},
	}

	result, err := dfr.ReadField([]string{"trigger_configuration"})
	if err != nil {
		t.Fatalf("ReadField failed: %s", err)
	}

	list := result.Value.(*Set).List()
	m := list[0].(map[string]interface{})

	expectedEvents := NewSet(HashString, []interface{}{"DeploymentSuccess", "DeploymentFailure"})
	if !m["trigger_events"].(*Set).Equal(expectedEvents) {
		t.Fatalf("ReadField returned unexpected trigger_events.\nGiven: %#v\nexpected: %#v",
			m["trigger_events"].(*Set).List(), expectedEvents.List())
	}
	delete(m, "trigger_events")

	expectedResult := map[string]interface{}{
		"trigger_name":       "foo-trigger",
		"trigger_target_arn": "arn:aws:sns:us-west-2:123456789012:foo-topic-rsimko-test",
	}
	if !reflect.DeepEqual(m, expectedResult) {
		t.Fatalf("ReadField returned unexpected result.\nGiven: %#v\nexpected: %#v",
			m, expectedResult)
	}
}

func TestDiffFieldReader_Fastly_special(t *testing.T) {
	schema := map[string]*Schema{
		"name": &Schema{
			Type:     TypeString,
			Required: true,
		},

		"domain": &Schema{
			Type:     TypeSet,
			Required: true,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"name": &Schema{
						Type:     TypeString,
						Required: true,
					},

					"comment": &Schema{
						Type:     TypeString,
						Optional: true,
					},
				},
			},
		},

		"default_ttl": &Schema{
			Type:     TypeInt,
			Optional: true,
			Default:  3600,
		},

		"backend": &Schema{
			Type:     TypeSet,
			Required: true,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"name": &Schema{
						Type:     TypeString,
						Required: true,
					},
					"address": &Schema{
						Type:     TypeString,
						Required: true,
					},
					"auto_loadbalance": &Schema{
						Type:     TypeBool,
						Optional: true,
						Default:  true,
					},
					"between_bytes_timeout": &Schema{
						Type:     TypeInt,
						Optional: true,
						Default:  10000,
					},
					"connect_timeout": &Schema{
						Type:     TypeInt,
						Optional: true,
						Default:  1000,
					},
					"error_threshold": &Schema{
						Type:     TypeInt,
						Optional: true,
						Default:  0,
					},
					"first_byte_timeout": &Schema{
						Type:     TypeInt,
						Optional: true,
						Default:  15000,
					},
					"max_conn": &Schema{
						Type:     TypeInt,
						Optional: true,
						Default:  200,
					},
					"port": &Schema{
						Type:     TypeInt,
						Optional: true,
						Default:  80,
					},
					"ssl_check_cert": &Schema{
						Type:     TypeBool,
						Optional: true,
						Default:  true,
					},
					"weight": &Schema{
						Type:     TypeInt,
						Optional: true,
						Default:  100,
					},
				},
			},
		},

		"force_destroy": &Schema{
			Type:     TypeBool,
			Optional: true,
		},

		"gzip": &Schema{
			Type:     TypeSet,
			Optional: true,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"name": &Schema{
						Type:     TypeString,
						Required: true,
					},
					"content_types": &Schema{
						Type:     TypeSet,
						Optional: true,
						Elem:     &Schema{Type: TypeString},
					},
					"extensions": &Schema{
						Type:     TypeSet,
						Optional: true,
						Elem:     &Schema{Type: TypeString},
					},
					"cache_condition": &Schema{
						Type:     TypeString,
						Computed: true,
					},
				},
			},
		},
	}

	var readers = make(map[string]FieldReader)
	readers["state"] = &MapFieldReader{
		Schema: schema,
		Map: BasicMapReader(map[string]string{
			"backend.#":                                "1",
			"backend.2044061536.address":               "aws.amazon.com",
			"backend.2044061536.auto_loadbalance":      "true",
			"backend.2044061536.between_bytes_timeout": "10000",
			"backend.2044061536.connect_timeout":       "1000",
			"backend.2044061536.first_byte_timeout":    "15000",
			"backend.2044061536.max_conn":              "200",
			"backend.2044061536.name":                  "amazon docs",
			"backend.2044061536.port":                  "80",
			"backend.2044061536.ssl_check_cert":        "true",
			"backend.2044061536.weight":                "100",
			"default_ttl":                              "3600",
			"domain.#":                                 "1",
			"domain.142573846.comment":                 "tf-testing-domain",
			"domain.142573846.name":                    "tf-test-yada",
			"force_destroy":                            "true",
			"gzip.#":                                   "2", "backend.2044061536.error_threshold": "0",
			"gzip.3704620722.cache_condition":          "",
			"gzip.3704620722.content_types.#":          "0",
			"gzip.3704620722.extensions.#":             "2",
			"gzip.3704620722.extensions.253252853":     "js",
			"gzip.3704620722.extensions.3950613225":    "css",
			"gzip.3704620722.name":                     "gzip file types",
			"gzip.3820313126.cache_condition":          "",
			"gzip.3820313126.content_types.#":          "2",
			"gzip.3820313126.content_types.366283795":  "text/css",
			"gzip.3820313126.content_types.4008173114": "text/html",
			"gzip.3820313126.extensions.#":             "0",
			"gzip.3820313126.name":                     "gzip extensions",
			"id":   "1609878447410863288",
			"name": "tf-test-yada",
		}),
	}

	dfr := &DiffFieldReader{
		Schema: schema,
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"gzip.#": &terraform.ResourceAttrDiff{
					Old: "2", New: "1", NewComputed: false, NewRemoved: false,
				},

				"gzip.3704620722.extensions.3950613225": &terraform.ResourceAttrDiff{
					Old: "css", New: "", NewComputed: false, NewRemoved: true,
				},
				"gzip.3704620722.name": &terraform.ResourceAttrDiff{
					Old: "gzip file types", New: "", NewComputed: false, NewRemoved: true,
				},
				"gzip.3704620722.content_types.#": &terraform.ResourceAttrDiff{
					Old: "0", New: "0", NewComputed: false, NewRemoved: false,
				},
				"gzip.3704620722.extensions.#": &terraform.ResourceAttrDiff{
					Old: "2", New: "0", NewComputed: false, NewRemoved: false,
				},
				"gzip.3704620722.extensions.253252853": &terraform.ResourceAttrDiff{
					Old: "js", New: "", NewComputed: false, NewRemoved: true,
				},

				"gzip.3820313126.name": &terraform.ResourceAttrDiff{
					Old: "gzip extensions", New: "", NewComputed: false, NewRemoved: true,
				},
				"gzip.3820313126.extensions.#": &terraform.ResourceAttrDiff{
					Old: "0", New: "0", NewComputed: false, NewRemoved: false,
				},
				"gzip.3820313126.content_types.#": &terraform.ResourceAttrDiff{
					Old: "2", New: "0", NewComputed: false, NewRemoved: false,
				},
				"gzip.3820313126.content_types.4008173114": &terraform.ResourceAttrDiff{
					Old: "text/html", New: "", NewComputed: false, NewRemoved: true,
				},
				"gzip.3820313126.content_types.366283795": &terraform.ResourceAttrDiff{
					Old: "text/css", New: "", NewComputed: false, NewRemoved: true,
				},

				"gzip.3694165387.extensions.#": &terraform.ResourceAttrDiff{
					Old: "0", New: "3", NewComputed: false, NewRemoved: false,
				},
				"gzip.3694165387.extensions.3950613225": &terraform.ResourceAttrDiff{
					Old: "", New: "css", NewComputed: false, NewRemoved: false,
				},
				"gzip.3694165387.extensions.253252853": &terraform.ResourceAttrDiff{
					Old: "", New: "js", NewComputed: false, NewRemoved: false,
				},
				"gzip.3694165387.extensions.3010554278": &terraform.ResourceAttrDiff{
					Old: "", New: "html", NewComputed: false, NewRemoved: false,
				},
				"gzip.3694165387.content_types.#": &terraform.ResourceAttrDiff{
					Old: "0", New: "5", NewComputed: false, NewRemoved: false,
				},
				"gzip.3694165387.content_types.3132442313": &terraform.ResourceAttrDiff{
					Old: "", New: "application/x-javascript", NewComputed: false, NewRemoved: false,
				},
				"gzip.3694165387.content_types.3453298448": &terraform.ResourceAttrDiff{
					Old: "", New: "application/javascript", NewComputed: false, NewRemoved: false,
				},
				"gzip.3694165387.content_types.4008173114": &terraform.ResourceAttrDiff{
					Old: "", New: "text/html", NewComputed: false, NewRemoved: false,
				},
				"gzip.3694165387.content_types.366283795": &terraform.ResourceAttrDiff{
					Old: "", New: "text/css", NewComputed: false, NewRemoved: false,
				},
				"gzip.3694165387.content_types.1951959585": &terraform.ResourceAttrDiff{
					Old: "", New: "text/javascript", NewComputed: false, NewRemoved: false,
				},
				"gzip.3694165387.cache_condition": &terraform.ResourceAttrDiff{
					Old: "", New: "", NewComputed: true, NewRemoved: false,
				},
				"gzip.3694165387.name": &terraform.ResourceAttrDiff{
					Old: "", New: "all", NewComputed: false, NewRemoved: false,
				},
			},
			Destroy:        false,
			DestroyTainted: false,
		},
		Source: &MultiLevelFieldReader{
			Levels:  []string{"state", "config"},
			Readers: readers,
		},
	}

	result, err := dfr.ReadField([]string{"gzip"})
	if err != nil {
		t.Fatalf("ReadField failed: %s", err)
	}

	list := result.Value.(*Set).List()
	m := list[0].(map[string]interface{})

	expectedContentTypes := NewSet(HashSchema(&Schema{Type: TypeString}), []interface{}{
		"text/javascript",
		"application/x-javascript",
		"application/javascript",
		"text/css",
		"text/html",
	})
	if !m["content_types"].(*Set).Equal(expectedContentTypes) {
		t.Fatalf("ReadField returned unexpected content_types.\nGiven: %#v\nexpected: %#v",
			m["content_types"].(*Set).List(), expectedContentTypes.List())
	}
	delete(m, "content_types")

	expectedExtensions := NewSet(HashSchema(&Schema{Type: TypeString}), []interface{}{
		"js", "html", "css",
	})
	if !m["extensions"].(*Set).Equal(expectedExtensions) {
		t.Fatalf("ReadField returned unexpected extensions.\nGiven: %#v\nexpected: %#v",
			m["extensions"].(*Set).List(), expectedExtensions.List())
	}
	delete(m, "extensions")

	expectedGzip := map[string]interface{}{
		"name":            "all",
		"cache_condition": "",
	}
	if !reflect.DeepEqual(m, expectedGzip) {
		t.Fatalf("ReadField returned unexpected result.\nGiven: %#v\nexpected: %#v",
			m, expectedGzip)
	}
}
