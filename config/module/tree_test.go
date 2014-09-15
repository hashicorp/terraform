package module

import (
	"reflect"
	"strings"
	"testing"
)

func TestTree_Load(t *testing.T) {
	storage := testStorage(t)
	tree := NewTree(testConfig(t, "basic"))

	// This should error because we haven't gotten things yet
	if err := tree.Load(storage, GetModeNone); err == nil {
		t.Fatal("should error")
	}

	// This should get things
	if err := tree.Load(storage, GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	// This should no longer error
	if err := tree.Load(storage, GetModeNone); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(tree.String())
	expected := strings.TrimSpace(treeLoadStr)
	if actual != expected {
		t.Fatalf("bad: \n\n%s", actual)
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

func TestTree_Name(t *testing.T) {
	tree := NewTree(testConfig(t, "basic"))
	actual := tree.Name()

	if actual != "<root>" {
		t.Fatalf("bad: %#v", actual)
	}
}

const treeLoadStr = `
<root>
  foo
`
