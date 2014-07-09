package diff

import (
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceBuilder_complex(t *testing.T) {
	rb := &ResourceBuilder{
		Attrs: map[string]AttrType{
			"listener": AttrTypeUpdate,
		},
	}

	state := &terraform.ResourceState{
		ID: "foo",
		Attributes: map[string]string{
			"ignore":          "1",
			"listener.#":      "1",
			"listener.0.port": "80",
		},
	}

	c := testConfig(t, map[string]interface{}{
		"listener": []interface{}{
			map[interface{}]interface{}{
				"port": 3000,
			},
		},
	}, nil)

	diff, err := rb.Diff(state, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if diff == nil {
		t.Fatal("should not be nil")
	}

	actual := testResourceDiffStr(diff)
	expected := testRBComplexDiff
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestResourceBuilder_complexReplace(t *testing.T) {
	rb := &ResourceBuilder{
		Attrs: map[string]AttrType{
			"listener": AttrTypeUpdate,
		},
	}

	state := &terraform.ResourceState{
		ID: "foo",
		Attributes: map[string]string{
			"ignore":          "1",
			"listener.#":      "1",
			"listener.0.port": "80",
		},
	}

	c := testConfig(t, map[string]interface{}{
		"listener": []interface{}{
			map[interface{}]interface{}{
				"value": "50",
			},
		},
	}, nil)

	diff, err := rb.Diff(state, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if diff == nil {
		t.Fatal("should not be nil")
	}

	actual := testResourceDiffStr(diff)
	expected := testRBComplexReplaceDiff
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestResourceBuilder_new(t *testing.T) {
	rb := &ResourceBuilder{
		Attrs: map[string]AttrType{
			"foo": AttrTypeUpdate,
		},
		ComputedAttrs: []string{"private_ip"},
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
		ComputedAttrs: []string{"private_ip"},
		Attrs: map[string]AttrType{
			"ami": AttrTypeCreate,
		},
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
		ComputedAttrs: []string{"private_ip"},
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
	rb := &ResourceBuilder{
		Attrs: map[string]AttrType{
			"foo": AttrTypeUpdate,
		},
	}

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
	rb := &ResourceBuilder{
		Attrs: map[string]AttrType{
			"foo": AttrTypeUpdate,
		},
	}

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

const testRBComplexDiff = `UPDATE
  IN  listener.0.port: "80" => "3000"
`

const testRBComplexReplaceDiff = `UPDATE
  IN  listener.0.port:  "80" => "<removed>"
  IN  listener.0.value: "" => "50"
`

const testRBNewDiff = `UPDATE
  IN  foo:        "" => "bar"
  OUT private_ip: "" => "<computed>"
`

const testRBRequiresNewDiff = `CREATE
  IN  ami:        "foo" => "bar" (forces new resource)
  OUT private_ip: "127.0.0.1" => "<computed>"
`

const testRBUnknownDiff = `UPDATE
  IN  foo: "" => "${var.unknown}"
`

const testRBVarsDiff = `UPDATE
  IN  foo: "" => "bar"
`
