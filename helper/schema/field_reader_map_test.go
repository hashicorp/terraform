package schema

import (
	"reflect"
	"testing"
)

func TestMapFieldReader_impl(t *testing.T) {
	var _ FieldReader = new(MapFieldReader)
}

func TestMapFieldReader(t *testing.T) {
	testFieldReader(t, func(s map[string]*Schema) FieldReader {
		return &MapFieldReader{
			Schema: s,

			Map: BasicMapReader(map[string]string{
				"bool":   "true",
				"int":    "42",
				"float":  "3.1415",
				"string": "string",

				"list.#": "2",
				"list.0": "foo",
				"list.1": "bar",

				"listInt.#": "2",
				"listInt.0": "21",
				"listInt.1": "42",

				"map.%":   "2",
				"map.foo": "bar",
				"map.bar": "baz",

				"set.#":  "2",
				"set.10": "10",
				"set.50": "50",

				"setDeep.#":        "2",
				"setDeep.10.index": "10",
				"setDeep.10.value": "foo",
				"setDeep.50.index": "50",
				"setDeep.50.value": "bar",
			}),
		}
	})
}

func TestMapFieldReader_extra(t *testing.T) {
	r := &MapFieldReader{
		Schema: map[string]*Schema{
			"mapDel":   &Schema{Type: TypeMap},
			"mapEmpty": &Schema{Type: TypeMap},
		},

		Map: BasicMapReader(map[string]string{
			"mapDel": "",

			"mapEmpty.%": "0",
		}),
	}

	cases := map[string]struct {
		Addr        []string
		Out         interface{}
		OutOk       bool
		OutComputed bool
		OutErr      bool
	}{
		"mapDel": {
			[]string{"mapDel"},
			map[string]interface{}{},
			true,
			false,
			false,
		},

		"mapEmpty": {
			[]string{"mapEmpty"},
			map[string]interface{}{},
			true,
			false,
			false,
		},
	}

	for name, tc := range cases {
		out, err := r.ReadField(tc.Addr)
		if err != nil != tc.OutErr {
			t.Fatalf("%s: err: %s", name, err)
		}
		if out.Computed != tc.OutComputed {
			t.Fatalf("%s: err: %#v", name, out.Computed)
		}

		if s, ok := out.Value.(*Set); ok {
			// If it is a set, convert to a list so its more easily checked.
			out.Value = s.List()
		}

		if !reflect.DeepEqual(out.Value, tc.Out) {
			t.Fatalf("%s: out: %#v", name, out.Value)
		}
		if out.Exists != tc.OutOk {
			t.Fatalf("%s: outOk: %#v", name, out.Exists)
		}
	}
}

func TestMapFieldReader_SetInSet(t *testing.T) {
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

	r := &MapFieldReader{
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

	result, err := r.ReadField([]string{"main_set"})
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

func TestMapFieldReader_SetInList(t *testing.T) {
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

	r := &MapFieldReader{
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

	result, err := r.ReadField([]string{"main_list"})
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

func TestMapFieldReader_SetInList_complex(t *testing.T) {
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

	r := &MapFieldReader{
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

	result, err := r.ReadField([]string{"main_list"})
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

func TestMapFieldReader_readSet_SetInSet(t *testing.T) {
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

	r := &MapFieldReader{
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

	result, err := r.readSet([]string{"main_set"}, schema["main_set"])
	if err != nil {
		t.Fatalf("readSet failed: %#v", err)
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
