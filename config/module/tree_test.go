package module

import (
	"reflect"
	"testing"
)

func TestTree_Load(t *testing.T) {
	tree := NewTree(testConfig(t, "basic"))
	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestTree_Modules(t *testing.T) {
	tree := NewTree(testConfig(t, "basic"))
	actual := tree.Modules()

	expected := []*Module{
		&Module{Name: "foo", Source: "./foo"},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}
