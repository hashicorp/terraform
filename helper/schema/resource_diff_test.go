package schema

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/configs/hcl2shim"
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

// resourceDiffTestCase provides a test case struct for SetNew and SetDiff.
type resourceDiffTestCase struct {
	Name          string
	Schema        map[string]*Schema
	State         *terraform.InstanceState
	Config        *terraform.ResourceConfig
	Diff          *terraform.InstanceDiff
	Key           string
	OldValue      interface{}
	NewValue      interface{}
	Expected      *terraform.InstanceDiff
	ExpectedKeys  []string
	ExpectedError bool
}

// testDiffCases produces a list of test cases for use with SetNew and SetDiff.
func testDiffCases(t *testing.T, oldPrefix string, oldOffset int, computed bool) []resourceDiffTestCase {
	return []resourceDiffTestCase{
		resourceDiffTestCase{
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
						New: func() string {
							if computed {
								return ""
							}
							return "qux"
						}(),
						NewComputed: computed,
					},
				},
			},
		},
		resourceDiffTestCase{
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
				Attributes: func() map[string]*terraform.ResourceAttrDiff {
					result := map[string]*terraform.ResourceAttrDiff{}
					if computed {
						result["foo.#"] = &terraform.ResourceAttrDiff{
							Old:         "1",
							New:         "",
							NewComputed: true,
						}
					} else {
						result["foo.2800005064"] = &terraform.ResourceAttrDiff{
							Old: "",
							New: "qux",
						}
						result["foo.1996459178"] = &terraform.ResourceAttrDiff{
							Old:        "bar",
							New:        "",
							NewRemoved: true,
						}
					}
					return result
				}(),
			},
		},
		resourceDiffTestCase{
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
				Attributes: func() map[string]*terraform.ResourceAttrDiff {
					result := make(map[string]*terraform.ResourceAttrDiff)
					if computed {
						result["foo.#"] = &terraform.ResourceAttrDiff{
							Old:         "1",
							New:         "",
							NewComputed: true,
						}
					} else {
						result["foo.0"] = &terraform.ResourceAttrDiff{
							Old: "bar",
							New: "qux",
						}
					}
					return result
				}(),
			},
		},
		resourceDiffTestCase{
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
				Attributes: func() map[string]*terraform.ResourceAttrDiff {
					result := make(map[string]*terraform.ResourceAttrDiff)
					if computed {
						result["foo.%"] = &terraform.ResourceAttrDiff{
							Old:         "",
							New:         "",
							NewComputed: true,
						}
						result["foo.bar"] = &terraform.ResourceAttrDiff{
							Old:        "baz",
							New:        "",
							NewRemoved: true,
						}
					} else {
						result["foo.bar"] = &terraform.ResourceAttrDiff{
							Old: "baz",
							New: "quux",
						}
					}
					return result
				}(),
			},
		},
		resourceDiffTestCase{
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
			NewValue: "four",
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old: "bar",
						New: "baz",
					},
					"one": &terraform.ResourceAttrDiff{
						Old: "two",
						New: func() string {
							if computed {
								return ""
							}
							return "four"
						}(),
						NewComputed: computed,
					},
				},
			},
		},
		resourceDiffTestCase{
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
						New: func() string {
							if computed {
								return ""
							}
							return "three"
						}(),
						NewComputed: computed,
					},
				},
			},
		},
		resourceDiffTestCase{
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
				Attributes: func() map[string]*terraform.ResourceAttrDiff {
					result := make(map[string]*terraform.ResourceAttrDiff)
					if computed {
						result["top.#"] = &terraform.ResourceAttrDiff{
							Old:         "2",
							New:         "",
							NewComputed: true,
						}
					} else {
						result["top.#"] = &terraform.ResourceAttrDiff{
							Old: "2",
							New: "3",
						}
						result["top.5.foo"] = &terraform.ResourceAttrDiff{
							Old: "",
							New: "1",
						}
						result["top.5.bar"] = &terraform.ResourceAttrDiff{
							Old: "",
							New: "4",
						}
						result["top.25.foo"] = &terraform.ResourceAttrDiff{
							Old: "",
							New: "13",
						}
						result["top.25.bar"] = &terraform.ResourceAttrDiff{
							Old: "",
							New: "12",
						}
						result["top.43.foo"] = &terraform.ResourceAttrDiff{
							Old: "",
							New: "21",
						}
						result["top.43.bar"] = &terraform.ResourceAttrDiff{
							Old: "",
							New: "22",
						}
					}
					return result
				}(),
			},
		},
		resourceDiffTestCase{
			Name: "primitive, no diff, no refresh",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeString,
					Computed: true,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo": "bar",
				},
			},
			Config:   testConfig(t, map[string]interface{}{}),
			Diff:     &terraform.InstanceDiff{Attributes: map[string]*terraform.ResourceAttrDiff{}},
			Key:      "foo",
			NewValue: "baz",
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old: "bar",
						New: func() string {
							if computed {
								return ""
							}
							return "baz"
						}(),
						NewComputed: computed,
					},
				},
			},
		},
		resourceDiffTestCase{
			Name: "non-computed key, should error",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeString,
					Required: true,
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
			Key:           "foo",
			NewValue:      "qux",
			ExpectedError: true,
		},
		resourceDiffTestCase{
			Name: "bad key, should error",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeString,
					Required: true,
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
			Key:           "bad",
			NewValue:      "qux",
			ExpectedError: true,
		},
		resourceDiffTestCase{
			// NOTE: This case is technically impossible in the current
			// implementation, because optional+computed values never show up in the
			// diff, and we actually clear existing diffs when SetNew or
			// SetNewComputed is run.  This test is here to ensure that if either of
			// these behaviors change that we don't introduce regressions.
			Name: "NewRemoved in diff for Optional and Computed, should be fully overridden",
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
			Config: testConfig(t, map[string]interface{}{}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old:        "bar",
						New:        "",
						NewRemoved: true,
					},
				},
			},
			Key:      "foo",
			NewValue: "qux",
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old: "bar",
						New: func() string {
							if computed {
								return ""
							}
							return "qux"
						}(),
						NewComputed: computed,
					},
				},
			},
		},
		resourceDiffTestCase{
			Name: "NewComputed should always propagate",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeString,
					Computed: true,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo": "",
				},
				ID: "pre-existing",
			},
			Config:   testConfig(t, map[string]interface{}{}),
			Diff:     &terraform.InstanceDiff{Attributes: map[string]*terraform.ResourceAttrDiff{}},
			Key:      "foo",
			NewValue: "",
			Expected: &terraform.InstanceDiff{
				Attributes: func() map[string]*terraform.ResourceAttrDiff {
					if computed {
						return map[string]*terraform.ResourceAttrDiff{
							"foo": &terraform.ResourceAttrDiff{
								NewComputed: computed,
							},
						}
					}
					return map[string]*terraform.ResourceAttrDiff{}
				}(),
			},
		},
	}
}

func TestSetNew(t *testing.T) {
	testCases := testDiffCases(t, "", 0, false)
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			m := schemaMap(tc.Schema)
			d := newResourceDiff(tc.Schema, tc.Config, tc.State, tc.Diff)
			err := d.SetNew(tc.Key, tc.NewValue)
			switch {
			case err != nil && !tc.ExpectedError:
				t.Fatalf("bad: %s", err)
			case err == nil && tc.ExpectedError:
				t.Fatalf("Expected error, got none")
			case err != nil && tc.ExpectedError:
				return
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

func TestSetNewComputed(t *testing.T) {
	testCases := testDiffCases(t, "", 0, true)
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			m := schemaMap(tc.Schema)
			d := newResourceDiff(tc.Schema, tc.Config, tc.State, tc.Diff)
			err := d.SetNewComputed(tc.Key)
			switch {
			case err != nil && !tc.ExpectedError:
				t.Fatalf("bad: %s", err)
			case err == nil && tc.ExpectedError:
				t.Fatalf("Expected error, got none")
			case err != nil && tc.ExpectedError:
				return
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

func TestForceNew(t *testing.T) {
	cases := []resourceDiffTestCase{
		resourceDiffTestCase{
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
			Key: "foo",
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old:         "bar",
						New:         "baz",
						RequiresNew: true,
					},
				},
			},
		},
		resourceDiffTestCase{
			Name: "no change, should error",
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
				"foo": "bar",
			}),
			ExpectedError: true,
		},
		resourceDiffTestCase{
			Name: "basic primitive, non-computed key",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeString,
					Required: true,
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
			Key: "foo",
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old:         "bar",
						New:         "baz",
						RequiresNew: true,
					},
				},
			},
		},
		resourceDiffTestCase{
			Name: "nested field",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeList,
					Required: true,
					MaxItems: 1,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"bar": {
								Type:     TypeString,
								Optional: true,
							},
							"baz": {
								Type:     TypeString,
								Optional: true,
							},
						},
					},
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo.#":     "1",
					"foo.0.bar": "abc",
					"foo.0.baz": "xyz",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"foo": []interface{}{
					map[string]interface{}{
						"bar": "abcdefg",
						"baz": "changed",
					},
				},
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo.0.bar": &terraform.ResourceAttrDiff{
						Old: "abc",
						New: "abcdefg",
					},
					"foo.0.baz": &terraform.ResourceAttrDiff{
						Old: "xyz",
						New: "changed",
					},
				},
			},
			Key: "foo.0.baz",
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo.0.bar": &terraform.ResourceAttrDiff{
						Old: "abc",
						New: "abcdefg",
					},
					"foo.0.baz": &terraform.ResourceAttrDiff{
						Old:         "xyz",
						New:         "changed",
						RequiresNew: true,
					},
				},
			},
		},
		resourceDiffTestCase{
			Name: "preserve NewRemoved on existing diff",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeString,
					Optional: true,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo": "bar",
				},
			},
			Config: testConfig(t, map[string]interface{}{}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old:        "bar",
						New:        "",
						NewRemoved: true,
					},
				},
			},
			Key: "foo",
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old:         "bar",
						New:         "",
						RequiresNew: true,
						NewRemoved:  true,
					},
				},
			},
		},
		resourceDiffTestCase{
			Name: "nested field, preserve original diff without zero values",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeList,
					Required: true,
					MaxItems: 1,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"bar": {
								Type:     TypeString,
								Optional: true,
							},
							"baz": {
								Type:     TypeInt,
								Optional: true,
							},
						},
					},
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo.#":     "1",
					"foo.0.bar": "abc",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"foo": []interface{}{
					map[string]interface{}{
						"bar": "abcdefg",
					},
				},
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo.0.bar": &terraform.ResourceAttrDiff{
						Old: "abc",
						New: "abcdefg",
					},
				},
			},
			Key: "foo.0.bar",
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo.0.bar": &terraform.ResourceAttrDiff{
						Old:         "abc",
						New:         "abcdefg",
						RequiresNew: true,
					},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			m := schemaMap(tc.Schema)
			d := newResourceDiff(m, tc.Config, tc.State, tc.Diff)
			err := d.ForceNew(tc.Key)
			switch {
			case err != nil && !tc.ExpectedError:
				t.Fatalf("bad: %s", err)
			case err == nil && tc.ExpectedError:
				t.Fatalf("Expected error, got none")
			case err != nil && tc.ExpectedError:
				return
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

func TestClear(t *testing.T) {
	cases := []resourceDiffTestCase{
		resourceDiffTestCase{
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
			Expected: &terraform.InstanceDiff{Attributes: map[string]*terraform.ResourceAttrDiff{}},
		},
		resourceDiffTestCase{
			Name: "non-computed key, should error",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeString,
					Required: true,
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
			Key:           "foo",
			ExpectedError: true,
		},
		resourceDiffTestCase{
			Name: "multi-value, one removed",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
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
			Key: "one",
			Expected: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old: "bar",
						New: "baz",
					},
				},
			},
		},
		resourceDiffTestCase{
			Name: "basic sub-block diff",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeList,
					Optional: true,
					Computed: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"bar": &Schema{
								Type:     TypeString,
								Optional: true,
								Computed: true,
							},
							"baz": &Schema{
								Type:     TypeString,
								Optional: true,
								Computed: true,
							},
						},
					},
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo.0.bar": "bar1",
					"foo.0.baz": "baz1",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"foo": []interface{}{
					map[string]interface{}{
						"bar": "bar2",
						"baz": "baz1",
					},
				},
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo.0.bar": &terraform.ResourceAttrDiff{
						Old: "bar1",
						New: "bar2",
					},
				},
			},
			Key:      "foo.0.bar",
			Expected: &terraform.InstanceDiff{Attributes: map[string]*terraform.ResourceAttrDiff{}},
		},
		resourceDiffTestCase{
			Name: "sub-block diff only partial clear",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeList,
					Optional: true,
					Computed: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"bar": &Schema{
								Type:     TypeString,
								Optional: true,
								Computed: true,
							},
							"baz": &Schema{
								Type:     TypeString,
								Optional: true,
								Computed: true,
							},
						},
					},
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"foo.0.bar": "bar1",
					"foo.0.baz": "baz1",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"foo": []interface{}{
					map[string]interface{}{
						"bar": "bar2",
						"baz": "baz2",
					},
				},
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo.0.bar": &terraform.ResourceAttrDiff{
						Old: "bar1",
						New: "bar2",
					},
					"foo.0.baz": &terraform.ResourceAttrDiff{
						Old: "baz1",
						New: "baz2",
					},
				},
			},
			Key: "foo.0.bar",
			Expected: &terraform.InstanceDiff{Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo.0.baz": &terraform.ResourceAttrDiff{
					Old: "baz1",
					New: "baz2",
				},
			}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			m := schemaMap(tc.Schema)
			d := newResourceDiff(m, tc.Config, tc.State, tc.Diff)
			err := d.Clear(tc.Key)
			switch {
			case err != nil && !tc.ExpectedError:
				t.Fatalf("bad: %s", err)
			case err == nil && tc.ExpectedError:
				t.Fatalf("Expected error, got none")
			case err != nil && tc.ExpectedError:
				return
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

func TestGetChangedKeysPrefix(t *testing.T) {
	cases := []resourceDiffTestCase{
		resourceDiffTestCase{
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
			Key: "foo",
			ExpectedKeys: []string{
				"foo",
			},
		},
		resourceDiffTestCase{
			Name: "nested field filtering",
			Schema: map[string]*Schema{
				"testfield": &Schema{
					Type:     TypeString,
					Required: true,
				},
				"foo": &Schema{
					Type:     TypeList,
					Required: true,
					MaxItems: 1,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"bar": {
								Type:     TypeString,
								Optional: true,
							},
							"baz": {
								Type:     TypeString,
								Optional: true,
							},
						},
					},
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"testfield": "blablah",
					"foo.#":     "1",
					"foo.0.bar": "abc",
					"foo.0.baz": "xyz",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"testfield": "modified",
				"foo": []interface{}{
					map[string]interface{}{
						"bar": "abcdefg",
						"baz": "changed",
					},
				},
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"testfield": &terraform.ResourceAttrDiff{
						Old: "blablah",
						New: "modified",
					},
					"foo.0.bar": &terraform.ResourceAttrDiff{
						Old: "abc",
						New: "abcdefg",
					},
					"foo.0.baz": &terraform.ResourceAttrDiff{
						Old: "xyz",
						New: "changed",
					},
				},
			},
			Key: "foo",
			ExpectedKeys: []string{
				"foo.0.bar",
				"foo.0.baz",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			m := schemaMap(tc.Schema)
			d := newResourceDiff(m, tc.Config, tc.State, tc.Diff)
			keys := d.GetChangedKeysPrefix(tc.Key)

			for _, k := range d.UpdatedKeys() {
				if err := m.diff(k, m[k], tc.Diff, d, false); err != nil {
					t.Fatalf("bad: %s", err)
				}
			}

			sort.Strings(keys)

			if !reflect.DeepEqual(tc.ExpectedKeys, keys) {
				t.Fatalf("Expected %s, got %s", spew.Sdump(tc.ExpectedKeys), spew.Sdump(keys))
			}
		})
	}
}

func TestResourceDiffGetOkExists(t *testing.T) {
	cases := []struct {
		Name   string
		Schema map[string]*Schema
		State  *terraform.InstanceState
		Config *terraform.ResourceConfig
		Diff   *terraform.InstanceDiff
		Key    string
		Value  interface{}
		Ok     bool
	}{
		/*
		 * Primitives
		 */
		{
			Name: "string-literal-empty",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State:  nil,
			Config: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old: "",
						New: "",
					},
				},
			},

			Key:   "availability_zone",
			Value: "",
			Ok:    true,
		},

		{
			Name: "string-computed-empty",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State:  nil,
			Config: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Key:   "availability_zone",
			Value: "",
			Ok:    false,
		},

		{
			Name: "string-optional-computed-nil-diff",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State:  nil,
			Config: nil,

			Diff: nil,

			Key:   "availability_zone",
			Value: "",
			Ok:    false,
		},

		/*
		 * Lists
		 */

		{
			Name: "list-optional",
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeList,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State:  nil,
			Config: nil,

			Diff: nil,

			Key:   "ports",
			Value: []interface{}{},
			Ok:    false,
		},

		/*
		 * Map
		 */

		{
			Name: "map-optional",
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeMap,
					Optional: true,
				},
			},

			State:  nil,
			Config: nil,

			Diff: nil,

			Key:   "ports",
			Value: map[string]interface{}{},
			Ok:    false,
		},

		/*
		 * Set
		 */

		{
			Name: "set-optional",
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(a interface{}) int { return a.(int) },
				},
			},

			State:  nil,
			Config: nil,

			Diff: nil,

			Key:   "ports",
			Value: []interface{}{},
			Ok:    false,
		},

		{
			Name: "set-optional-key",
			Schema: map[string]*Schema{
				"ports": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(a interface{}) int { return a.(int) },
				},
			},

			State:  nil,
			Config: nil,

			Diff: nil,

			Key:   "ports.0",
			Value: 0,
			Ok:    false,
		},

		{
			Name: "bool-literal-empty",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeBool,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State:  nil,
			Config: nil,
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old: "",
						New: "",
					},
				},
			},

			Key:   "availability_zone",
			Value: false,
			Ok:    true,
		},

		{
			Name: "bool-literal-set",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeBool,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State:  nil,
			Config: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						New: "true",
					},
				},
			},

			Key:   "availability_zone",
			Value: true,
			Ok:    true,
		},
		{
			Name: "value-in-config",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"availability_zone": "foo",
			}),

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{},
			},

			Key:   "availability_zone",
			Value: "foo",
			Ok:    true,
		},
		{
			Name: "new-value-in-config",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: nil,
			Config: testConfig(t, map[string]interface{}{
				"availability_zone": "foo",
			}),

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old: "",
						New: "foo",
					},
				},
			},

			Key:   "availability_zone",
			Value: "foo",
			Ok:    true,
		},
		{
			Name: "optional-computed-value-in-config",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"availability_zone": "bar",
			}),

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old: "foo",
						New: "bar",
					},
				},
			},

			Key:   "availability_zone",
			Value: "bar",
			Ok:    true,
		},
		{
			Name: "removed-value",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},
			Config: testConfig(t, map[string]interface{}{}),

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:        "foo",
						New:        "",
						NewRemoved: true,
					},
				},
			},

			Key:   "availability_zone",
			Value: "",
			Ok:    true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			d := newResourceDiff(tc.Schema, tc.Config, tc.State, tc.Diff)

			v, ok := d.GetOkExists(tc.Key)
			if s, ok := v.(*Set); ok {
				v = s.List()
			}

			if !reflect.DeepEqual(v, tc.Value) {
				t.Fatalf("Bad %s: \n%#v", tc.Name, v)
			}
			if ok != tc.Ok {
				t.Fatalf("%s: expected ok: %t, got: %t", tc.Name, tc.Ok, ok)
			}
		})
	}
}

func TestResourceDiffGetOkExistsSetNew(t *testing.T) {
	tc := struct {
		Schema map[string]*Schema
		State  *terraform.InstanceState
		Diff   *terraform.InstanceDiff
		Key    string
		Value  interface{}
		Ok     bool
	}{
		Schema: map[string]*Schema{
			"availability_zone": {
				Type:     TypeString,
				Optional: true,
				Computed: true,
			},
		},

		State: nil,

		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{},
		},

		Key:   "availability_zone",
		Value: "foobar",
		Ok:    true,
	}

	d := newResourceDiff(tc.Schema, testConfig(t, map[string]interface{}{}), tc.State, tc.Diff)
	d.SetNew(tc.Key, tc.Value)

	v, ok := d.GetOkExists(tc.Key)
	if s, ok := v.(*Set); ok {
		v = s.List()
	}

	if !reflect.DeepEqual(v, tc.Value) {
		t.Fatalf("Bad: \n%#v", v)
	}
	if ok != tc.Ok {
		t.Fatalf("expected ok: %t, got: %t", tc.Ok, ok)
	}
}

func TestResourceDiffGetOkExistsSetNewComputed(t *testing.T) {
	tc := struct {
		Schema map[string]*Schema
		State  *terraform.InstanceState
		Diff   *terraform.InstanceDiff
		Key    string
		Value  interface{}
		Ok     bool
	}{
		Schema: map[string]*Schema{
			"availability_zone": {
				Type:     TypeString,
				Optional: true,
				Computed: true,
			},
		},

		State: &terraform.InstanceState{
			Attributes: map[string]string{
				"availability_zone": "foo",
			},
		},

		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{},
		},

		Key:   "availability_zone",
		Value: "foobar",
		Ok:    false,
	}

	d := newResourceDiff(tc.Schema, testConfig(t, map[string]interface{}{}), tc.State, tc.Diff)
	d.SetNewComputed(tc.Key)

	_, ok := d.GetOkExists(tc.Key)

	if ok != tc.Ok {
		t.Fatalf("expected ok: %t, got: %t", tc.Ok, ok)
	}
}

func TestResourceDiffNewValueKnown(t *testing.T) {
	cases := []struct {
		Name     string
		Schema   map[string]*Schema
		State    *terraform.InstanceState
		Config   *terraform.ResourceConfig
		Diff     *terraform.InstanceDiff
		Key      string
		Expected bool
	}{
		{
			Name: "in config, no state",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
				},
			},
			State: nil,
			Config: testConfig(t, map[string]interface{}{
				"availability_zone": "foo",
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old: "",
						New: "foo",
					},
				},
			},
			Key:      "availability_zone",
			Expected: true,
		},
		{
			Name: "in config, has state, no diff",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"availability_zone": "foo",
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{},
			},
			Key:      "availability_zone",
			Expected: true,
		},
		{
			Name: "computed attribute, in state, no diff",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Computed: true,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},
			Config: testConfig(t, map[string]interface{}{}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{},
			},
			Key:      "availability_zone",
			Expected: true,
		},
		{
			Name: "optional and computed attribute, in state, no config",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},
			Config: testConfig(t, map[string]interface{}{}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{},
			},
			Key:      "availability_zone",
			Expected: true,
		},
		{
			Name: "optional and computed attribute, in state, with config",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},
			Config: testConfig(t, map[string]interface{}{
				"availability_zone": "foo",
			}),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{},
			},
			Key:      "availability_zone",
			Expected: true,
		},
		{
			Name: "computed value, through config reader",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},
			Config: testConfig(
				t,
				map[string]interface{}{
					"availability_zone": hcl2shim.UnknownVariableValue,
				},
			),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{},
			},
			Key:      "availability_zone",
			Expected: false,
		},
		{
			Name: "computed value, through diff reader",
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
				},
			},
			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},
			Config: testConfig(
				t,
				map[string]interface{}{
					"availability_zone": hcl2shim.UnknownVariableValue,
				},
			),
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old:         "foo",
						New:         "",
						NewComputed: true,
					},
				},
			},
			Key:      "availability_zone",
			Expected: false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			d := newResourceDiff(tc.Schema, tc.Config, tc.State, tc.Diff)

			actual := d.NewValueKnown(tc.Key)
			if tc.Expected != actual {
				t.Fatalf("%s: expected ok: %t, got: %t", tc.Name, tc.Expected, actual)
			}
		})
	}
}

func TestResourceDiffNewValueKnownSetNew(t *testing.T) {
	tc := struct {
		Schema   map[string]*Schema
		State    *terraform.InstanceState
		Config   *terraform.ResourceConfig
		Diff     *terraform.InstanceDiff
		Key      string
		Value    interface{}
		Expected bool
	}{
		Schema: map[string]*Schema{
			"availability_zone": {
				Type:     TypeString,
				Optional: true,
				Computed: true,
			},
		},
		State: &terraform.InstanceState{
			Attributes: map[string]string{
				"availability_zone": "foo",
			},
		},
		Config: testConfig(
			t,
			map[string]interface{}{
				"availability_zone": hcl2shim.UnknownVariableValue,
			},
		),
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"availability_zone": {
					Old:         "foo",
					New:         "",
					NewComputed: true,
				},
			},
		},
		Key:      "availability_zone",
		Value:    "bar",
		Expected: true,
	}

	d := newResourceDiff(tc.Schema, tc.Config, tc.State, tc.Diff)
	d.SetNew(tc.Key, tc.Value)

	actual := d.NewValueKnown(tc.Key)
	if tc.Expected != actual {
		t.Fatalf("expected ok: %t, got: %t", tc.Expected, actual)
	}
}

func TestResourceDiffNewValueKnownSetNewComputed(t *testing.T) {
	tc := struct {
		Schema   map[string]*Schema
		State    *terraform.InstanceState
		Config   *terraform.ResourceConfig
		Diff     *terraform.InstanceDiff
		Key      string
		Expected bool
	}{
		Schema: map[string]*Schema{
			"availability_zone": {
				Type:     TypeString,
				Computed: true,
			},
		},
		State: &terraform.InstanceState{
			Attributes: map[string]string{
				"availability_zone": "foo",
			},
		},
		Config: testConfig(t, map[string]interface{}{}),
		Diff: &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{},
		},
		Key:      "availability_zone",
		Expected: false,
	}

	d := newResourceDiff(tc.Schema, tc.Config, tc.State, tc.Diff)
	d.SetNewComputed(tc.Key)

	actual := d.NewValueKnown(tc.Key)
	if tc.Expected != actual {
		t.Fatalf("expected ok: %t, got: %t", tc.Expected, actual)
	}
}
