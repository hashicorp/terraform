package diff

import (
	"testing"

	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceBuilder_attrSetComputed(t *testing.T) {
	rb := &ResourceBuilder{
		Attrs: map[string]AttrType{
			"foo": AttrTypeCreate,
		},
		ComputedAttrs: []string{
			"foo",
		},
	}

	state := &terraform.InstanceState{}
	c := testConfig(t, map[string]interface{}{
		"foo": "bar",
	}, nil)

	diff, err := rb.Diff(state, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if diff == nil {
		t.Fatal("diff shold not be nil")
	}

	actual := testResourceDiffStr(diff)
	expected := testRBAttrSetComputedDiff
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestResourceBuilder_attrSetComputedComplex(t *testing.T) {
	rb := &ResourceBuilder{
		Attrs: map[string]AttrType{
			"foo": AttrTypeCreate,
		},
		ComputedAttrs: []string{
			"foo",
		},
	}

	state := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo.#": "0",
		},
	}

	c := testConfig(t, map[string]interface{}{}, nil)

	diff, err := rb.Diff(state, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if diff != nil {
		t.Fatalf("diff shold be nil: %#v", diff)
	}
}

func TestResourceBuilder_replaceComputed(t *testing.T) {
	rb := &ResourceBuilder{
		Attrs: map[string]AttrType{
			"foo": AttrTypeCreate,
		},
		ComputedAttrs: []string{
			"foo",
		},
	}

	state := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo": "bar",
		},
	}
	c := testConfig(t, nil, nil)

	diff, err := rb.Diff(state, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if diff != nil {
		t.Fatalf("should be nil: %#v", diff)
	}
}

func TestResourceBuilder_complex(t *testing.T) {
	rb := &ResourceBuilder{
		Attrs: map[string]AttrType{
			"listener": AttrTypeUpdate,
		},
	}

	state := &terraform.InstanceState{
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

	state := &terraform.InstanceState{
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

func TestResourceBuilder_computedAttrsUpdate(t *testing.T) {
	rb := &ResourceBuilder{
		Attrs: map[string]AttrType{
			"foo": AttrTypeUpdate,
		},
		ComputedAttrsUpdate: []string{
			"bar",
		},
	}

	state := &terraform.InstanceState{
		Attributes: map[string]string{"foo": "foo"},
	}
	c := testConfig(t, map[string]interface{}{
		"foo": "bar",
	}, nil)

	diff, err := rb.Diff(state, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if diff == nil {
		t.Fatal("diff shold not be nil")
	}

	actual := testResourceDiffStr(diff)
	expected := testRBComputedAttrUpdate
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

	state := &terraform.InstanceState{}

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

func TestResourceBuilder_preProcess(t *testing.T) {
	rb := &ResourceBuilder{
		Attrs: map[string]AttrType{
			"foo": AttrTypeCreate,
		},

		PreProcess: map[string]PreProcessFunc{
			"foo": func(v string) string {
				return "bar" + v
			},
		},
	}

	state := &terraform.InstanceState{}
	c := testConfig(t, map[string]interface{}{
		"foo": "foo",
	}, nil)

	diff, err := rb.Diff(state, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if diff == nil {
		t.Fatal("diff shold not be nil")
	}

	actual := testResourceDiffStr(diff)
	expected := testRBPreProcessDiff
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}

	actual = diff.Attributes["foo"].NewExtra.(string)
	expected = "foo"
	if actual != expected {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceBuilder_preProcessUnknown(t *testing.T) {
	rb := &ResourceBuilder{
		Attrs: map[string]AttrType{
			"foo": AttrTypeCreate,
		},

		PreProcess: map[string]PreProcessFunc{
			"foo": func(string) string {
				return "bar"
			},
		},
	}

	state := &terraform.InstanceState{}
	c := testConfig(t, map[string]interface{}{
		"foo": "${var.unknown}",
	}, map[string]string{
		"var.unknown": hcl2shim.UnknownVariableValue,
	})

	diff, err := rb.Diff(state, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if diff == nil {
		t.Fatal("diff shold not be nil")
	}

	actual := testResourceDiffStr(diff)
	expected := testRBPreProcessUnknownDiff
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

	state := &terraform.InstanceState{
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

	state := &terraform.InstanceState{
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
		t.Fatalf("should not diff: %#v", diff)
	}
}

func TestResourceBuilder_unknown(t *testing.T) {
	rb := &ResourceBuilder{
		Attrs: map[string]AttrType{
			"foo": AttrTypeUpdate,
		},
	}

	state := &terraform.InstanceState{}

	c := testConfig(t, map[string]interface{}{
		"foo": "${var.unknown}",
	}, map[string]string{
		"var.foo":     "bar",
		"var.unknown": hcl2shim.UnknownVariableValue,
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

	state := &terraform.InstanceState{}

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

const testRBAttrSetComputedDiff = `CREATE
  IN  foo: "" => "bar" (forces new resource)
`

const testRBComplexDiff = `UPDATE
  IN  listener.0.port: "80" => "3000"
`

const testRBComplexReplaceDiff = `UPDATE
  IN  listener.0.port:  "80" => "<removed>"
  IN  listener.0.value: "" => "50"
`

const testRBComputedAttrUpdate = `UPDATE
  OUT bar: "" => "<computed>"
  IN  foo: "foo" => "bar"
`

const testRBNewDiff = `UPDATE
  IN  foo:        "" => "bar"
  OUT private_ip: "" => "<computed>"
`

const testRBPreProcessDiff = `CREATE
  IN  foo: "" => "barfoo" (forces new resource)
`

const testRBPreProcessUnknownDiff = `CREATE
  IN  foo: "" => "${var.unknown}" (forces new resource)
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
