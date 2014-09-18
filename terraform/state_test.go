package terraform

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
)

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

func TestReadUpgradeState(t *testing.T) {
	state := &StateV1{
		Resources: map[string]*ResourceStateV1{
			"foo": &ResourceStateV1{
				ID: "bar",
			},
		},
	}
	buf := new(bytes.Buffer)
	if err := testWriteStateV1(state, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	// ReadState should transparently detect the old
	// version and upgrade up so the latest.
	actual, err := ReadState(buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	upgraded, err := upgradeV1State(state)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(actual, upgraded) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestReadWriteState(t *testing.T) {
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
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

	// Checksum before the write
	chksum := checksumStruct(t, state)

	buf := new(bytes.Buffer)
	if err := WriteState(state, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Checksum after the write
	chksumAfter := checksumStruct(t, state)
	if chksumAfter != chksum {
		t.Fatalf("structure changed during serialization!")
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

func TestUpgradeV1State(t *testing.T) {
	old := &StateV1{
		Outputs: map[string]string{
			"ip": "127.0.0.1",
		},
		Resources: map[string]*ResourceStateV1{
			"foo": &ResourceStateV1{
				Type: "test_resource",
				ID:   "bar",
				Attributes: map[string]string{
					"key": "val",
				},
			},
			"bar": &ResourceStateV1{
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
	state, err := upgradeV1State(old)
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
