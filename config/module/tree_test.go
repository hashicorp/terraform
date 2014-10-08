package module

import (
	"reflect"
	"strings"
	"testing"
)

func TestTreeChild(t *testing.T) {
	storage := testStorage(t)
	tree := NewTree("", testConfig(t, "child"))
	if err := tree.Load(storage, GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Should be able to get the root child
	if c := tree.Child([]string{}); c == nil {
		t.Fatal("should not be nil")
	} else if c.Name() != "root" {
		t.Fatalf("bad: %#v", c.Name())
	}

	// Should be able to get the root child
	if c := tree.Child(nil); c == nil {
		t.Fatal("should not be nil")
	} else if c.Name() != "root" {
		t.Fatalf("bad: %#v", c.Name())
	}

	// Should be able to get the foo child
	if c := tree.Child([]string{"foo"}); c == nil {
		t.Fatal("should not be nil")
	} else if c.Name() != "foo" {
		t.Fatalf("bad: %#v", c.Name())
	}

	// Should be able to get the nested child
	if c := tree.Child([]string{"foo", "bar"}); c == nil {
		t.Fatal("should not be nil")
	} else if c.Name() != "bar" {
		t.Fatalf("bad: %#v", c.Name())
	}
}

func TestTreeLoad(t *testing.T) {
	storage := testStorage(t)
	tree := NewTree("", testConfig(t, "basic"))

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should error because we haven't gotten things yet
	if err := tree.Load(storage, GetModeNone); err == nil {
		t.Fatal("should error")
	}

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should get things
	if err := tree.Load(storage, GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !tree.Loaded() {
		t.Fatal("should be loaded")
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

func TestTreeLoad_duplicate(t *testing.T) {
	storage := testStorage(t)
	tree := NewTree("", testConfig(t, "dup"))

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should get things
	if err := tree.Load(storage, GetModeGet); err == nil {
		t.Fatalf("should error")
	}
}

func TestTreeLoad_subdir(t *testing.T) {
	storage := testStorage(t)
	tree := NewTree("", testConfig(t, "basic-subdir"))

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should error because we haven't gotten things yet
	if err := tree.Load(storage, GetModeNone); err == nil {
		t.Fatal("should error")
	}

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should get things
	if err := tree.Load(storage, GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !tree.Loaded() {
		t.Fatal("should be loaded")
	}

	// This should no longer error
	if err := tree.Load(storage, GetModeNone); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(tree.String())
	expected := strings.TrimSpace(treeLoadSubdirStr)
	if actual != expected {
		t.Fatalf("bad: \n\n%s", actual)
	}
}

func TestTreeModules(t *testing.T) {
	tree := NewTree("", testConfig(t, "basic"))
	actual := tree.Modules()

	expected := []*Module{
		&Module{Name: "foo", Source: "./foo"},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestTreeName(t *testing.T) {
	tree := NewTree("", testConfig(t, "basic"))
	actual := tree.Name()

	if actual != RootName {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestTreeValidate_badChild(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-child-bad"))

	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_badChildOutput(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-bad-output"))

	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_badChildVar(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-bad-var"))

	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_badRoot(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-root-bad"))

	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_good(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-child-good"))

	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestTreeValidate_notLoaded(t *testing.T) {
	tree := NewTree("", testConfig(t, "basic"))

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_requiredChildVar(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-required-var"))

	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

const treeLoadStr = `
root
  foo
`

const treeLoadSubdirStr = `
root
  foo
    bar
`
