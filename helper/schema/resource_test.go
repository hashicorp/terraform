package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/terraform"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func TestResourceApply_create(t *testing.T) {
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
}

func TestResourceApply_Timeout_state(t *testing.T) {
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
}

// Regression test to ensure that the meta data is read from state, if a
// resource is destroyed and the timeout meta is no longer available from the
// config
func TestResourceApply_Timeout_destroy(t *testing.T) {
	timeouts := &ResourceTimeout{
		Create: DefaultTimeout(40 * time.Minute),
		Update: DefaultTimeout(80 * time.Minute),
		Delete: DefaultTimeout(40 * time.Minute),
	}

	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
		Timeouts: timeouts,
	}

	called := false
	var delTimeout time.Duration
	r.Delete = func(d *ResourceData, m interface{}) error {
		delTimeout = d.Timeout(TimeoutDelete)
		called = true
		return nil
	}

	s := &terraform.InstanceState{
		ID: "bar",
	}

	if err := timeouts.StateEncode(s); err != nil {
		t.Fatalf("Error encoding to state: %s", err)
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

	if *timeouts.Delete != delTimeout {
		t.Fatalf("timeouts don't match, expected (%#v), got (%#v)", timeouts.Delete, delTimeout)
	}

	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceDiff_Timeout_diff(t *testing.T) {
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
}

func TestResourceDiff_CustomizeFunc(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	var called bool

	r.CustomizeDiff = func(d *ResourceDiff, m interface{}) error {
		called = true
		return nil
	}

	raw, err := config.NewRawConfig(
		map[string]interface{}{
			"foo": 42,
		})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var s *terraform.InstanceState
	conf := terraform.NewResourceConfig(raw)

	_, err = r.Diff(s, conf, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !called {
		t.Fatalf("diff customization not called")
	}
}

func TestResourceApply_destroy(t *testing.T) {
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
}

func TestResourceApply_destroyCreate(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
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
			"foo":       "bar",
			"tags.Name": "foo",
		},
	}

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
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
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceApply_destroyPartial(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
		SchemaVersion: 3,
	}

	r.Delete = func(d *ResourceData, m interface{}) error {
		d.Set("foo", 42)
		return fmt.Errorf("some error")
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	d := &terraform.InstanceDiff{
		Destroy: true,
	}

	actual, err := r.Apply(s, d, nil)
	if err == nil {
		t.Fatal("should error")
	}

	expected := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"id":  "bar",
			"foo": "42",
		},
		Meta: map[string]interface{}{
			"schema_version": "3",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected:\n%#v\n\ngot:\n%#v", expected, actual)
	}
}

func TestResourceApply_update(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Update = func(d *ResourceData, m interface{}) error {
		d.Set("foo", 42)
		return nil
	}

	s := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				New: "13",
			},
		},
	}

	actual, err := r.Apply(s, d, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "42",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceApply_updateNoCallback(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Update = nil

	s := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				New: "13",
			},
		},
	}

	actual, err := r.Apply(s, d, nil)
	if err == nil {
		t.Fatal("should error")
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceApply_isNewResource(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeString,
				Optional: true,
			},
		},
	}

	updateFunc := func(d *ResourceData, m interface{}) error {
		d.Set("foo", "updated")
		if d.IsNewResource() {
			d.Set("foo", "new-resource")
		}
		return nil
	}
	r.Create = func(d *ResourceData, m interface{}) error {
		d.SetId("foo")
		d.Set("foo", "created")
		return updateFunc(d, m)
	}
	r.Update = updateFunc

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				New: "bla-blah",
			},
		},
	}

	// positive test
	var s *terraform.InstanceState = nil

	actual, err := r.Apply(s, d, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "new-resource",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual: %#v\nexpected: %#v",
			actual, expected)
	}

	// negative test
	s = &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "new-resource",
		},
	}

	actual, err = r.Apply(s, d, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected = &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "updated",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual: %#v\nexpected: %#v",
			actual, expected)
	}
}

func TestResourceInternalValidate(t *testing.T) {
	cases := []struct {
		In       *Resource
		Writable bool
		Err      bool
	}{
		0: {
			nil,
			true,
			true,
		},

		// No optional and no required
		1: {
			&Resource{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type:     TypeInt,
						Optional: true,
						Required: true,
					},
				},
			},
			true,
			true,
		},

		// Update undefined for non-ForceNew field
		2: {
			&Resource{
				Create: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"boo": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
			true,
			true,
		},

		// Update defined for ForceNew field
		3: {
			&Resource{
				Create: func(d *ResourceData, meta interface{}) error { return nil },
				Update: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"goo": &Schema{
						Type:     TypeInt,
						Optional: true,
						ForceNew: true,
					},
				},
			},
			true,
			true,
		},

		// non-writable doesn't need Update, Create or Delete
		4: {
			&Resource{
				Schema: map[string]*Schema{
					"goo": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
			false,
			false,
		},

		// non-writable *must not* have Create
		5: {
			&Resource{
				Create: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"goo": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
			false,
			true,
		},

		// writable must have Read
		6: {
			&Resource{
				Create: func(d *ResourceData, meta interface{}) error { return nil },
				Update: func(d *ResourceData, meta interface{}) error { return nil },
				Delete: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"goo": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
			true,
			true,
		},

		// writable must have Delete
		7: {
			&Resource{
				Create: func(d *ResourceData, meta interface{}) error { return nil },
				Read:   func(d *ResourceData, meta interface{}) error { return nil },
				Update: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"goo": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
			true,
			true,
		},

		8: { // Reserved name at root should be disallowed
			&Resource{
				Create: func(d *ResourceData, meta interface{}) error { return nil },
				Read:   func(d *ResourceData, meta interface{}) error { return nil },
				Update: func(d *ResourceData, meta interface{}) error { return nil },
				Delete: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"count": {
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
			true,
			true,
		},

		9: { // Reserved name at nested levels should be allowed
			&Resource{
				Create: func(d *ResourceData, meta interface{}) error { return nil },
				Read:   func(d *ResourceData, meta interface{}) error { return nil },
				Update: func(d *ResourceData, meta interface{}) error { return nil },
				Delete: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"parent_list": &Schema{
						Type:     TypeString,
						Optional: true,
						Elem: &Resource{
							Schema: map[string]*Schema{
								"provisioner": {
									Type:     TypeString,
									Optional: true,
								},
							},
						},
					},
				},
			},
			true,
			false,
		},

		10: { // Provider reserved name should be allowed in resource
			&Resource{
				Create: func(d *ResourceData, meta interface{}) error { return nil },
				Read:   func(d *ResourceData, meta interface{}) error { return nil },
				Update: func(d *ResourceData, meta interface{}) error { return nil },
				Delete: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"alias": &Schema{
						Type:     TypeString,
						Optional: true,
					},
				},
			},
			true,
			false,
		},

		11: { // ID should be allowed in data source
			&Resource{
				Read: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"id": &Schema{
						Type:     TypeString,
						Optional: true,
					},
				},
			},
			false,
			false,
		},

		12: { // Deprecated ID should be allowed in resource
			&Resource{
				Create: func(d *ResourceData, meta interface{}) error { return nil },
				Read:   func(d *ResourceData, meta interface{}) error { return nil },
				Update: func(d *ResourceData, meta interface{}) error { return nil },
				Delete: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"id": &Schema{
						Type:       TypeString,
						Optional:   true,
						Deprecated: "Use x_id instead",
					},
				},
			},
			true,
			false,
		},

		13: { // non-writable must not define CustomizeDiff
			&Resource{
				Read: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"goo": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},
				CustomizeDiff: func(*ResourceDiff, interface{}) error { return nil },
			},
			false,
			true,
		},
		14: { // Deprecated resource
			&Resource{
				Read: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"goo": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},
				DeprecationMessage: "This resource has been deprecated.",
			},
			true,
			true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			sm := schemaMap{}
			if tc.In != nil {
				sm = schemaMap(tc.In.Schema)
			}

			err := tc.In.InternalValidate(sm, tc.Writable)
			if err != nil && !tc.Err {
				t.Fatalf("%d: expected validation to pass: %s", i, err)
			}
			if err == nil && tc.Err {
				t.Fatalf("%d: expected validation to fail", i)
			}
		})
	}
}

func TestResourceRefresh(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		if m != 42 {
			return fmt.Errorf("meta not passed")
		}

		return d.Set("foo", d.Get("foo").(int)+1)
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	expected := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"id":  "bar",
			"foo": "13",
		},
		Meta: map[string]interface{}{
			"schema_version": "2",
		},
	}

	actual, err := r.Refresh(s, 42)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceRefresh_blankId(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		d.SetId("foo")
		return nil
	}

	s := &terraform.InstanceState{
		ID:         "",
		Attributes: map[string]string{},
	}

	actual, err := r.Refresh(s, 42)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceRefresh_delete(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		d.SetId("")
		return nil
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	actual, err := r.Refresh(s, 42)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceRefresh_existsError(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Exists = func(*ResourceData, interface{}) (bool, error) {
		return false, fmt.Errorf("error")
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		panic("shouldn't be called")
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	actual, err := r.Refresh(s, 42)
	if err == nil {
		t.Fatalf("should error")
	}
	if !reflect.DeepEqual(actual, s) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceRefresh_noExists(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Exists = func(*ResourceData, interface{}) (bool, error) {
		return false, nil
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		panic("shouldn't be called")
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	actual, err := r.Refresh(s, 42)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != nil {
		t.Fatalf("should have no state")
	}
}

func TestResourceRefresh_needsMigration(t *testing.T) {
	// Schema v2 it deals only in newfoo, which tracks foo as an int
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"newfoo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		return d.Set("newfoo", d.Get("newfoo").(int)+1)
	}

	r.MigrateState = func(
		v int,
		s *terraform.InstanceState,
		meta interface{}) (*terraform.InstanceState, error) {
		// Real state migration functions will probably switch on this value,
		// but we'll just assert on it for now.
		if v != 1 {
			t.Fatalf("Expected StateSchemaVersion to be 1, got %d", v)
		}

		if meta != 42 {
			t.Fatal("Expected meta to be passed through to the migration function")
		}

		oldfoo, err := strconv.ParseFloat(s.Attributes["oldfoo"], 64)
		if err != nil {
			t.Fatalf("err: %#v", err)
		}
		s.Attributes["newfoo"] = strconv.Itoa(int(oldfoo * 10))
		delete(s.Attributes, "oldfoo")

		return s, nil
	}

	// State is v1 and deals in oldfoo, which tracked foo as a float at 1/10th
	// the scale of newfoo
	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"oldfoo": "1.2",
		},
		Meta: map[string]interface{}{
			"schema_version": "1",
		},
	}

	actual, err := r.Refresh(s, 42)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"id":     "bar",
			"newfoo": "13",
		},
		Meta: map[string]interface{}{
			"schema_version": "2",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad:\n\nexpected: %#v\ngot: %#v", expected, actual)
	}
}

func TestResourceRefresh_noMigrationNeeded(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"newfoo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		return d.Set("newfoo", d.Get("newfoo").(int)+1)
	}

	r.MigrateState = func(
		v int,
		s *terraform.InstanceState,
		meta interface{}) (*terraform.InstanceState, error) {
		t.Fatal("Migrate function shouldn't be called!")
		return nil, nil
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"newfoo": "12",
		},
		Meta: map[string]interface{}{
			"schema_version": "2",
		},
	}

	actual, err := r.Refresh(s, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"id":     "bar",
			"newfoo": "13",
		},
		Meta: map[string]interface{}{
			"schema_version": "2",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad:\n\nexpected: %#v\ngot: %#v", expected, actual)
	}
}

func TestResourceRefresh_stateSchemaVersionUnset(t *testing.T) {
	r := &Resource{
		// Version 1 > Version 0
		SchemaVersion: 1,
		Schema: map[string]*Schema{
			"newfoo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		return d.Set("newfoo", d.Get("newfoo").(int)+1)
	}

	r.MigrateState = func(
		v int,
		s *terraform.InstanceState,
		meta interface{}) (*terraform.InstanceState, error) {
		s.Attributes["newfoo"] = s.Attributes["oldfoo"]
		return s, nil
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"oldfoo": "12",
		},
	}

	actual, err := r.Refresh(s, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"id":     "bar",
			"newfoo": "13",
		},
		Meta: map[string]interface{}{
			"schema_version": "1",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad:\n\nexpected: %#v\ngot: %#v", expected, actual)
	}
}

func TestResourceRefresh_migrateStateErr(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"newfoo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		t.Fatal("Read should never be called!")
		return nil
	}

	r.MigrateState = func(
		v int,
		s *terraform.InstanceState,
		meta interface{}) (*terraform.InstanceState, error) {
		return s, fmt.Errorf("triggering an error")
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"oldfoo": "12",
		},
	}

	_, err := r.Refresh(s, nil)
	if err == nil {
		t.Fatal("expected error, but got none!")
	}
}

func TestResourceData(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	state := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "42",
		},
	}

	data := r.Data(state)
	if data.Id() != "foo" {
		t.Fatalf("err: %s", data.Id())
	}
	if v := data.Get("foo"); v != 42 {
		t.Fatalf("bad: %#v", v)
	}

	// Set expectations
	state.Meta = map[string]interface{}{
		"schema_version": "2",
	}

	result := data.State()
	if !reflect.DeepEqual(result, state) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestResourceData_blank(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	data := r.Data(nil)
	if data.Id() != "" {
		t.Fatalf("err: %s", data.Id())
	}
	if v := data.Get("foo"); v != 0 {
		t.Fatalf("bad: %#v", v)
	}
}

func TestResourceData_timeouts(t *testing.T) {
	one := 1 * time.Second
	two := 2 * time.Second
	three := 3 * time.Second
	four := 4 * time.Second
	five := 5 * time.Second

	timeouts := &ResourceTimeout{
		Create:  &one,
		Read:    &two,
		Update:  &three,
		Delete:  &four,
		Default: &five,
	}

	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
		Timeouts: timeouts,
	}

	data := r.Data(nil)
	if data.Id() != "" {
		t.Fatalf("err: %s", data.Id())
	}

	if !reflect.DeepEqual(timeouts, data.timeouts) {
		t.Fatalf("incorrect ResourceData timeouts: %#v\n", *data.timeouts)
	}
}

func TestResource_UpgradeState(t *testing.T) {
	// While this really only calls itself and therefore doesn't test any of
	// the Resource code directly, it still serves as an example of registering
	// a StateUpgrader.
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"newfoo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.StateUpgraders = []StateUpgrader{
		{
			Version: 1,
			Type: cty.Object(map[string]cty.Type{
				"id":     cty.String,
				"oldfoo": cty.Number,
			}),
			Upgrade: func(m map[string]interface{}, meta interface{}) (map[string]interface{}, error) {

				oldfoo, ok := m["oldfoo"].(float64)
				if !ok {
					t.Fatalf("expected 1.2, got %#v", m["oldfoo"])
				}
				m["newfoo"] = int(oldfoo * 10)
				delete(m, "oldfoo")

				return m, nil
			},
		},
	}

	oldStateAttrs := map[string]string{
		"id":     "bar",
		"oldfoo": "1.2",
	}

	// convert the legacy flatmap state to the json equivalent
	ty := r.StateUpgraders[0].Type
	val, err := hcl2shim.HCL2ValueFromFlatmap(oldStateAttrs, ty)
	if err != nil {
		t.Fatal(err)
	}
	js, err := ctyjson.Marshal(val, ty)
	if err != nil {
		t.Fatal(err)
	}

	// unmarshal the state using the json default types
	var m map[string]interface{}
	if err := json.Unmarshal(js, &m); err != nil {
		t.Fatal(err)
	}

	actual, err := r.StateUpgraders[0].Upgrade(m, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := map[string]interface{}{
		"id":     "bar",
		"newfoo": 12,
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected: %#v\ngot: %#v\n", expected, actual)
	}
}

func TestResource_ValidateUpgradeState(t *testing.T) {
	r := &Resource{
		SchemaVersion: 3,
		Schema: map[string]*Schema{
			"newfoo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatal(err)
	}

	r.StateUpgraders = append(r.StateUpgraders, StateUpgrader{
		Version: 2,
		Type: cty.Object(map[string]cty.Type{
			"id": cty.String,
		}),
		Upgrade: func(m map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
			return m, nil
		},
	})
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatal(err)
	}

	// check for missing type
	r.StateUpgraders[0].Type = cty.Type{}
	if err := r.InternalValidate(nil, true); err == nil {
		t.Fatal("StateUpgrader must have type")
	}
	r.StateUpgraders[0].Type = cty.Object(map[string]cty.Type{
		"id": cty.String,
	})

	// check for missing Upgrade func
	r.StateUpgraders[0].Upgrade = nil
	if err := r.InternalValidate(nil, true); err == nil {
		t.Fatal("StateUpgrader must have an Upgrade func")
	}
	r.StateUpgraders[0].Upgrade = func(m map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
		return m, nil
	}

	// check for skipped version
	r.StateUpgraders[0].Version = 0
	r.StateUpgraders = append(r.StateUpgraders, StateUpgrader{
		Version: 2,
		Type: cty.Object(map[string]cty.Type{
			"id": cty.String,
		}),
		Upgrade: func(m map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
			return m, nil
		},
	})
	if err := r.InternalValidate(nil, true); err == nil {
		t.Fatal("StateUpgraders cannot skip versions")
	}

	// add the missing version
	// out of order upgraders should be OK
	r.StateUpgraders = append(r.StateUpgraders, StateUpgrader{
		Version: 1,
		Type: cty.Object(map[string]cty.Type{
			"id": cty.String,
		}),
		Upgrade: func(m map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
			return m, nil
		},
	})
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatal(err)
	}

	// can't add an upgrader for a schema >= the current version
	r.StateUpgraders = append(r.StateUpgraders, StateUpgrader{
		Version: 3,
		Type: cty.Object(map[string]cty.Type{
			"id": cty.String,
		}),
		Upgrade: func(m map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
			return m, nil
		},
	})
	if err := r.InternalValidate(nil, true); err == nil {
		t.Fatal("StateUpgraders cannot have a version >= current SchemaVersion")
	}
}
