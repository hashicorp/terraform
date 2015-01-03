package schema

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestDiffFieldReader_impl(t *testing.T) {
	var _ FieldReader = new(DiffFieldReader)
}

func TestDiffFieldReader(t *testing.T) {
	r := &DiffFieldReader{
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

				"mapRemove.bar": &terraform.ResourceAttrDiff{
					NewRemoved: true,
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

				"listMap.0.bar": &terraform.ResourceAttrDiff{
					NewRemoved: true,
				},

				"setChange.10.value": &terraform.ResourceAttrDiff{
					Old: "50",
					New: "80",
				},
			},
		},

		Source: &MapFieldReader{
			Map: map[string]string{
				"listMap.#":     "2",
				"listMap.0.foo": "bar",
				"listMap.0.bar": "baz",
				"listMap.1.baz": "baz",

				"mapRemove.foo": "bar",
				"mapRemove.bar": "bar",

				"setChange.#":        "1",
				"setChange.10.index": "10",
				"setChange.10.value": "50",
			},
		},
	}

	cases := map[string]struct {
		Addr   []string
		Schema *Schema
		Result FieldReadResult
		Err    bool
	}{
		"noexist": {
			[]string{"boolNOPE"},
			&Schema{Type: TypeBool},
			FieldReadResult{
				Value:    nil,
				Exists:   false,
				Computed: false,
			},
			false,
		},

		"bool": {
			[]string{"bool"},
			&Schema{Type: TypeBool},
			FieldReadResult{
				Value:    true,
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"int": {
			[]string{"int"},
			&Schema{Type: TypeInt},
			FieldReadResult{
				Value:    42,
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"string": {
			[]string{"string"},
			&Schema{Type: TypeString},
			FieldReadResult{
				Value:    "string",
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"stringComputed": {
			[]string{"stringComputed"},
			&Schema{Type: TypeString},
			FieldReadResult{
				Value:    "",
				Exists:   true,
				Computed: true,
			},
			false,
		},

		"list": {
			[]string{"list"},
			&Schema{
				Type: TypeList,
				Elem: &Schema{Type: TypeString},
			},
			FieldReadResult{
				Value: []interface{}{
					"foo",
					"bar",
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"listInt": {
			[]string{"listInt"},
			&Schema{
				Type: TypeList,
				Elem: &Schema{Type: TypeInt},
			},
			FieldReadResult{
				Value: []interface{}{
					21,
					42,
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"map": {
			[]string{"map"},
			&Schema{Type: TypeMap},
			FieldReadResult{
				Value: map[string]interface{}{
					"foo": "bar",
					"bar": "baz",
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"mapelem": {
			[]string{"map", "foo"},
			&Schema{Type: TypeString},
			FieldReadResult{
				Value:    "bar",
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"mapRemove": {
			[]string{"mapRemove"},
			&Schema{Type: TypeMap},
			FieldReadResult{
				Value: map[string]interface{}{
					"foo": "bar",
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"set": {
			[]string{"set"},
			&Schema{
				Type: TypeSet,
				Elem: &Schema{Type: TypeInt},
				Set: func(a interface{}) int {
					return a.(int)
				},
			},
			FieldReadResult{
				Value:    []interface{}{10, 50},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"setDeep": {
			[]string{"setDeep"},
			&Schema{
				Type: TypeSet,
				Elem: &Resource{
					Schema: map[string]*Schema{
						"index": &Schema{Type: TypeInt},
						"value": &Schema{Type: TypeString},
					},
				},
				Set: func(a interface{}) int {
					return a.(map[string]interface{})["index"].(int)
				},
			},
			FieldReadResult{
				Value: []interface{}{
					map[string]interface{}{
						"index": 10,
						"value": "foo",
					},
					map[string]interface{}{
						"index": 50,
						"value": "bar",
					},
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"listMapRemoval": {
			[]string{"listMap"},
			&Schema{
				Type: TypeList,
				Elem: &Schema{
					Type: TypeMap,
				},
			},
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

		"setChange": {
			[]string{"setChange"},
			&Schema{
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
	}

	for name, tc := range cases {
		out, err := r.ReadField(tc.Addr, tc.Schema)
		if (err != nil) != tc.Err {
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
