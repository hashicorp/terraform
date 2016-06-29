package schema

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestMultiLevelFieldReaderReadFieldExact(t *testing.T) {
	cases := map[string]struct {
		Addr    []string
		Readers []FieldReader
		Level   string
		Result  FieldReadResult
	}{
		"specific": {
			Addr: []string{"foo"},

			Readers: []FieldReader{
				&MapFieldReader{
					Schema: map[string]*Schema{
						"foo": &Schema{Type: TypeString},
					},
					Map: BasicMapReader(map[string]string{
						"foo": "bar",
					}),
				},
				&MapFieldReader{
					Schema: map[string]*Schema{
						"foo": &Schema{Type: TypeString},
					},
					Map: BasicMapReader(map[string]string{
						"foo": "baz",
					}),
				},
				&MapFieldReader{
					Schema: map[string]*Schema{
						"foo": &Schema{Type: TypeString},
					},
					Map: BasicMapReader(map[string]string{}),
				},
			},

			Level: "1",
			Result: FieldReadResult{
				Value:  "baz",
				Exists: true,
			},
		},
	}

	for name, tc := range cases {
		readers := make(map[string]FieldReader)
		levels := make([]string, len(tc.Readers))
		for i, r := range tc.Readers {
			is := strconv.FormatInt(int64(i), 10)
			readers[is] = r
			levels[i] = is
		}

		r := &MultiLevelFieldReader{
			Readers: readers,
			Levels:  levels,
		}

		out, err := r.ReadFieldExact(tc.Addr, tc.Level)
		if err != nil {
			t.Fatalf("%s: err: %s", name, err)
		}

		if !reflect.DeepEqual(tc.Result, out) {
			t.Fatalf("%s: bad: %#v", name, out)
		}
	}
}

func TestMultiLevelFieldReaderReadFieldMerge(t *testing.T) {
	cases := map[string]struct {
		Addr    []string
		Readers []FieldReader
		Result  FieldReadResult
	}{
		"stringInDiff": {
			Addr: []string{"availability_zone"},

			Readers: []FieldReader{
				&DiffFieldReader{
					Schema: map[string]*Schema{
						"availability_zone": &Schema{Type: TypeString},
					},

					Source: &MapFieldReader{
						Schema: map[string]*Schema{
							"availability_zone": &Schema{Type: TypeString},
						},
						Map: BasicMapReader(map[string]string{
							"availability_zone": "foo",
						}),
					},

					Diff: &terraform.InstanceDiff{
						Attributes: map[string]*terraform.ResourceAttrDiff{
							"availability_zone": &terraform.ResourceAttrDiff{
								Old:         "foo",
								New:         "bar",
								RequiresNew: true,
							},
						},
					},
				},
			},

			Result: FieldReadResult{
				Value:  "bar",
				Exists: true,
			},
		},

		"lastLevelComputed": {
			Addr: []string{"availability_zone"},

			Readers: []FieldReader{
				&MapFieldReader{
					Schema: map[string]*Schema{
						"availability_zone": &Schema{Type: TypeString},
					},

					Map: BasicMapReader(map[string]string{
						"availability_zone": "foo",
					}),
				},

				&DiffFieldReader{
					Schema: map[string]*Schema{
						"availability_zone": &Schema{Type: TypeString},
					},

					Source: &MapFieldReader{
						Schema: map[string]*Schema{
							"availability_zone": &Schema{Type: TypeString},
						},

						Map: BasicMapReader(map[string]string{
							"availability_zone": "foo",
						}),
					},

					Diff: &terraform.InstanceDiff{
						Attributes: map[string]*terraform.ResourceAttrDiff{
							"availability_zone": &terraform.ResourceAttrDiff{
								Old:         "foo",
								New:         "bar",
								NewComputed: true,
							},
						},
					},
				},
			},

			Result: FieldReadResult{
				Value:    "",
				Exists:   true,
				Computed: true,
			},
		},

		"list of maps with removal in diff": {
			Addr: []string{"config_vars"},

			Readers: []FieldReader{
				&DiffFieldReader{
					Schema: map[string]*Schema{
						"config_vars": &Schema{
							Type: TypeList,
							Elem: &Schema{Type: TypeMap},
						},
					},

					Source: &MapFieldReader{
						Schema: map[string]*Schema{
							"config_vars": &Schema{
								Type: TypeList,
								Elem: &Schema{Type: TypeMap},
							},
						},

						Map: BasicMapReader(map[string]string{
							"config_vars.#":     "2",
							"config_vars.0.foo": "bar",
							"config_vars.0.bar": "bar",
							"config_vars.1.bar": "baz",
						}),
					},

					Diff: &terraform.InstanceDiff{
						Attributes: map[string]*terraform.ResourceAttrDiff{
							"config_vars.0.bar": &terraform.ResourceAttrDiff{
								NewRemoved: true,
							},
						},
					},
				},
			},

			Result: FieldReadResult{
				Value: []interface{}{
					map[string]interface{}{
						"foo": "bar",
					},
					map[string]interface{}{
						"bar": "baz",
					},
				},
				Exists: true,
			},
		},

		"first level only": {
			Addr: []string{"foo"},

			Readers: []FieldReader{
				&MapFieldReader{
					Schema: map[string]*Schema{
						"foo": &Schema{Type: TypeString},
					},
					Map: BasicMapReader(map[string]string{
						"foo": "bar",
					}),
				},
				&MapFieldReader{
					Schema: map[string]*Schema{
						"foo": &Schema{Type: TypeString},
					},
					Map: BasicMapReader(map[string]string{}),
				},
			},

			Result: FieldReadResult{
				Value:  "bar",
				Exists: true,
			},
		},
	}

	for name, tc := range cases {
		readers := make(map[string]FieldReader)
		levels := make([]string, len(tc.Readers))
		for i, r := range tc.Readers {
			is := strconv.FormatInt(int64(i), 10)
			readers[is] = r
			levels[i] = is
		}

		r := &MultiLevelFieldReader{
			Readers: readers,
			Levels:  levels,
		}

		out, err := r.ReadFieldMerge(tc.Addr, levels[len(levels)-1])
		if err != nil {
			t.Fatalf("%s: err: %s", name, err)
		}

		if !reflect.DeepEqual(tc.Result, out) {
			t.Fatalf("Case %s:\ngiven: %#v\nexpected: %#v", name, out, tc.Result)
		}
	}
}

func TestMultiLevelFieldReader_ReadField_SetInSet(t *testing.T) {
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
			"main_set.2813616083.inner_string_set.#":          "2",
			"main_set.2813616083.inner_string_set.2654390964": "blue",
			"main_set.2813616083.inner_string_set.3499814433": "green",
		}),
	}
	readers["diff"] = &DiffFieldReader{
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

	mr := &MultiLevelFieldReader{
		Levels: []string{
			"state",
			"diff",
		},

		Readers: readers,
	}

	result, err := mr.ReadField([]string{"main_set"})
	if err != nil {
		t.Fatalf("ReadField failed: %#v", err)
	}

	v := result.Value
	if v == nil {
		t.Fatal("Expected Value to be not nil")
	}
	list := v.(*Set).List()
	if len(list) != 1 {
		t.Fatalf("Expected exactly 1 instance, got %d", len(list))
	}
	if list[0] == nil {
		t.Fatalf("Expected value to be not nil: %#v", list)
	}

	m := list[0].(map[string]interface{})
	set := m["inner_string_set"].(*Set).List()

	expectedSet := NewSet(HashString, []interface{}{"blue", "green"}).List()

	if !reflect.DeepEqual(set, expectedSet) {
		t.Fatalf("Given: %#v\n\nExpected: %#v", set, expectedSet)
	}
}

func TestMultiLevelFieldReader_ReadField_SetInSet_complex(t *testing.T) {
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
	readers["diff"] = &DiffFieldReader{
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

	mr := &MultiLevelFieldReader{
		Levels: []string{
			"state",
			"diff",
		},

		Readers: readers,
	}

	result, err := mr.ReadFieldMerge([]string{"main_set"}, "diff")
	if err != nil {
		t.Fatalf("ReadFieldMerge failed: %#v", err)
	}

	v := result.Value
	if v == nil {
		t.Fatal("Expected Value to be not nil")
	}
	list := v.(*Set).List()
	if len(list) != 1 {
		t.Fatalf("Expected exactly 1 instance, got %d", len(list))
	}
	if list[0] == nil {
		t.Fatalf("Expected value to be not nil: %#v", list)
	}

	m := list[0].(map[string]interface{})
	set := m["inner_string_set"].(*Set).List()

	expectedSet := NewSet(HashString, []interface{}{"blue", "green"}).List()

	if !reflect.DeepEqual(set, expectedSet) {
		t.Fatalf("Given: %#v\n\nExpected: %#v", set, expectedSet)
	}
}

func TestMultiLevelFieldReader_ReadField_SetInList_simple(t *testing.T) {
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
		}),
	}
	readers["diff"] = &DiffFieldReader{
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

	mr := &MultiLevelFieldReader{
		Levels: []string{
			"state",
			"diff",
		},

		Readers: readers,
	}

	result, err := mr.ReadField([]string{"main_list"})
	if err != nil {
		t.Fatalf("ReadField failed: %#v", err)
	}

	v := result.Value
	if v == nil {
		t.Fatal("Expected Value to be not nil")
	}
	list := v.([]interface{})
	if len(list) != 1 {
		t.Fatalf("Expected exactly 1 instance, got %d", len(list))
	}
	if list[0] == nil {
		t.Fatalf("Expected value to be not nil: %#v", list)
	}

	m := list[0].(map[string]interface{})
	set := m["inner_string_set"].(*Set).List()

	expectedSet := NewSet(HashString, []interface{}{"blue", "green"}).List()

	if !reflect.DeepEqual(set, expectedSet) {
		t.Fatalf("Given: %#v\n\nExpected: %#v", set, expectedSet)
	}
}

func TestMultiLevelFieldReader_ReadField_SetInList_complex(t *testing.T) {
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
	readers["diff"] = &DiffFieldReader{
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

	mr := &MultiLevelFieldReader{
		Levels: []string{
			"state",
			"diff",
		},

		Readers: readers,
	}

	result, err := mr.ReadFieldMerge([]string{"main_list"}, "diff")
	if err != nil {
		t.Fatalf("ReadFieldMerge failed: %#v", err)
	}

	v := result.Value
	if v == nil {
		t.Fatal("Expected Value to be not nil")
	}
	list := v.([]interface{})
	if len(list) != 1 {
		t.Fatalf("Expected exactly 1 instance, got %d", len(list))
	}
	if list[0] == nil {
		t.Fatalf("Expected value to be not nil: %#v", list)
	}

	m := list[0].(map[string]interface{})
	set := m["inner_string_set"].(*Set).List()

	expectedSet := NewSet(HashString, []interface{}{"blue", "green"}).List()

	if !reflect.DeepEqual(set, expectedSet) {
		t.Fatalf("Given: %#v\n\nExpected: %#v", set, expectedSet)
	}
}
