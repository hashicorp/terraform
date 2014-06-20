package diff

import (
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceBuilder_new(t *testing.T) {
	rb := &ResourceBuilder{
		CreateComputedAttrs: []string{"private_ip"},
	}

	state := &terraform.ResourceState{}

	c := testConfig(t, map[string]interface{}{
		"foo": "bar",
	}, nil)

	diff, err := rb.Diff(state, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if diff == nil {
		t.Fatal("should not be nil")
	}

	actual := testResourceDiffStr(diff)
	expected := testRBNewDiff
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestResourceBuilder_requiresNew(t *testing.T) {
	rb := &ResourceBuilder{
		CreateComputedAttrs: []string{"private_ip"},
		RequiresNewAttrs:    []string{"ami"},
	}

	state := &terraform.ResourceState{
		ID: "1",
		Attributes: map[string]string{
			"ami":        "foo",
			"private_ip": "127.0.0.1",
		},
	}

	c := testConfig(t, map[string]interface{}{
		"ami": "bar",
	}, nil)

	diff, err := rb.Diff(state, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if diff == nil {
		t.Fatal("should not be nil")
	}

	actual := testResourceDiffStr(diff)
	expected := testRBRequiresNewDiff
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestResourceBuilder_same(t *testing.T) {
	rb := &ResourceBuilder{
		CreateComputedAttrs: []string{"private_ip"},
	}

	state := &terraform.ResourceState{
		ID: "1",
		Attributes: map[string]string{
			"foo": "bar",
		},
	}

	c := testConfig(t, map[string]interface{}{
		"foo": "bar",
	}, nil)

	diff, err := rb.Diff(state, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if diff != nil {
		t.Fatal("should not diff: %s", diff)
	}
}

func TestResourceBuilder_unknown(t *testing.T) {
	rb := &ResourceBuilder{}

	state := &terraform.ResourceState{}

	c := testConfig(t, map[string]interface{}{
		"foo": "${var.unknown}",
	}, map[string]string{
		"var.foo":     "bar",
		"var.unknown": config.UnknownVariableValue,
	})

	diff, err := rb.Diff(state, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if diff == nil {
		t.Fatal("should not be nil")
	}

	actual := testResourceDiffStr(diff)
	expected := testRBUnknownDiff
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestResourceBuilder_vars(t *testing.T) {
	rb := &ResourceBuilder{}

	state := &terraform.ResourceState{}

	c := testConfig(t, map[string]interface{}{
		"foo": "${var.foo}",
	}, map[string]string{
		"var.foo": "bar",
	})

	diff, err := rb.Diff(state, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if diff == nil {
		t.Fatal("should not be nil")
	}

	actual := testResourceDiffStr(diff)
	expected := testRBVarsDiff
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

const testRBNewDiff = `CREATE
  foo:        "" => "bar"
  id:         "" => "<computed>" (forces new resource)
  private_ip: "" => "<computed>"
`

const testRBRequiresNewDiff = `CREATE
  ami:        "foo" => "bar" (forces new resource)
  id:         "1" => "<computed>" (forces new resource)
  private_ip: "127.0.0.1" => "<computed>"
`

const testRBUnknownDiff = `CREATE
  foo: "" => "${var.unknown}"
  id:  "" => "<computed>" (forces new resource)
`

const testRBVarsDiff = `CREATE
  foo: "" => "bar"
  id:  "" => "<computed>" (forces new resource)
`
