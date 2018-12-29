package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/config"
)

func TestStateValidate(t *testing.T) {
	cases := map[string]struct {
		In  *State
		Err bool
	}{
		"empty state": {
			&State{},
			false,
		},

		"multiple modules": {
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root", "foo"},
					},
					&ModuleState{
						Path: []string{"root", "foo"},
					},
				},
			},
			true,
		},
	}

	for name, tc := range cases {
		// Init the state
		tc.In.init()

		err := tc.In.Validate()
		if (err != nil) != tc.Err {
			t.Fatalf("%s: err: %s", name, err)
		}
	}
}

func TestStateAddModule(t *testing.T) {
	cases := []struct {
		In  []addrs.ModuleInstance
		Out [][]string
	}{
		{
			[]addrs.ModuleInstance{
				addrs.RootModuleInstance,
				addrs.RootModuleInstance.Child("child", addrs.NoKey),
			},
			[][]string{
				[]string{"root"},
				[]string{"root", "child"},
			},
		},

		{
			[]addrs.ModuleInstance{
				addrs.RootModuleInstance.Child("foo", addrs.NoKey).Child("bar", addrs.NoKey),
				addrs.RootModuleInstance.Child("foo", addrs.NoKey),
				addrs.RootModuleInstance,
				addrs.RootModuleInstance.Child("bar", addrs.NoKey),
			},
			[][]string{
				[]string{"root"},
				[]string{"root", "bar"},
				[]string{"root", "foo"},
				[]string{"root", "foo", "bar"},
			},
		},
		// Same last element, different middle element
		{
			[]addrs.ModuleInstance{
				addrs.RootModuleInstance.Child("foo", addrs.NoKey).Child("bar", addrs.NoKey), // This one should sort after...
				addrs.RootModuleInstance.Child("foo", addrs.NoKey),
				addrs.RootModuleInstance,
				addrs.RootModuleInstance.Child("bar", addrs.NoKey).Child("bar", addrs.NoKey), // ...this one.
				addrs.RootModuleInstance.Child("bar", addrs.NoKey),
			},
			[][]string{
				[]string{"root"},
				[]string{"root", "bar"},
				[]string{"root", "foo"},
				[]string{"root", "bar", "bar"},
				[]string{"root", "foo", "bar"},
			},
		},
	}

	for _, tc := range cases {
		s := new(State)
		for _, p := range tc.In {
			s.AddModule(p)
		}

		actual := make([][]string, 0, len(tc.In))
		for _, m := range s.Modules {
			actual = append(actual, m.Path)
		}

		if !reflect.DeepEqual(actual, tc.Out) {
			t.Fatalf("wrong result\ninput: %sgot:   %#v\nwant:  %#v", spew.Sdump(tc.In), actual, tc.Out)
		}
	}
}

func TestStateOutputTypeRoundTrip(t *testing.T) {
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root"},
				Outputs: map[string]*OutputState{
					"string_output": &OutputState{
						Value: "String Value",
						Type:  "string",
					},
				},
			},
		},
	}
	state.init()

	buf := new(bytes.Buffer)
	if err := WriteState(state, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	roundTripped, err := ReadState(buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(state, roundTripped) {
		t.Logf("expected:\n%#v", state)
		t.Fatalf("got:\n%#v", roundTripped)
	}
}

func TestStateDeepCopy(t *testing.T) {
	cases := []struct {
		State *State
	}{
		// Nil
		{nil},

		// Version
		{
			&State{Version: 5},
		},
		// TFVersion
		{
			&State{TFVersion: "5"},
		},
		// Modules
		{
			&State{
				Version: 6,
				Modules: []*ModuleState{
					&ModuleState{
						Path: rootModulePath,
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Primary: &InstanceState{
									Meta: map[string]interface{}{},
								},
							},
						},
					},
				},
			},
		},
		// Deposed
		// The nil values shouldn't be there if the State was properly init'ed,
		// but the Copy should still work anyway.
		{
			&State{
				Version: 6,
				Modules: []*ModuleState{
					&ModuleState{
						Path: rootModulePath,
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Primary: &InstanceState{
									Meta: map[string]interface{}{},
								},
								Deposed: []*InstanceState{
									{ID: "test"},
									nil,
								},
							},
						},
					},
				},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("copy-%d", i), func(t *testing.T) {
			actual := tc.State.DeepCopy()
			expected := tc.State
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("Expected: %#v\nRecevied: %#v\n", expected, actual)
			}
		})
	}
}

func TestStateEqual(t *testing.T) {
	cases := []struct {
		Name     string
		Result   bool
		One, Two *State
	}{
		// Nils
		{
			"one nil",
			false,
			nil,
			&State{Version: 2},
		},

		{
			"both nil",
			true,
			nil,
			nil,
		},

		// Different versions
		{
			"different state versions",
			false,
			&State{Version: 5},
			&State{Version: 2},
		},

		// Different modules
		{
			"different module states",
			false,
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root"},
					},
				},
			},
			&State{},
		},

		{
			"same module states",
			true,
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root"},
					},
				},
			},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root"},
					},
				},
			},
		},

		// Meta differs
		{
			"differing meta values with primitives",
			false,
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: rootModulePath,
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Primary: &InstanceState{
									Meta: map[string]interface{}{
										"schema_version": "1",
									},
								},
							},
						},
					},
				},
			},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: rootModulePath,
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Primary: &InstanceState{
									Meta: map[string]interface{}{
										"schema_version": "2",
									},
								},
							},
						},
					},
				},
			},
		},

		// Meta with complex types
		{
			"same meta with complex types",
			true,
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: rootModulePath,
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Primary: &InstanceState{
									Meta: map[string]interface{}{
										"timeouts": map[string]interface{}{
											"create": 42,
											"read":   "27",
										},
									},
								},
							},
						},
					},
				},
			},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: rootModulePath,
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Primary: &InstanceState{
									Meta: map[string]interface{}{
										"timeouts": map[string]interface{}{
											"create": 42,
											"read":   "27",
										},
									},
								},
							},
						},
					},
				},
			},
		},

		// Meta with complex types that have been altered during serialization
		{
			"same meta with complex types that have been json-ified",
			true,
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: rootModulePath,
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Primary: &InstanceState{
									Meta: map[string]interface{}{
										"timeouts": map[string]interface{}{
											"create": int(42),
											"read":   "27",
										},
									},
								},
							},
						},
					},
				},
			},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: rootModulePath,
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Primary: &InstanceState{
									Meta: map[string]interface{}{
										"timeouts": map[string]interface{}{
											"create": float64(42),
											"read":   "27",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			if tc.One.Equal(tc.Two) != tc.Result {
				t.Fatalf("Bad: %d\n\n%s\n\n%s", i, tc.One.String(), tc.Two.String())
			}
			if tc.Two.Equal(tc.One) != tc.Result {
				t.Fatalf("Bad: %d\n\n%s\n\n%s", i, tc.One.String(), tc.Two.String())
			}
		})
	}
}

func TestStateCompareAges(t *testing.T) {
	cases := []struct {
		Result   StateAgeComparison
		Err      bool
		One, Two *State
	}{
		{
			StateAgeEqual, false,
			&State{
				Lineage: "1",
				Serial:  2,
			},
			&State{
				Lineage: "1",
				Serial:  2,
			},
		},
		{
			StateAgeReceiverOlder, false,
			&State{
				Lineage: "1",
				Serial:  2,
			},
			&State{
				Lineage: "1",
				Serial:  3,
			},
		},
		{
			StateAgeReceiverNewer, false,
			&State{
				Lineage: "1",
				Serial:  3,
			},
			&State{
				Lineage: "1",
				Serial:  2,
			},
		},
		{
			StateAgeEqual, true,
			&State{
				Lineage: "1",
				Serial:  2,
			},
			&State{
				Lineage: "2",
				Serial:  2,
			},
		},
		{
			StateAgeEqual, true,
			&State{
				Lineage: "1",
				Serial:  3,
			},
			&State{
				Lineage: "2",
				Serial:  2,
			},
		},
	}

	for i, tc := range cases {
		result, err := tc.One.CompareAges(tc.Two)

		if err != nil && !tc.Err {
			t.Errorf(
				"%d: got error, but want success\n\n%s\n\n%s",
				i, tc.One, tc.Two,
			)
			continue
		}

		if err == nil && tc.Err {
			t.Errorf(
				"%d: got success, but want error\n\n%s\n\n%s",
				i, tc.One, tc.Two,
			)
			continue
		}

		if result != tc.Result {
			t.Errorf(
				"%d: got result %d, but want %d\n\n%s\n\n%s",
				i, result, tc.Result, tc.One, tc.Two,
			)
			continue
		}
	}
}

func TestStateSameLineage(t *testing.T) {
	cases := []struct {
		Result   bool
		One, Two *State
	}{
		{
			true,
			&State{
				Lineage: "1",
			},
			&State{
				Lineage: "1",
			},
		},
		{
			// Empty lineage is compatible with all
			true,
			&State{
				Lineage: "",
			},
			&State{
				Lineage: "1",
			},
		},
		{
			// Empty lineage is compatible with all
			true,
			&State{
				Lineage: "1",
			},
			&State{
				Lineage: "",
			},
		},
		{
			false,
			&State{
				Lineage: "1",
			},
			&State{
				Lineage: "2",
			},
		},
	}

	for i, tc := range cases {
		result := tc.One.SameLineage(tc.Two)

		if result != tc.Result {
			t.Errorf(
				"%d: got %v, but want %v\n\n%s\n\n%s",
				i, result, tc.Result, tc.One, tc.Two,
			)
			continue
		}
	}
}

func TestStateMarshalEqual(t *testing.T) {
	tests := map[string]struct {
		S1, S2 *State
		Want   bool
	}{
		"both nil": {
			nil,
			nil,
			true,
		},
		"first zero, second nil": {
			&State{},
			nil,
			false,
		},
		"first nil, second zero": {
			nil,
			&State{},
			false,
		},
		"both zero": {
			// These are not equal because they both implicitly init with
			// different lineage.
			&State{},
			&State{},
			false,
		},
		"both set, same lineage": {
			&State{
				Lineage: "abc123",
			},
			&State{
				Lineage: "abc123",
			},
			true,
		},
		"both set, same lineage, different serial": {
			&State{
				Lineage: "abc123",
				Serial:  1,
			},
			&State{
				Lineage: "abc123",
				Serial:  2,
			},
			false,
		},
		"both set, same lineage, same serial, same resources": {
			&State{
				Lineage: "abc123",
				Serial:  1,
				Modules: []*ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"foo_bar.baz": {},
						},
					},
				},
			},
			&State{
				Lineage: "abc123",
				Serial:  1,
				Modules: []*ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"foo_bar.baz": {},
						},
					},
				},
			},
			true,
		},
		"both set, same lineage, same serial, different resources": {
			&State{
				Lineage: "abc123",
				Serial:  1,
				Modules: []*ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"foo_bar.baz": {},
						},
					},
				},
			},
			&State{
				Lineage: "abc123",
				Serial:  1,
				Modules: []*ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"pizza_crust.tasty": {},
						},
					},
				},
			},
			false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.S1.MarshalEqual(test.S2)
			if got != test.Want {
				t.Errorf("wrong result %#v; want %#v", got, test.Want)
				s1Buf := &bytes.Buffer{}
				s2Buf := &bytes.Buffer{}
				_ = WriteState(test.S1, s1Buf)
				_ = WriteState(test.S2, s2Buf)
				t.Logf("\nState 1: %s\nState 2: %s", s1Buf.Bytes(), s2Buf.Bytes())
			}
		})
	}
}

func TestResourceStateEqual(t *testing.T) {
	cases := []struct {
		Result   bool
		One, Two *ResourceState
	}{
		// Different types
		{
			false,
			&ResourceState{Type: "foo"},
			&ResourceState{Type: "bar"},
		},

		// Different dependencies
		{
			false,
			&ResourceState{Dependencies: []string{"foo"}},
			&ResourceState{Dependencies: []string{"bar"}},
		},

		{
			false,
			&ResourceState{Dependencies: []string{"foo", "bar"}},
			&ResourceState{Dependencies: []string{"foo"}},
		},

		{
			true,
			&ResourceState{Dependencies: []string{"bar", "foo"}},
			&ResourceState{Dependencies: []string{"foo", "bar"}},
		},

		// Different primaries
		{
			false,
			&ResourceState{Primary: nil},
			&ResourceState{Primary: &InstanceState{ID: "foo"}},
		},

		{
			true,
			&ResourceState{Primary: &InstanceState{ID: "foo"}},
			&ResourceState{Primary: &InstanceState{ID: "foo"}},
		},

		// Different tainted
		{
			false,
			&ResourceState{
				Primary: &InstanceState{
					ID: "foo",
				},
			},
			&ResourceState{
				Primary: &InstanceState{
					ID:      "foo",
					Tainted: true,
				},
			},
		},

		{
			true,
			&ResourceState{
				Primary: &InstanceState{
					ID:      "foo",
					Tainted: true,
				},
			},
			&ResourceState{
				Primary: &InstanceState{
					ID:      "foo",
					Tainted: true,
				},
			},
		},
	}

	for i, tc := range cases {
		if tc.One.Equal(tc.Two) != tc.Result {
			t.Fatalf("Bad: %d\n\n%s\n\n%s", i, tc.One.String(), tc.Two.String())
		}
		if tc.Two.Equal(tc.One) != tc.Result {
			t.Fatalf("Bad: %d\n\n%s\n\n%s", i, tc.One.String(), tc.Two.String())
		}
	}
}

func TestResourceStateTaint(t *testing.T) {
	cases := map[string]struct {
		Input  *ResourceState
		Output *ResourceState
	}{
		"no primary": {
			&ResourceState{},
			&ResourceState{},
		},

		"primary, not tainted": {
			&ResourceState{
				Primary: &InstanceState{ID: "foo"},
			},
			&ResourceState{
				Primary: &InstanceState{
					ID:      "foo",
					Tainted: true,
				},
			},
		},

		"primary, tainted": {
			&ResourceState{
				Primary: &InstanceState{
					ID:      "foo",
					Tainted: true,
				},
			},
			&ResourceState{
				Primary: &InstanceState{
					ID:      "foo",
					Tainted: true,
				},
			},
		},
	}

	for k, tc := range cases {
		tc.Input.Taint()
		if !reflect.DeepEqual(tc.Input, tc.Output) {
			t.Fatalf(
				"Failure: %s\n\nExpected: %#v\n\nGot: %#v",
				k, tc.Output, tc.Input)
		}
	}
}

func TestResourceStateUntaint(t *testing.T) {
	cases := map[string]struct {
		Input          *ResourceState
		ExpectedOutput *ResourceState
	}{
		"no primary, err": {
			Input:          &ResourceState{},
			ExpectedOutput: &ResourceState{},
		},

		"primary, not tainted": {
			Input: &ResourceState{
				Primary: &InstanceState{ID: "foo"},
			},
			ExpectedOutput: &ResourceState{
				Primary: &InstanceState{ID: "foo"},
			},
		},
		"primary, tainted": {
			Input: &ResourceState{
				Primary: &InstanceState{
					ID:      "foo",
					Tainted: true,
				},
			},
			ExpectedOutput: &ResourceState{
				Primary: &InstanceState{ID: "foo"},
			},
		},
	}

	for k, tc := range cases {
		tc.Input.Untaint()
		if !reflect.DeepEqual(tc.Input, tc.ExpectedOutput) {
			t.Fatalf(
				"Failure: %s\n\nExpected: %#v\n\nGot: %#v",
				k, tc.ExpectedOutput, tc.Input)
		}
	}
}

func TestInstanceStateEmpty(t *testing.T) {
	cases := map[string]struct {
		In     *InstanceState
		Result bool
	}{
		"nil is empty": {
			nil,
			true,
		},
		"non-nil but without ID is empty": {
			&InstanceState{},
			true,
		},
		"with ID is not empty": {
			&InstanceState{
				ID: "i-abc123",
			},
			false,
		},
	}

	for tn, tc := range cases {
		if tc.In.Empty() != tc.Result {
			t.Fatalf("%q expected %#v to be empty: %#v", tn, tc.In, tc.Result)
		}
	}
}

func TestInstanceStateEqual(t *testing.T) {
	cases := []struct {
		Result   bool
		One, Two *InstanceState
	}{
		// Nils
		{
			false,
			nil,
			&InstanceState{},
		},

		{
			false,
			&InstanceState{},
			nil,
		},

		// Different IDs
		{
			false,
			&InstanceState{ID: "foo"},
			&InstanceState{ID: "bar"},
		},

		// Different Attributes
		{
			false,
			&InstanceState{Attributes: map[string]string{"foo": "bar"}},
			&InstanceState{Attributes: map[string]string{"foo": "baz"}},
		},

		// Different Attribute keys
		{
			false,
			&InstanceState{Attributes: map[string]string{"foo": "bar"}},
			&InstanceState{Attributes: map[string]string{"bar": "baz"}},
		},

		{
			false,
			&InstanceState{Attributes: map[string]string{"bar": "baz"}},
			&InstanceState{Attributes: map[string]string{"foo": "bar"}},
		},
	}

	for i, tc := range cases {
		if tc.One.Equal(tc.Two) != tc.Result {
			t.Fatalf("Bad: %d\n\n%s\n\n%s", i, tc.One.String(), tc.Two.String())
		}
	}
}

func TestStateEmpty(t *testing.T) {
	cases := []struct {
		In     *State
		Result bool
	}{
		{
			nil,
			true,
		},
		{
			&State{},
			true,
		},
		{
			&State{
				Remote: &RemoteState{Type: "foo"},
			},
			true,
		},
		{
			&State{
				Modules: []*ModuleState{
					&ModuleState{},
				},
			},
			false,
		},
	}

	for i, tc := range cases {
		if tc.In.Empty() != tc.Result {
			t.Fatalf("bad %d %#v:\n\n%#v", i, tc.Result, tc.In)
		}
	}
}

func TestStateHasResources(t *testing.T) {
	cases := []struct {
		In     *State
		Result bool
	}{
		{
			nil,
			false,
		},
		{
			&State{},
			false,
		},
		{
			&State{
				Remote: &RemoteState{Type: "foo"},
			},
			false,
		},
		{
			&State{
				Modules: []*ModuleState{
					&ModuleState{},
				},
			},
			false,
		},
		{
			&State{
				Modules: []*ModuleState{
					&ModuleState{},
					&ModuleState{},
				},
			},
			false,
		},
		{
			&State{
				Modules: []*ModuleState{
					&ModuleState{},
					&ModuleState{
						Resources: map[string]*ResourceState{
							"foo.foo": &ResourceState{},
						},
					},
				},
			},
			true,
		},
	}

	for i, tc := range cases {
		if tc.In.HasResources() != tc.Result {
			t.Fatalf("bad %d %#v:\n\n%#v", i, tc.Result, tc.In)
		}
	}
}

func TestStateFromFutureTerraform(t *testing.T) {
	cases := []struct {
		In     string
		Result bool
	}{
		{
			"",
			false,
		},
		{
			"0.1",
			false,
		},
		{
			"999.15.1",
			true,
		},
	}

	for _, tc := range cases {
		state := &State{TFVersion: tc.In}
		actual := state.FromFutureTerraform()
		if actual != tc.Result {
			t.Fatalf("%s: bad: %v", tc.In, actual)
		}
	}
}

func TestStateIsRemote(t *testing.T) {
	cases := []struct {
		In     *State
		Result bool
	}{
		{
			nil,
			false,
		},
		{
			&State{},
			false,
		},
		{
			&State{
				Remote: &RemoteState{Type: "foo"},
			},
			true,
		},
	}

	for i, tc := range cases {
		if tc.In.IsRemote() != tc.Result {
			t.Fatalf("bad %d %#v:\n\n%#v", i, tc.Result, tc.In)
		}
	}
}

func TestInstanceState_MergeDiff(t *testing.T) {
	is := InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo":  "bar",
			"port": "8000",
		},
	}

	diff := &InstanceDiff{
		Attributes: map[string]*ResourceAttrDiff{
			"foo": &ResourceAttrDiff{
				Old: "bar",
				New: "baz",
			},
			"bar": &ResourceAttrDiff{
				Old: "",
				New: "foo",
			},
			"baz": &ResourceAttrDiff{
				Old:         "",
				New:         "foo",
				NewComputed: true,
			},
			"port": &ResourceAttrDiff{
				NewRemoved: true,
			},
		},
	}

	is2 := is.MergeDiff(diff)

	expected := map[string]string{
		"foo": "baz",
		"bar": "foo",
		"baz": config.UnknownVariableValue,
	}

	if !reflect.DeepEqual(expected, is2.Attributes) {
		t.Fatalf("bad: %#v", is2.Attributes)
	}
}

// GH-12183. This tests that a list with a computed set generates the
// right partial state. This never failed but is put here for completion
// of the test case for GH-12183.
func TestInstanceState_MergeDiff_computedSet(t *testing.T) {
	is := InstanceState{}

	diff := &InstanceDiff{
		Attributes: map[string]*ResourceAttrDiff{
			"config.#": &ResourceAttrDiff{
				Old:         "0",
				New:         "1",
				RequiresNew: true,
			},

			"config.0.name": &ResourceAttrDiff{
				Old: "",
				New: "hello",
			},

			"config.0.rules.#": &ResourceAttrDiff{
				Old:         "",
				NewComputed: true,
			},
		},
	}

	is2 := is.MergeDiff(diff)

	expected := map[string]string{
		"config.#":         "1",
		"config.0.name":    "hello",
		"config.0.rules.#": config.UnknownVariableValue,
	}

	if !reflect.DeepEqual(expected, is2.Attributes) {
		t.Fatalf("bad: %#v", is2.Attributes)
	}
}

func TestInstanceState_MergeDiff_nil(t *testing.T) {
	var is *InstanceState

	diff := &InstanceDiff{
		Attributes: map[string]*ResourceAttrDiff{
			"foo": &ResourceAttrDiff{
				Old: "",
				New: "baz",
			},
		},
	}

	is2 := is.MergeDiff(diff)

	expected := map[string]string{
		"foo": "baz",
	}

	if !reflect.DeepEqual(expected, is2.Attributes) {
		t.Fatalf("bad: %#v", is2.Attributes)
	}
}

func TestInstanceState_MergeDiff_nilDiff(t *testing.T) {
	is := InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo": "bar",
		},
	}

	is2 := is.MergeDiff(nil)

	expected := map[string]string{
		"foo": "bar",
	}

	if !reflect.DeepEqual(expected, is2.Attributes) {
		t.Fatalf("bad: %#v", is2.Attributes)
	}
}

func TestReadWriteState(t *testing.T) {
	state := &State{
		Serial:  9,
		Lineage: "5d1ad1a1-4027-4665-a908-dbe6adff11d8",
		Remote: &RemoteState{
			Type: "http",
			Config: map[string]string{
				"url": "http://my-cool-server.com/",
			},
		},
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Dependencies: []string{
					"aws_instance.bar",
				},
				Resources: map[string]*ResourceState{
					"foo": &ResourceState{
						Primary: &InstanceState{
							ID: "bar",
							Ephemeral: EphemeralState{
								ConnInfo: map[string]string{
									"type":     "ssh",
									"user":     "root",
									"password": "supersecret",
								},
							},
						},
					},
				},
			},
		},
	}
	state.init()

	buf := new(bytes.Buffer)
	if err := WriteState(state, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify that the version and serial are set
	if state.Version != StateVersion {
		t.Fatalf("bad version number: %d", state.Version)
	}

	actual, err := ReadState(buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// ReadState should not restore sensitive information!
	mod := state.RootModule()
	mod.Resources["foo"].Primary.Ephemeral = EphemeralState{}
	mod.Resources["foo"].Primary.Ephemeral.init()

	if !reflect.DeepEqual(actual, state) {
		t.Logf("expected:\n%#v", state)
		t.Fatalf("got:\n%#v", actual)
	}
}

func TestReadStateNewVersion(t *testing.T) {
	type out struct {
		Version int
	}

	buf, err := json.Marshal(&out{StateVersion + 1})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	s, err := ReadState(bytes.NewReader(buf))
	if s != nil {
		t.Fatalf("unexpected: %#v", s)
	}
	if !strings.Contains(err.Error(), "does not support state version") {
		t.Fatalf("err: %v", err)
	}
}

func TestReadStateEmptyOrNilFile(t *testing.T) {
	var emptyState bytes.Buffer
	_, err := ReadState(&emptyState)
	if err != ErrNoState {
		t.Fatal("expected ErrNostate, got", err)
	}

	var nilFile *os.File
	_, err = ReadState(nilFile)
	if err != ErrNoState {
		t.Fatal("expected ErrNostate, got", err)
	}
}

func TestReadStateTFVersion(t *testing.T) {
	type tfVersion struct {
		Version   int    `json:"version"`
		TFVersion string `json:"terraform_version"`
	}

	cases := []struct {
		Written string
		Read    string
		Err     bool
	}{
		{
			"0.0.0",
			"0.0.0",
			false,
		},
		{
			"",
			"",
			false,
		},
		{
			"bad",
			"",
			true,
		},
	}

	for _, tc := range cases {
		buf, err := json.Marshal(&tfVersion{
			Version:   2,
			TFVersion: tc.Written,
		})
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		s, err := ReadState(bytes.NewReader(buf))
		if (err != nil) != tc.Err {
			t.Fatalf("%s: err: %s", tc.Written, err)
		}
		if err != nil {
			continue
		}

		if s.TFVersion != tc.Read {
			t.Fatalf("%s: bad: %s", tc.Written, s.TFVersion)
		}
	}
}

func TestWriteStateTFVersion(t *testing.T) {
	cases := []struct {
		Write string
		Read  string
		Err   bool
	}{
		{
			"0.0.0",
			"0.0.0",
			false,
		},
		{
			"",
			"",
			false,
		},
		{
			"bad",
			"",
			true,
		},
	}

	for _, tc := range cases {
		var buf bytes.Buffer
		err := WriteState(&State{TFVersion: tc.Write}, &buf)
		if (err != nil) != tc.Err {
			t.Fatalf("%s: err: %s", tc.Write, err)
		}
		if err != nil {
			continue
		}

		s, err := ReadState(&buf)
		if err != nil {
			t.Fatalf("%s: err: %s", tc.Write, err)
		}

		if s.TFVersion != tc.Read {
			t.Fatalf("%s: bad: %s", tc.Write, s.TFVersion)
		}
	}
}

func TestParseResourceStateKey(t *testing.T) {
	cases := []struct {
		Input       string
		Expected    *ResourceStateKey
		ExpectedErr bool
	}{
		{
			Input: "aws_instance.foo.3",
			Expected: &ResourceStateKey{
				Mode:  config.ManagedResourceMode,
				Type:  "aws_instance",
				Name:  "foo",
				Index: 3,
			},
		},
		{
			Input: "aws_instance.foo.0",
			Expected: &ResourceStateKey{
				Mode:  config.ManagedResourceMode,
				Type:  "aws_instance",
				Name:  "foo",
				Index: 0,
			},
		},
		{
			Input: "aws_instance.foo",
			Expected: &ResourceStateKey{
				Mode:  config.ManagedResourceMode,
				Type:  "aws_instance",
				Name:  "foo",
				Index: -1,
			},
		},
		{
			Input: "data.aws_ami.foo",
			Expected: &ResourceStateKey{
				Mode:  config.DataResourceMode,
				Type:  "aws_ami",
				Name:  "foo",
				Index: -1,
			},
		},
		{
			Input:       "aws_instance.foo.malformed",
			ExpectedErr: true,
		},
		{
			Input:       "aws_instance.foo.malformedwithnumber.123",
			ExpectedErr: true,
		},
		{
			Input:       "malformed",
			ExpectedErr: true,
		},
	}
	for _, tc := range cases {
		rsk, err := ParseResourceStateKey(tc.Input)
		if rsk != nil && tc.Expected != nil && !rsk.Equal(tc.Expected) {
			t.Fatalf("%s: expected %s, got %s", tc.Input, tc.Expected, rsk)
		}
		if (err != nil) != tc.ExpectedErr {
			t.Fatalf("%s: expected err: %t, got %s", tc.Input, tc.ExpectedErr, err)
		}
	}
}

func TestReadState_prune(t *testing.T) {
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{Path: rootModulePath},
			nil,
		},
	}
	state.init()

	buf := new(bytes.Buffer)
	if err := WriteState(state, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual, err := ReadState(buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &State{
		Version: state.Version,
		Lineage: state.Lineage,
	}
	expected.init()

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("got:\n%#v", actual)
	}
}

func TestReadState_pruneDependencies(t *testing.T) {
	state := &State{
		Serial:  9,
		Lineage: "5d1ad1a1-4027-4665-a908-dbe6adff11d8",
		Remote: &RemoteState{
			Type: "http",
			Config: map[string]string{
				"url": "http://my-cool-server.com/",
			},
		},
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Dependencies: []string{
					"aws_instance.bar",
					"aws_instance.bar",
				},
				Resources: map[string]*ResourceState{
					"foo": &ResourceState{
						Dependencies: []string{
							"aws_instance.baz",
							"aws_instance.baz",
						},
						Primary: &InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}
	state.init()

	buf := new(bytes.Buffer)
	if err := WriteState(state, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual, err := ReadState(buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// make sure the duplicate Dependencies are filtered
	modDeps := actual.Modules[0].Dependencies
	resourceDeps := actual.Modules[0].Resources["foo"].Dependencies

	if len(modDeps) > 1 || modDeps[0] != "aws_instance.bar" {
		t.Fatalf("expected 1 module depends_on entry, got %q", modDeps)
	}

	if len(resourceDeps) > 1 || resourceDeps[0] != "aws_instance.baz" {
		t.Fatalf("expected 1 resource depends_on entry, got  %q", resourceDeps)
	}
}

func TestResourceNameSort(t *testing.T) {
	names := []string{
		"a",
		"b",
		"a.0",
		"a.c",
		"a.d",
		"c",
		"a.b.0",
		"a.b.1",
		"a.b.10",
		"a.b.2",
	}

	sort.Sort(resourceNameSort(names))

	expected := []string{
		"a",
		"a.0",
		"a.b.0",
		"a.b.1",
		"a.b.2",
		"a.b.10",
		"a.c",
		"a.d",
		"b",
		"c",
	}

	if !reflect.DeepEqual(names, expected) {
		t.Fatalf("got: %q\nexpected: %q\n", names, expected)
	}
}
