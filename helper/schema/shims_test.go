package schema

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

var (
	typeComparer  = cmp.Comparer(cty.Type.Equals)
	valueComparer = cmp.Comparer(cty.Value.RawEquals)
	equateEmpty   = cmpopts.EquateEmpty()
)

func testApplyDiff(t *testing.T,
	resource *Resource,
	state, expected *terraform.InstanceState,
	diff *terraform.InstanceDiff) {

	testSchema := providers.Schema{
		Version: uint64(resource.SchemaVersion),
		Block:   resourceSchemaToBlock(resource.Schema),
	}

	stateVal, err := StateValueFromInstanceState(state, testSchema.Block.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	newState, err := ApplyDiff(stateVal, diff, testSchema.Block)
	if err != nil {
		t.Fatal(err)
	}

	// verify that "id" is correct
	id := newState.AsValueMap()["id"]

	switch {
	case diff.Destroy || diff.DestroyDeposed || diff.DestroyTainted:
		// there should be no id
		if !id.IsNull() {
			t.Fatalf("destroyed instance should have no id: %#v", id)
		}
	default:
		// the "id" field always exists and is computed, so it must have a
		// valid value or be unknown.
		if id.IsNull() {
			t.Fatal("new instance state cannot have a null id")
		}

		if id.IsKnown() && id.AsString() == "" {
			t.Fatal("new instance id cannot be an empty string")
		}
	}

	// Resource.Meta will be hanlded separately, so it's OK that we lose the
	// timeout values here.
	expectedState, err := StateValueFromInstanceState(expected, testSchema.Block.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(expectedState, newState, equateEmpty, typeComparer, valueComparer) {
		t.Fatalf(cmp.Diff(expectedState, newState, equateEmpty, typeComparer, valueComparer))
	}
}

func TestShimResourcePlan_destroyCreate(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
				ForceNew: true,
			},
		},
	}

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				RequiresNew: true,
				Old:         "3",
				New:         "42",
			},
		},
	}

	state := &terraform.InstanceState{
		Attributes: map[string]string{"foo": "3"},
	}

	expected := &terraform.InstanceState{
		ID: config.UnknownVariableValue,
		Attributes: map[string]string{
			"id":  config.UnknownVariableValue,
			"foo": "42",
		},
		Meta: map[string]interface{}{
			"schema_version": "2",
		},
	}

	testApplyDiff(t, r, state, expected, d)
}

func TestShimResourceApply_create(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	called := false
	r.Create = func(d *ResourceData, m interface{}) error {
		called = true
		d.SetId("foo")
		return nil
	}

	var s *terraform.InstanceState = nil

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				New: "42",
			},
		},
	}

	actual, err := r.Apply(s, d, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !called {
		t.Fatal("not called")
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "42",
		},
		Meta: map[string]interface{}{
			"schema_version": "2",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	// Shim
	// now that we have our diff and desired state, see if we can reproduce
	// that with the shim
	// we're not testing Resource.Create, so we need to start with the "created" state
	createdState := &terraform.InstanceState{
		ID:         "foo",
		Attributes: map[string]string{"id": "foo"},
	}

	testApplyDiff(t, r, createdState, expected, d)
}

func TestShimResourceApply_Timeout_state(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
		Timeouts: &ResourceTimeout{
			Create: DefaultTimeout(40 * time.Minute),
			Update: DefaultTimeout(80 * time.Minute),
			Delete: DefaultTimeout(40 * time.Minute),
		},
	}

	called := false
	r.Create = func(d *ResourceData, m interface{}) error {
		called = true
		d.SetId("foo")
		return nil
	}

	var s *terraform.InstanceState = nil

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				New: "42",
			},
		},
	}

	diffTimeout := &ResourceTimeout{
		Create: DefaultTimeout(40 * time.Minute),
		Update: DefaultTimeout(80 * time.Minute),
		Delete: DefaultTimeout(40 * time.Minute),
	}

	if err := diffTimeout.DiffEncode(d); err != nil {
		t.Fatalf("Error encoding timeout to diff: %s", err)
	}

	actual, err := r.Apply(s, d, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !called {
		t.Fatal("not called")
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "42",
		},
		Meta: map[string]interface{}{
			"schema_version": "2",
			TimeoutKey:       expectedForValues(40, 0, 80, 40, 0),
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Not equal in Timeout State:\n\texpected: %#v\n\tactual: %#v", expected.Meta, actual.Meta)
	}

	// Shim
	// we're not testing Resource.Create, so we need to start with the "created" state
	createdState := &terraform.InstanceState{
		ID:         "foo",
		Attributes: map[string]string{"id": "foo"},
	}

	testApplyDiff(t, r, createdState, expected, d)
}

func TestShimResourceDiff_Timeout_diff(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
		Timeouts: &ResourceTimeout{
			Create: DefaultTimeout(40 * time.Minute),
			Update: DefaultTimeout(80 * time.Minute),
			Delete: DefaultTimeout(40 * time.Minute),
		},
	}

	r.Create = func(d *ResourceData, m interface{}) error {
		d.SetId("foo")
		return nil
	}

	raw, err := config.NewRawConfig(
		map[string]interface{}{
			"foo": 42,
			"timeouts": []map[string]interface{}{
				map[string]interface{}{
					"create": "2h",
				}},
		})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var s *terraform.InstanceState = nil
	conf := terraform.NewResourceConfig(raw)

	actual, err := r.Diff(s, conf, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				New: "42",
			},
		},
	}

	diffTimeout := &ResourceTimeout{
		Create: DefaultTimeout(120 * time.Minute),
		Update: DefaultTimeout(80 * time.Minute),
		Delete: DefaultTimeout(40 * time.Minute),
	}

	if err := diffTimeout.DiffEncode(expected); err != nil {
		t.Fatalf("Error encoding timeout to diff: %s", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Not equal in Timeout Diff:\n\texpected: %#v\n\tactual: %#v", expected.Meta, actual.Meta)
	}

	// Shim
	// apply this diff, so we have a state to compare
	applied, err := r.Apply(s, actual, nil)
	if err != nil {
		t.Fatal(err)
	}

	// we're not testing Resource.Create, so we need to start with the "created" state
	createdState := &terraform.InstanceState{
		ID:         "foo",
		Attributes: map[string]string{"id": "foo"},
	}

	testSchema := providers.Schema{
		Version: uint64(r.SchemaVersion),
		Block:   resourceSchemaToBlock(r.Schema),
	}

	initialVal, err := StateValueFromInstanceState(createdState, testSchema.Block.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	appliedVal, err := StateValueFromInstanceState(applied, testSchema.Block.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	d, err := DiffFromValues(initialVal, appliedVal, r)
	if err != nil {
		t.Fatal(err)
	}
	if eq, _ := d.Same(expected); !eq {
		t.Fatal(cmp.Diff(d, expected))
	}
}

func TestShimResourceApply_destroy(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	called := false
	r.Delete = func(d *ResourceData, m interface{}) error {
		called = true
		return nil
	}

	s := &terraform.InstanceState{
		ID: "bar",
	}

	d := &terraform.InstanceDiff{
		Destroy: true,
	}

	actual, err := r.Apply(s, d, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !called {
		t.Fatal("delete not called")
	}

	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}

	// Shim
	// now that we have our diff and desired state, see if we can reproduce
	// that with the shim
	testApplyDiff(t, r, s, actual, d)
}

func TestShimResourceApply_destroyCreate(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"tags": &Schema{
				Type:     TypeMap,
				Optional: true,
				Computed: true,
			},
		},
	}

	change := false
	r.Create = func(d *ResourceData, m interface{}) error {
		change = d.HasChange("tags")
		d.SetId("foo")
		return nil
	}
	r.Delete = func(d *ResourceData, m interface{}) error {
		return nil
	}

	var s *terraform.InstanceState = &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo":       "7",
			"tags.Name": "foo",
		},
	}

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"id": &terraform.ResourceAttrDiff{
				New: "foo",
			},
			"foo": &terraform.ResourceAttrDiff{
				Old:         "7",
				New:         "42",
				RequiresNew: true,
			},
			"tags.Name": &terraform.ResourceAttrDiff{
				Old:         "foo",
				New:         "foo",
				RequiresNew: true,
			},
		},
	}

	actual, err := r.Apply(s, d, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !change {
		t.Fatal("should have change")
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":        "foo",
			"foo":       "42",
			"tags.%":    "1",
			"tags.Name": "foo",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		cmp.Diff(actual, expected)
	}

	// Shim
	// now that we have our diff and desired state, see if we can reproduce
	// that with the shim
	// we're not testing Resource.Create, so we need to start with the "created" state
	createdState := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":        "foo",
			"foo":       "7",
			"tags.%":    "1",
			"tags.Name": "foo",
		},
	}

	testApplyDiff(t, r, createdState, expected, d)
}

func TestShimSchemaMap_Diff(t *testing.T) {
	cases := []struct {
		Name            string
		Schema          map[string]*Schema
		State           *terraform.InstanceState
		Config          map[string]interface{}
		ConfigVariables map[string]ast.Variable
		CustomizeDiff   CustomizeDiffFunc
		Diff            *terraform.InstanceDiff
		Err             bool
	}{
		{
			Name: "diff-1",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "diff-2",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "",
						NewComputed: true,
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "diff-3",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: &terraform.InstanceState{
				ID: "foo",
			},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Computed, but set in config",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
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

			Config: map[string]interface{}{
				"availability_zone": "bar",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old: "foo",
						New: "bar",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Default",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Default:  "foo",
				},
			},

			State: nil,

			Config: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old: "",
						New: "foo",
					},
				},
			},

			Err: false,
		},

		{
			Name: "DefaultFunc, value",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					DefaultFunc: func() (interface{}, error) {
						return "foo", nil
					},
				},
			},

			State: nil,

			Config: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old: "",
						New: "foo",
					},
				},
			},

			Err: false,
		},

		{
			Name: "DefaultFunc, configuration set",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					DefaultFunc: func() (interface{}, error) {
						return "foo", nil
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "bar",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old: "",
						New: "bar",
					},
				},
			},

			Err: false,
		},

		{
			Name: "String with StateFunc",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					StateFunc: func(a interface{}) string {
						return a.(string) + "!"
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:      "",
						New:      "foo!",
						NewExtra: "foo",
					},
				},
			},

			Err: false,
		},

		{
			Name: "StateFunc not called with nil value",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					StateFunc: func(a interface{}) string {
						t.Error("should not get here!")
						return ""
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Variable (just checking)",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "${var.foo}",
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError("bar"),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old: "",
						New: "bar",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Variable computed",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "${var.foo}",
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "${var.foo}",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Int decode",
			Schema: map[string]*Schema{
				"port": &Schema{
					Type:     TypeInt,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"port": 27,
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"port": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "27",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "bool decode",
			Schema: map[string]*Schema{
				"port": &Schema{
					Type:     TypeBool,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"port": false,
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"port": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "false",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Bool",
			Schema: map[string]*Schema{
				"delete": &Schema{
					Type:     TypeBool,
					Optional: true,
					Default:  false,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"delete": "false",
				},
			},

			Config: nil,

			Diff: nil,

			Err: false,
		},

		{
			Name: "List decode",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, 2, 5},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "3",
					},
					"ports.0": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "",
						New: "2",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		/*
			// disabled for shims
			// there is no longer any "list promotion"
			{
				Name: "List decode with promotion",
				Schema: map[string]*Schema{
					"ports": &Schema{
						Type:          TypeList,
						Required:      true,
						Elem:          &Schema{Type: TypeInt},
						PromoteSingle: true,
					},
				},

				State: nil,

				Config: map[string]interface{}{
					"ports": "5",
				},

				Diff: &terraform.InstanceDiff{
					Attributes: map[string]*terraform.ResourceAttrDiff{
						"ports.#": &terraform.ResourceAttrDiff{
							Old: "0",
							New: "1",
						},
						"ports.0": &terraform.ResourceAttrDiff{
							Old: "",
							New: "5",
						},
					},
				},

				Err: false,
			},
		*/

		{
			Name: "List decode with promotion with list",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:          TypeList,
					Required:      true,
					Elem:          &Schema{Type: TypeInt},
					PromoteSingle: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{"5"},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"ports.0": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, "${var.foo}"},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError([]interface{}{"2", "5"}),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "3",
					},
					"ports.0": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "",
						New: "2",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, "${var.foo}"},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError([]interface{}{
					config.UnknownVariableValue, "5"}),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "0",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "3",
					"ports.0": "1",
					"ports.1": "2",
					"ports.2": "5",
				},
			},

			Config: map[string]interface{}{
				"ports": []interface{}{1, 2, 5},
			},

			Diff: nil,

			Err: false,
		},

		{
			Name: "",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "2",
					"ports.0": "1",
					"ports.1": "2",
				},
			},

			Config: map[string]interface{}{
				"ports": []interface{}{1, 2, 5},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "3",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Name: "",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, 2, 5},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "0",
						New:         "3",
						RequiresNew: true,
					},
					"ports.0": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "1",
						RequiresNew: true,
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "2",
						RequiresNew: true,
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "5",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "List with computed set",
			Schema: map[string]*Schema{
				"config": &Schema{
					Type:     TypeList,
					Optional: true,
					ForceNew: true,
					MinItems: 1,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"name": {
								Type:     TypeString,
								Required: true,
							},

							"rules": {
								Type:     TypeSet,
								Computed: true,
								Elem:     &Schema{Type: TypeString},
								Set:      HashString,
							},
						},
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"config": []interface{}{
					map[string]interface{}{
						"name": "hello",
					},
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config.#": &terraform.ResourceAttrDiff{
						Old:         "0",
						New:         "1",
						RequiresNew: true,
					},

					"config.0.name": &terraform.ResourceAttrDiff{
						Old: "",
						New: "hello",
					},

					"config.0.rules.#": &terraform.ResourceAttrDiff{
						Old:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set-1",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{5, 2, 1},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "3",
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "",
						New: "2",
					},
					"ports.5": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set-2",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Computed: true,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "0",
				},
			},

			Config: nil,

			Diff: nil,

			Err: false,
		},

		{
			Name: "Set-3",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: nil,

			Config: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set-4",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{"${var.foo}", 1},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError([]interface{}{"2", "5"}),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "3",
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "",
						New: "2",
					},
					"ports.5": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set-5",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, "${var.foo}"},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError([]interface{}{
					config.UnknownVariableValue, "5"}),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set-6",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "2",
					"ports.1": "1",
					"ports.2": "2",
				},
			},

			Config: map[string]interface{}{
				"ports": []interface{}{5, 2, 1},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "3",
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "1",
						New: "1",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "2",
					},
					"ports.5": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		/*
			// disabled for shims
			// you can't remove a required attribute
			{
				Name: "Set-7",
				Schema: map[string]*Schema{
					"ports": &Schema{
						Type:     TypeSet,
						Required: true,
						Elem:     &Schema{Type: TypeInt},
						Set: func(a interface{}) int {
							return a.(int)
						},
					},
				},

				State: &terraform.InstanceState{
					Attributes: map[string]string{
						"ports.#": "2",
						"ports.1": "1",
						"ports.2": "2",
					},
				},

				Config: map[string]interface{}{},

				Diff: &terraform.InstanceDiff{
					Attributes: map[string]*terraform.ResourceAttrDiff{
						"ports.#": &terraform.ResourceAttrDiff{
							Old: "2",
							New: "0",
						},
						"ports.1": &terraform.ResourceAttrDiff{
							Old:        "1",
							New:        "0",
							NewRemoved: true,
						},
						"ports.2": &terraform.ResourceAttrDiff{
							Old:        "2",
							New:        "0",
							NewRemoved: true,
						},
					},
				},

				Err: false,
			},
		*/

		{
			Name: "Set-8",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "bar",
					"ports.#":           "1",
					"ports.80":          "80",
				},
			},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Set-9",
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"ports": &Schema{
								Type:     TypeList,
								Optional: true,
								Elem:     &Schema{Type: TypeInt},
							},
						},
					},
					Set: func(v interface{}) int {
						m := v.(map[string]interface{})
						ps := m["ports"].([]interface{})
						result := 0
						for _, p := range ps {
							result += p.(int)
						}
						return result
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ingress.#":           "2",
					"ingress.80.ports.#":  "1",
					"ingress.80.ports.0":  "80",
					"ingress.443.ports.#": "1",
					"ingress.443.ports.0": "443",
				},
			},

			Config: map[string]interface{}{
				"ingress": []interface{}{
					map[string]interface{}{
						"ports": []interface{}{443},
					},
					map[string]interface{}{
						"ports": []interface{}{80},
					},
				},
			},

			Diff: nil,

			Err: false,
		},

		{
			Name: "List of structure decode",
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:     TypeList,
					Required: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"from": &Schema{
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ingress": []interface{}{
					map[string]interface{}{
						"from": 8080,
					},
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ingress.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"ingress.0.from": &terraform.ResourceAttrDiff{
						Old: "",
						New: "8080",
					},
				},
			},

			Err: false,
		},

		{
			Name: "ComputedWhen",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:         TypeString,
					Computed:     true,
					ComputedWhen: []string{"port"},
				},

				"port": &Schema{
					Type:     TypeInt,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
					"port":              "80",
				},
			},

			Config: map[string]interface{}{
				"port": 80,
			},

			Diff: nil,

			Err: false,
		},

		{
			Name: "",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:         TypeString,
					Computed:     true,
					ComputedWhen: []string{"port"},
				},

				"port": &Schema{
					Type:     TypeInt,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"port": "80",
				},
			},

			Config: map[string]interface{}{
				"port": 80,
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Maps-1",
			Schema: map[string]*Schema{
				"config_vars": &Schema{
					Type: TypeMap,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"config_vars": map[string]interface{}{
					"bar": "baz",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config_vars.%": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},

					"config_vars.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "baz",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Maps-2",
			Schema: map[string]*Schema{
				"config_vars": &Schema{
					Type: TypeMap,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"config_vars.%":   "1",
					"config_vars.foo": "bar",
				},
			},

			Config: map[string]interface{}{
				"config_vars": map[string]interface{}{
					"bar": "baz",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config_vars.foo": &terraform.ResourceAttrDiff{
						Old:        "bar",
						NewRemoved: true,
					},
					"config_vars.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "baz",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Maps-3",
			Schema: map[string]*Schema{
				"vars": &Schema{
					Type:     TypeMap,
					Optional: true,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"vars.%":   "1",
					"vars.foo": "bar",
				},
			},

			Config: map[string]interface{}{
				"vars": map[string]interface{}{
					"bar": "baz",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"vars.foo": &terraform.ResourceAttrDiff{
						Old:        "bar",
						New:        "",
						NewRemoved: true,
					},
					"vars.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "baz",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Maps-4",
			Schema: map[string]*Schema{
				"vars": &Schema{
					Type:     TypeMap,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"vars.%":   "1",
					"vars.foo": "bar",
				},
			},

			Config: nil,

			Diff: nil,

			Err: false,
		},

		{
			Name: "Maps-5",
			Schema: map[string]*Schema{
				"config_vars": &Schema{
					Type: TypeList,
					Elem: &Schema{Type: TypeMap},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"config_vars.#":     "1",
					"config_vars.0.%":   "1",
					"config_vars.0.foo": "bar",
				},
			},

			Config: map[string]interface{}{
				"config_vars": []interface{}{
					map[string]interface{}{
						"bar": "baz",
					},
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config_vars.0.foo": &terraform.ResourceAttrDiff{
						Old:        "bar",
						NewRemoved: true,
					},
					"config_vars.0.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "baz",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Maps-6",
			Schema: map[string]*Schema{
				"config_vars": &Schema{
					Type:     TypeList,
					Elem:     &Schema{Type: TypeMap},
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"config_vars.#":     "1",
					"config_vars.0.%":   "2",
					"config_vars.0.foo": "bar",
					"config_vars.0.bar": "baz",
				},
			},

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config_vars.#": &terraform.ResourceAttrDiff{
						Old: "1",
						New: "0",
					},
					"config_vars.0.%": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "0",
					},
					"config_vars.0.foo": &terraform.ResourceAttrDiff{
						Old:        "bar",
						NewRemoved: true,
					},
					"config_vars.0.bar": &terraform.ResourceAttrDiff{
						Old:        "baz",
						NewRemoved: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "ForceNews",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					ForceNew: true,
				},

				"address": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "bar",
					"address":           "foo",
				},
			},

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "bar",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set-10",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					ForceNew: true,
				},

				"ports": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "bar",
					"ports.#":           "1",
					"ports.80":          "80",
				},
			},

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "bar",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set-11",
			Schema: map[string]*Schema{
				"instances": &Schema{
					Type:     TypeSet,
					Elem:     &Schema{Type: TypeString},
					Optional: true,
					Computed: true,
					Set: func(v interface{}) int {
						return len(v.(string))
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"instances.#": "0",
				},
			},

			Config: map[string]interface{}{
				"instances": []interface{}{"${var.foo}"},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"instances.#": &terraform.ResourceAttrDiff{
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set-12",
			Schema: map[string]*Schema{
				"route": &Schema{
					Type:     TypeSet,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"index": &Schema{
								Type:     TypeInt,
								Required: true,
							},

							"gateway": &Schema{
								Type:     TypeString,
								Optional: true,
							},
						},
					},
					Set: func(v interface{}) int {
						m := v.(map[string]interface{})
						return m["index"].(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"route": []interface{}{
					map[string]interface{}{
						"index":   "1",
						"gateway": "${var.foo}",
					},
				},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"route.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"route.~1.index": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"route.~1.gateway": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "${var.foo}",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set-13",
			Schema: map[string]*Schema{
				"route": &Schema{
					Type:     TypeSet,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"index": &Schema{
								Type:     TypeInt,
								Required: true,
							},

							"gateway": &Schema{
								Type:     TypeSet,
								Optional: true,
								Elem:     &Schema{Type: TypeInt},
								Set: func(a interface{}) int {
									return a.(int)
								},
							},
						},
					},
					Set: func(v interface{}) int {
						m := v.(map[string]interface{})
						return m["index"].(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"route": []interface{}{
					map[string]interface{}{
						"index": "1",
						"gateway": []interface{}{
							"${var.foo}",
						},
					},
				},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"route.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"route.~1.index": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"route.~1.gateway.#": &terraform.ResourceAttrDiff{
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Computed maps",
			Schema: map[string]*Schema{
				"vars": &Schema{
					Type:     TypeMap,
					Computed: true,
				},
			},

			State: nil,

			Config: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"vars.%": &terraform.ResourceAttrDiff{
						Old:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Computed maps",
			Schema: map[string]*Schema{
				"vars": &Schema{
					Type:     TypeMap,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"vars.%": "0",
				},
			},

			Config: map[string]interface{}{
				"vars": map[string]interface{}{
					"bar": "${var.foo}",
				},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"vars.%": &terraform.ResourceAttrDiff{
						Old:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name:   " - Empty",
			Schema: map[string]*Schema{},

			State: &terraform.InstanceState{},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Float",
			Schema: map[string]*Schema{
				"some_threshold": &Schema{
					Type: TypeFloat,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"some_threshold": "567.8",
				},
			},

			Config: map[string]interface{}{
				"some_threshold": 12.34,
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"some_threshold": &terraform.ResourceAttrDiff{
						Old: "567.8",
						New: "12.34",
					},
				},
			},

			Err: false,
		},

		{
			Name: "https://github.com/hashicorp/terraform/issues/824",
			Schema: map[string]*Schema{
				"block_device": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"device_name": &Schema{
								Type:     TypeString,
								Required: true,
							},
							"delete_on_termination": &Schema{
								Type:     TypeBool,
								Optional: true,
								Default:  true,
							},
						},
					},
					Set: func(v interface{}) int {
						var buf bytes.Buffer
						m := v.(map[string]interface{})
						buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
						buf.WriteString(fmt.Sprintf("%t-", m["delete_on_termination"].(bool)))
						return hashcode.String(buf.String())
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"block_device.#": "2",
					"block_device.616397234.delete_on_termination":  "true",
					"block_device.616397234.device_name":            "/dev/sda1",
					"block_device.2801811477.delete_on_termination": "true",
					"block_device.2801811477.device_name":           "/dev/sdx",
				},
			},

			Config: map[string]interface{}{
				"block_device": []interface{}{
					map[string]interface{}{
						"device_name": "/dev/sda1",
					},
					map[string]interface{}{
						"device_name": "/dev/sdx",
					},
				},
			},
			Diff: nil,
			Err:  false,
		},

		{
			Name: "Zero value in state shouldn't result in diff",
			Schema: map[string]*Schema{
				"port": &Schema{
					Type:     TypeBool,
					Optional: true,
					ForceNew: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"port": "false",
				},
			},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Same as prev, but for sets",
			Schema: map[string]*Schema{
				"route": &Schema{
					Type:     TypeSet,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"index": &Schema{
								Type:     TypeInt,
								Required: true,
							},

							"gateway": &Schema{
								Type:     TypeSet,
								Optional: true,
								Elem:     &Schema{Type: TypeInt},
								Set: func(a interface{}) int {
									return a.(int)
								},
							},
						},
					},
					Set: func(v interface{}) int {
						m := v.(map[string]interface{})
						return m["index"].(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"route.#": "0",
				},
			},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "A set computed element shouldn't cause a diff",
			Schema: map[string]*Schema{
				"active": &Schema{
					Type:     TypeBool,
					Computed: true,
					ForceNew: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"active": "true",
				},
			},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "An empty set should show up in the diff",
			Schema: map[string]*Schema{
				"instances": &Schema{
					Type:     TypeSet,
					Elem:     &Schema{Type: TypeString},
					Optional: true,
					ForceNew: true,
					Set: func(v interface{}) int {
						return len(v.(string))
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"instances.#": "1",
					"instances.3": "foo",
				},
			},

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"instances.#": &terraform.ResourceAttrDiff{
						Old:         "1",
						New:         "0",
						RequiresNew: true,
					},
					"instances.3": &terraform.ResourceAttrDiff{
						Old:         "foo",
						New:         "",
						NewRemoved:  true,
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Map with empty value",
			Schema: map[string]*Schema{
				"vars": &Schema{
					Type: TypeMap,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"vars": map[string]interface{}{
					"foo": "",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"vars.%": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"vars.foo": &terraform.ResourceAttrDiff{
						Old: "",
						New: "",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Unset bool, not in state",
			Schema: map[string]*Schema{
				"force": &Schema{
					Type:     TypeBool,
					Optional: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Unset set, not in state",
			Schema: map[string]*Schema{
				"metadata_keys": &Schema{
					Type:     TypeSet,
					Optional: true,
					ForceNew: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(interface{}) int { return 0 },
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Unset list in state, should not show up computed",
			Schema: map[string]*Schema{
				"metadata_keys": &Schema{
					Type:     TypeList,
					Optional: true,
					Computed: true,
					ForceNew: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"metadata_keys.#": "0",
				},
			},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Set element computed substring",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, "${var.foo}32"},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Computed map without config that's known to be empty does not generate diff",
			Schema: map[string]*Schema{
				"tags": &Schema{
					Type:     TypeMap,
					Computed: true,
				},
			},

			Config: nil,

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"tags.%": "0",
				},
			},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Set with hyphen keys",
			Schema: map[string]*Schema{
				"route": &Schema{
					Type:     TypeSet,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"index": &Schema{
								Type:     TypeInt,
								Required: true,
							},

							"gateway-name": &Schema{
								Type:     TypeString,
								Optional: true,
							},
						},
					},
					Set: func(v interface{}) int {
						m := v.(map[string]interface{})
						return m["index"].(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"route": []interface{}{
					map[string]interface{}{
						"index":        "1",
						"gateway-name": "hello",
					},
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"route.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"route.1.index": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"route.1.gateway-name": &terraform.ResourceAttrDiff{
						Old: "",
						New: "hello",
					},
				},
			},

			Err: false,
		},

		{
			Name: ": StateFunc in nested set (#1759)",
			Schema: map[string]*Schema{
				"service_account": &Schema{
					Type:     TypeList,
					Optional: true,
					ForceNew: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"scopes": &Schema{
								Type:     TypeSet,
								Required: true,
								ForceNew: true,
								Elem: &Schema{
									Type: TypeString,
									StateFunc: func(v interface{}) string {
										return v.(string) + "!"
									},
								},
								Set: func(v interface{}) int {
									i, err := strconv.Atoi(v.(string))
									if err != nil {
										t.Fatalf("err: %s", err)
									}
									return i
								},
							},
						},
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"service_account": []interface{}{
					map[string]interface{}{
						"scopes": []interface{}{"123"},
					},
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"service_account.#": &terraform.ResourceAttrDiff{
						Old:         "0",
						New:         "1",
						RequiresNew: true,
					},
					"service_account.0.scopes.#": &terraform.ResourceAttrDiff{
						Old:         "0",
						New:         "1",
						RequiresNew: true,
					},
					"service_account.0.scopes.123": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "123!",
						NewExtra:    "123",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Removing set elements",
			Schema: map[string]*Schema{
				"instances": &Schema{
					Type:     TypeSet,
					Elem:     &Schema{Type: TypeString},
					Optional: true,
					ForceNew: true,
					Set: func(v interface{}) int {
						return len(v.(string))
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"instances.#": "2",
					"instances.3": "333",
					"instances.2": "22",
				},
			},

			Config: map[string]interface{}{
				"instances": []interface{}{"333", "4444"},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"instances.2": &terraform.ResourceAttrDiff{
						Old:         "22",
						New:         "",
						NewRemoved:  true,
						RequiresNew: true,
					},
					"instances.3": &terraform.ResourceAttrDiff{
						Old: "333",
						New: "333",
					},
					"instances.4": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "4444",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Bools can be set with 0/1 in config, still get true/false",
			Schema: map[string]*Schema{
				"one": &Schema{
					Type:     TypeBool,
					Optional: true,
				},
				"two": &Schema{
					Type:     TypeBool,
					Optional: true,
				},
				"three": &Schema{
					Type:     TypeBool,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"one":   "false",
					"two":   "true",
					"three": "true",
				},
			},

			Config: map[string]interface{}{
				"one": "1",
				"two": "0",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"one": &terraform.ResourceAttrDiff{
						Old: "false",
						New: "true",
					},
					"two": &terraform.ResourceAttrDiff{
						Old: "true",
						New: "false",
					},
					"three": &terraform.ResourceAttrDiff{
						Old:        "true",
						New:        "false",
						NewRemoved: true,
					},
				},
			},

			Err: false,
		},

		{
			Name:   "tainted in state w/ no attr changes is still a replacement",
			Schema: map[string]*Schema{},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"id": "someid",
				},
				Tainted: true,
			},

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes:     map[string]*terraform.ResourceAttrDiff{},
				DestroyTainted: true,
			},
		},

		{
			Name: "Set ForceNew only marks the changing element as ForceNew",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					ForceNew: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "3",
					"ports.1": "1",
					"ports.2": "2",
					"ports.4": "4",
				},
			},

			Config: map[string]interface{}{
				"ports": []interface{}{5, 2, 1},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "1",
						New: "1",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "2",
					},
					"ports.5": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "5",
						RequiresNew: true,
					},
					"ports.4": &terraform.ResourceAttrDiff{
						Old:         "4",
						New:         "0",
						NewRemoved:  true,
						RequiresNew: true,
					},
				},
			},
		},

		{
			Name: "removed optional items should trigger ForceNew",
			Schema: map[string]*Schema{
				"description": &Schema{
					Type:     TypeString,
					ForceNew: true,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"description": "foo",
				},
			},

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"description": &terraform.ResourceAttrDiff{
						Old:         "foo",
						New:         "",
						RequiresNew: true,
						NewRemoved:  true,
					},
				},
			},

			Err: false,
		},

		// GH-7715
		{
			Name: "computed value for boolean field",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeBool,
					ForceNew: true,
					Computed: true,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{},

			Config: map[string]interface{}{
				"foo": "${var.foo}",
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(
					config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "false",
						NewComputed: true,
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set ForceNew marks count as ForceNew if computed",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					ForceNew: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "3",
					"ports.1": "1",
					"ports.2": "2",
					"ports.4": "4",
				},
			},

			Config: map[string]interface{}{
				"ports": []interface{}{"${var.foo}", 2, 1},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						NewComputed: true,
						RequiresNew: true,
					},
				},
			},
		},

		{
			Name: "List with computed schema and ForceNew",
			Schema: map[string]*Schema{
				"config": &Schema{
					Type:     TypeList,
					Optional: true,
					ForceNew: true,
					Elem: &Schema{
						Type: TypeString,
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"config.#": "2",
					"config.0": "a",
					"config.1": "b",
				},
			},

			Config: map[string]interface{}{
				"config": []interface{}{"${var.a}", "${var.b}"},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.a": interfaceToVariableSwallowError(
					config.UnknownVariableValue),
				"var.b": interfaceToVariableSwallowError(
					config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config.#": &terraform.ResourceAttrDiff{
						Old:         "2",
						New:         "",
						RequiresNew: true,
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "overridden diff with a CustomizeDiff function, ForceNew not in schema",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},

			CustomizeDiff: func(d *ResourceDiff, meta interface{}) error {
				if err := d.SetNew("availability_zone", "bar"); err != nil {
					return err
				}
				if err := d.ForceNew("availability_zone"); err != nil {
					return err
				}
				return nil
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "bar",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			// NOTE: This case is technically impossible in the current
			// implementation, because optional+computed values never show up in the
			// diff. In the event behavior changes this test should ensure that the
			// intended diff still shows up.
			Name: "overridden removed attribute diff with a CustomizeDiff function, ForceNew not in schema",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			CustomizeDiff: func(d *ResourceDiff, meta interface{}) error {
				if err := d.SetNew("availability_zone", "bar"); err != nil {
					return err
				}
				if err := d.ForceNew("availability_zone"); err != nil {
					return err
				}
				return nil
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "bar",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{

			Name: "overridden diff with a CustomizeDiff function, ForceNew in schema",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},

			CustomizeDiff: func(d *ResourceDiff, meta interface{}) error {
				if err := d.SetNew("availability_zone", "bar"); err != nil {
					return err
				}
				return nil
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "bar",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "required field with computed diff added with CustomizeDiff function",
			Schema: map[string]*Schema{
				"ami_id": &Schema{
					Type:     TypeString,
					Required: true,
				},
				"instance_id": &Schema{
					Type:     TypeString,
					Computed: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ami_id": "foo",
			},

			CustomizeDiff: func(d *ResourceDiff, meta interface{}) error {
				if err := d.SetNew("instance_id", "bar"); err != nil {
					return err
				}
				return nil
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ami_id": &terraform.ResourceAttrDiff{
						Old: "",
						New: "foo",
					},
					"instance_id": &terraform.ResourceAttrDiff{
						Old: "",
						New: "bar",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set ForceNew only marks the changing element as ForceNew - CustomizeDiffFunc edition",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "3",
					"ports.1": "1",
					"ports.2": "2",
					"ports.4": "4",
				},
			},

			Config: map[string]interface{}{
				"ports": []interface{}{5, 2, 6},
			},

			CustomizeDiff: func(d *ResourceDiff, meta interface{}) error {
				if err := d.SetNew("ports", []interface{}{5, 2, 1}); err != nil {
					return err
				}
				if err := d.ForceNew("ports"); err != nil {
					return err
				}
				return nil
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "1",
						New: "1",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "2",
					},
					"ports.5": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "5",
						RequiresNew: true,
					},
					"ports.4": &terraform.ResourceAttrDiff{
						Old:         "4",
						New:         "0",
						NewRemoved:  true,
						RequiresNew: true,
					},
				},
			},
		},

		{
			Name:   "tainted resource does not run CustomizeDiffFunc",
			Schema: map[string]*Schema{},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"id": "someid",
				},
				Tainted: true,
			},

			Config: map[string]interface{}{},

			CustomizeDiff: func(d *ResourceDiff, meta interface{}) error {
				return errors.New("diff customization should not have run")
			},

			Diff: &terraform.InstanceDiff{
				Attributes:     map[string]*terraform.ResourceAttrDiff{},
				DestroyTainted: true,
			},

			Err: false,
		},

		{
			Name: "NewComputed based on a conditional with CustomizeDiffFunc",
			Schema: map[string]*Schema{
				"etag": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
				"version_id": &Schema{
					Type:     TypeString,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"etag":       "foo",
					"version_id": "1",
				},
			},

			Config: map[string]interface{}{
				"etag": "bar",
			},

			CustomizeDiff: func(d *ResourceDiff, meta interface{}) error {
				if d.HasChange("etag") {
					d.SetNewComputed("version_id")
				}
				return nil
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"etag": &terraform.ResourceAttrDiff{
						Old: "foo",
						New: "bar",
					},
					"version_id": &terraform.ResourceAttrDiff{
						Old:         "1",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "vetoing a diff",
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

			Config: map[string]interface{}{
				"foo": "baz",
			},

			CustomizeDiff: func(d *ResourceDiff, meta interface{}) error {
				return fmt.Errorf("diff vetoed")
			},

			Err: true,
		},

		// A lot of resources currently depended on using the empty string as a
		// nil/unset value.
		{
			Name: "optional, computed, empty string",
			Schema: map[string]*Schema{
				"attr": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"attr": "bar",
				},
			},

			// this does necessarily depend on an interpolated value, but this
			// is often how it comes about in a configuration, otherwise the
			// value would be unset.
			Config: map[string]interface{}{
				"attr": "${var.foo}",
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(""),
			},
		},

		{
			Name: "optional, computed, empty string should not crash in CustomizeDiff",
			Schema: map[string]*Schema{
				"unrelated_set": {
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeString},
				},
				"stream_enabled": {
					Type:     TypeBool,
					Optional: true,
				},
				"stream_view_type": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"unrelated_set.#":  "0",
					"stream_enabled":   "true",
					"stream_view_type": "KEYS_ONLY",
				},
			},
			Config: map[string]interface{}{
				"stream_enabled":   false,
				"stream_view_type": "",
			},
			CustomizeDiff: func(diff *ResourceDiff, v interface{}) error {
				v, ok := diff.GetOk("unrelated_set")
				if ok {
					return fmt.Errorf("Didn't expect unrelated_set: %#v", v)
				}
				return nil
			},
			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"stream_enabled": {
						Old: "true",
						New: "false",
					},
				},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			c, err := config.NewRawConfig(tc.Config)
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			if len(tc.ConfigVariables) > 0 {
				if err := c.Interpolate(tc.ConfigVariables); err != nil {
					t.Fatalf("err: %s", err)
				}
			}

			{
				d, err := InternalMap(tc.Schema).Diff(tc.State, terraform.NewResourceConfig(c), tc.CustomizeDiff, nil, false)
				if err != nil != tc.Err {
					t.Fatalf("err: %s", err)
				}

				if !cmp.Equal(d, tc.Diff, equateEmpty) {
					t.Fatal(cmp.Diff(d, tc.Diff, equateEmpty))
				}
			}
			// up to here is already tested in helper/schema; we're just
			// verify that we haven't broken any tests in transition.

			// create a schema from the schemaMap
			testSchema := resourceSchemaToBlock(tc.Schema)

			// get our initial state cty.Value
			stateVal, err := StateValueFromInstanceState(tc.State, testSchema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			// this is the desired cty.Value from the configuration
			configVal := hcl2shim.HCL2ValueFromConfigValue(c.Config())

			// verify that we can round-trip the config
			origConfig := hcl2shim.ConfigValueFromHCL2(configVal)
			if !cmp.Equal(c.Config(), origConfig, equateEmpty) {
				t.Fatal(cmp.Diff(c.Config(), origConfig, equateEmpty))
			}

			// make sure our config conforms precisely to the schema
			configVal, err = testSchema.CoerceValue(configVal)
			if err != nil {
				t.Fatal(tfdiags.FormatError(err))
			}

			// The new API requires returning the desired state rather than a
			// diff, so we need to verify that we can combine the state and
			// diff and recreate a new state.

			// now verify that we can create diff, using the new config and state values
			// customize isn't run on tainted resources
			tainted := tc.State != nil && tc.State.Tainted
			if tainted {
				tc.CustomizeDiff = nil
			}

			res := &Resource{Schema: tc.Schema}
			d, err := diffFromValues(stateVal, configVal, res, tc.CustomizeDiff)
			if err != nil {
				if !tc.Err {
					t.Fatal(err)
				}
			}

			// our diff function can't set DestroyTainted, but match the
			// expected value here for the test fixtures
			if tainted {
				if d == nil {
					d = &terraform.InstanceDiff{}
				}
				d.DestroyTainted = true
			}

			if eq, _ := d.Same(tc.Diff); !eq {
				t.Fatal(cmp.Diff(d, tc.Diff))
			}

		})
	}
}

func resourceSchemaToBlock(s map[string]*Schema) *configschema.Block {
	return (&Resource{Schema: s}).CoreConfigSchema()
}
