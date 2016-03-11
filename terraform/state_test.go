package terraform

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config"
)

func TestStateAddModule(t *testing.T) {
	cases := []struct {
		In  [][]string
		Out [][]string
	}{
		{
			[][]string{
				[]string{"root"},
				[]string{"root", "child"},
			},
			[][]string{
				[]string{"root"},
				[]string{"root", "child"},
			},
		},

		{
			[][]string{
				[]string{"root", "foo", "bar"},
				[]string{"root", "foo"},
				[]string{"root"},
				[]string{"root", "bar"},
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
			[][]string{
				[]string{"root", "foo", "bar"}, // This one should sort after...
				[]string{"root", "foo"},
				[]string{"root"},
				[]string{"root", "bar", "bar"}, // ...this one.
				[]string{"root", "bar"},
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
			t.Fatalf("In: %#v\n\nOut: %#v", tc.In, actual)
		}
	}
}

func TestStateOutputTypeRoundTrip(t *testing.T) {
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
				Outputs: map[string]interface{}{
					"string_output": "String Value",
					"list_output":   []interface{}{"List", "Value"},
					"map_output": map[string]interface{}{
						"key1": "Map",
						"key2": "Value",
					},
				},
			},
		},
	}

	buf := new(bytes.Buffer)
	if err := WriteState(state, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	roundTripped, err := ReadState(buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(state, roundTripped) {
		t.Fatalf("bad: %#v", roundTripped)
	}
}

func TestStateModuleOrphans(t *testing.T) {
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
			},
			&ModuleState{
				Path: []string{RootModuleName, "foo"},
			},
			&ModuleState{
				Path: []string{RootModuleName, "bar"},
			},
		},
	}

	config := testModule(t, "state-module-orphans").Config()
	actual := state.ModuleOrphans(RootModulePath, config)
	expected := [][]string{
		[]string{RootModuleName, "foo"},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestStateModuleOrphans_nested(t *testing.T) {
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
			},
			&ModuleState{
				Path: []string{RootModuleName, "foo", "bar"},
			},
		},
	}

	actual := state.ModuleOrphans(RootModulePath, nil)
	expected := [][]string{
		[]string{RootModuleName, "foo"},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestStateModuleOrphans_nilConfig(t *testing.T) {
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
			},
			&ModuleState{
				Path: []string{RootModuleName, "foo"},
			},
			&ModuleState{
				Path: []string{RootModuleName, "bar"},
			},
		},
	}

	actual := state.ModuleOrphans(RootModulePath, nil)
	expected := [][]string{
		[]string{RootModuleName, "foo"},
		[]string{RootModuleName, "bar"},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestStateModuleOrphans_deepNestedNilConfig(t *testing.T) {
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
			},
			&ModuleState{
				Path: []string{RootModuleName, "parent", "childfoo"},
			},
			&ModuleState{
				Path: []string{RootModuleName, "parent", "childbar"},
			},
		},
	}

	actual := state.ModuleOrphans(RootModulePath, nil)
	expected := [][]string{
		[]string{RootModuleName, "parent"},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestStateEqual(t *testing.T) {
	cases := []struct {
		Result   bool
		One, Two *State
	}{
		// Nils
		{
			false,
			nil,
			&State{Version: 2},
		},

		{
			true,
			nil,
			nil,
		},

		// Different versions
		{
			false,
			&State{Version: 5},
			&State{Version: 2},
		},

		// Different modules
		{
			false,
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: RootModulePath,
					},
				},
			},
			&State{},
		},

		{
			true,
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: RootModulePath,
					},
				},
			},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: RootModulePath,
					},
				},
			},
		},

		// Meta differs
		{
			false,
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: rootModulePath,
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Primary: &InstanceState{
									Meta: map[string]string{
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
									Meta: map[string]string{
										"schema_version": "2",
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
		if tc.One.Equal(tc.Two) != tc.Result {
			t.Fatalf("Bad: %d\n\n%s\n\n%s", i, tc.One.String(), tc.Two.String())
		}
		if tc.Two.Equal(tc.One) != tc.Result {
			t.Fatalf("Bad: %d\n\n%s\n\n%s", i, tc.One.String(), tc.Two.String())
		}
	}
}

func TestStateIncrementSerialMaybe(t *testing.T) {
	cases := map[string]struct {
		S1, S2 *State
		Serial int64
	}{
		"S2 is nil": {
			&State{},
			nil,
			0,
		},
		"S2 is identical": {
			&State{},
			&State{},
			0,
		},
		"S2 is different": {
			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{Path: rootModulePath},
				},
			},
			1,
		},
		"S2 is different, but only via Instance Metadata": {
			&State{
				Serial: 3,
				Modules: []*ModuleState{
					&ModuleState{
						Path: rootModulePath,
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Primary: &InstanceState{
									Meta: map[string]string{},
								},
							},
						},
					},
				},
			},
			&State{
				Serial: 3,
				Modules: []*ModuleState{
					&ModuleState{
						Path: rootModulePath,
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Primary: &InstanceState{
									Meta: map[string]string{
										"schema_version": "1",
									},
								},
							},
						},
					},
				},
			},
			4,
		},
		"S1 serial is higher": {
			&State{Serial: 5},
			&State{
				Serial: 3,
				Modules: []*ModuleState{
					&ModuleState{Path: rootModulePath},
				},
			},
			5,
		},
	}

	for name, tc := range cases {
		tc.S1.IncrementSerialMaybe(tc.S2)
		if tc.S1.Serial != tc.Serial {
			t.Fatalf("Bad: %s\nGot: %d", name, tc.S1.Serial)
		}
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
				Tainted: nil,
			},
			&ResourceState{
				Tainted: []*InstanceState{
					&InstanceState{ID: "foo"},
				},
			},
		},

		{
			true,
			&ResourceState{
				Tainted: []*InstanceState{
					&InstanceState{ID: "foo"},
				},
			},
			&ResourceState{
				Tainted: []*InstanceState{
					&InstanceState{ID: "foo"},
				},
			},
		},

		{
			true,
			&ResourceState{
				Tainted: []*InstanceState{
					&InstanceState{ID: "foo"},
					nil,
				},
			},
			&ResourceState{
				Tainted: []*InstanceState{
					&InstanceState{ID: "foo"},
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

		"primary, no tainted": {
			&ResourceState{
				Primary: &InstanceState{ID: "foo"},
			},
			&ResourceState{
				Tainted: []*InstanceState{
					&InstanceState{ID: "foo"},
				},
			},
		},

		"primary, with tainted": {
			&ResourceState{
				Primary: &InstanceState{ID: "foo"},
				Tainted: []*InstanceState{
					&InstanceState{ID: "bar"},
				},
			},
			&ResourceState{
				Tainted: []*InstanceState{
					&InstanceState{ID: "bar"},
					&InstanceState{ID: "foo"},
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
		Index          func() int
		ExpectedOutput *ResourceState
		ExpectedErrMsg string
	}{
		"no primary, no tainted, err": {
			Input:          &ResourceState{},
			ExpectedOutput: &ResourceState{},
			ExpectedErrMsg: "Nothing to untaint",
		},

		"one tainted, no primary": {
			Input: &ResourceState{
				Tainted: []*InstanceState{
					&InstanceState{ID: "foo"},
				},
			},
			ExpectedOutput: &ResourceState{
				Primary: &InstanceState{ID: "foo"},
				Tainted: []*InstanceState{},
			},
		},

		"one tainted, existing primary error": {
			Input: &ResourceState{
				Primary: &InstanceState{ID: "foo"},
				Tainted: []*InstanceState{
					&InstanceState{ID: "foo"},
				},
			},
			ExpectedErrMsg: "Resource has a primary",
		},

		"multiple tainted, no index": {
			Input: &ResourceState{
				Tainted: []*InstanceState{
					&InstanceState{ID: "bar"},
					&InstanceState{ID: "foo"},
				},
			},
			ExpectedErrMsg: "please specify an index",
		},

		"multiple tainted, with index": {
			Input: &ResourceState{
				Tainted: []*InstanceState{
					&InstanceState{ID: "bar"},
					&InstanceState{ID: "foo"},
				},
			},
			Index: func() int { return 1 },
			ExpectedOutput: &ResourceState{
				Primary: &InstanceState{ID: "foo"},
				Tainted: []*InstanceState{
					&InstanceState{ID: "bar"},
				},
			},
		},

		"index out of bounds error": {
			Input: &ResourceState{
				Tainted: []*InstanceState{
					&InstanceState{ID: "bar"},
					&InstanceState{ID: "foo"},
				},
			},
			Index:          func() int { return 2 },
			ExpectedErrMsg: "out of range",
		},
	}

	for k, tc := range cases {
		index := -1
		if tc.Index != nil {
			index = tc.Index()
		}
		err := tc.Input.Untaint(index)
		if tc.ExpectedErrMsg == "" && err != nil {
			t.Fatalf("[%s] unexpected err: %s", k, err)
		}
		if tc.ExpectedErrMsg != "" {
			if strings.Contains(err.Error(), tc.ExpectedErrMsg) {
				continue
			}
			t.Fatalf("[%s] expected err: %s to contain: %s",
				k, err, tc.ExpectedErrMsg)
		}
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

func TestInstanceState_MergeDiff_nil(t *testing.T) {
	var is *InstanceState = nil

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

func TestReadUpgradeStateV1toV2(t *testing.T) {
	// ReadState should transparently detect the old version but will upgrade
	// it on Write.
	actual, err := ReadState(strings.NewReader(testV1State))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	buf := new(bytes.Buffer)
	if err := WriteState(actual, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	if actual.Version != 2 {
		t.Fatalf("bad: State version not incremented; is %d", actual.Version)
	}

	roundTripped, err := ReadState(buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(actual, roundTripped) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestReadUpgradeState(t *testing.T) {
	state := &StateV0{
		Resources: map[string]*ResourceStateV0{
			"foo": &ResourceStateV0{
				ID: "bar",
			},
		},
	}
	buf := new(bytes.Buffer)
	if err := testWriteStateV0(state, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	// ReadState should transparently detect the old
	// version and upgrade up so the latest.
	actual, err := ReadState(buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	upgraded, err := upgradeV0State(state)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(actual, upgraded) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestReadWriteState(t *testing.T) {
	state := &State{
		Serial: 9,
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

	if !reflect.DeepEqual(actual, state) {
		t.Fatalf("bad: %#v", actual)
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
	if !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("err: %v", err)
	}
}

func TestReadStateTFVersion(t *testing.T) {
	type tfVersion struct {
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
			"0.0.0",
			false,
		},
		{
			"bad",
			"",
			true,
		},
	}

	for _, tc := range cases {
		buf, err := json.Marshal(&tfVersion{tc.Written})
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

func TestUpgradeV0State(t *testing.T) {
	old := &StateV0{
		Outputs: map[string]string{
			"ip": "127.0.0.1",
		},
		Resources: map[string]*ResourceStateV0{
			"foo": &ResourceStateV0{
				Type: "test_resource",
				ID:   "bar",
				Attributes: map[string]string{
					"key": "val",
				},
			},
			"bar": &ResourceStateV0{
				Type: "test_resource",
				ID:   "1234",
				Attributes: map[string]string{
					"a": "b",
				},
			},
		},
		Tainted: map[string]struct{}{
			"bar": struct{}{},
		},
	}
	state, err := upgradeV0State(old)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(state.Modules) != 1 {
		t.Fatalf("should only have root module: %#v", state.Modules)
	}
	root := state.RootModule()

	if len(root.Outputs) != 1 {
		t.Fatalf("bad outputs: %v", root.Outputs)
	}
	if root.Outputs["ip"] != "127.0.0.1" {
		t.Fatalf("bad outputs: %v", root.Outputs)
	}

	if len(root.Resources) != 2 {
		t.Fatalf("bad resources: %v", root.Resources)
	}

	foo := root.Resources["foo"]
	if foo.Type != "test_resource" {
		t.Fatalf("bad: %#v", foo)
	}
	if foo.Primary == nil || foo.Primary.ID != "bar" ||
		foo.Primary.Attributes["key"] != "val" {
		t.Fatalf("bad: %#v", foo)
	}
	if len(foo.Tainted) > 0 {
		t.Fatalf("bad: %#v", foo)
	}

	bar := root.Resources["bar"]
	if bar.Type != "test_resource" {
		t.Fatalf("bad: %#v", bar)
	}
	if bar.Primary != nil {
		t.Fatalf("bad: %#v", bar)
	}
	if len(bar.Tainted) != 1 {
		t.Fatalf("bad: %#v", bar)
	}
	bt := bar.Tainted[0]
	if bt.ID != "1234" || bt.Attributes["a"] != "b" {
		t.Fatalf("bad: %#v", bt)
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
				Type:  "aws_instance",
				Name:  "foo",
				Index: 3,
			},
		},
		{
			Input: "aws_instance.foo.0",
			Expected: &ResourceStateKey{
				Type:  "aws_instance",
				Name:  "foo",
				Index: 0,
			},
		},
		{
			Input: "aws_instance.foo",
			Expected: &ResourceStateKey{
				Type:  "aws_instance",
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

const testV1State = `{
    "version": 1,
    "serial": 9,
    "remote": {
        "type": "http",
        "config": {
            "url": "http://my-cool-server.com/"
        }
    },
    "modules": [
        {
            "path": [
                "root"
            ],
            "outputs": null,
            "resources": {
                "foo": {
                    "type": "",
                    "primary": {
                        "id": "bar"
                    }
                }
            },
            "depends_on": [
                "aws_instance.bar"
            ]
        }
    ]
}
`
