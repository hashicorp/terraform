package config

import (
	"encoding/gob"
	"reflect"
	"testing"

	hcl2 "github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/config/hcl2shim"
)

func TestNewRawConfig(t *testing.T) {
	raw := map[string]interface{}{
		"foo": "${var.bar}",
		"bar": `${file("boom.txt")}`,
	}

	rc, err := NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(rc.Interpolations) != 2 {
		t.Fatalf("bad: %#v", rc.Interpolations)
	}
	if len(rc.Variables) != 1 {
		t.Fatalf("bad: %#v", rc.Variables)
	}
}

func TestRawConfig_basic(t *testing.T) {
	raw := map[string]interface{}{
		"foo": "${var.bar}",
	}

	rc, err := NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Before interpolate, Config() should be the raw
	if !reflect.DeepEqual(rc.Config(), raw) {
		t.Fatalf("bad: %#v", rc.Config())
	}

	vars := map[string]ast.Variable{
		"var.bar": ast.Variable{
			Value: "baz",
			Type:  ast.TypeString,
		},
	}
	if err := rc.Interpolate(vars); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := rc.Config()
	expected := map[string]interface{}{
		"foo": "baz",
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
	if len(rc.UnknownKeys()) != 0 {
		t.Fatalf("bad: %#v", rc.UnknownKeys())
	}
}

func TestRawConfig_double(t *testing.T) {
	raw := map[string]interface{}{
		"foo": "${var.bar}",
	}

	rc, err := NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	vars := map[string]ast.Variable{
		"var.bar": ast.Variable{
			Value: "baz",
			Type:  ast.TypeString,
		},
	}
	if err := rc.Interpolate(vars); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := rc.Config()
	expected := map[string]interface{}{
		"foo": "baz",
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	vars = map[string]ast.Variable{
		"var.bar": ast.Variable{
			Value: "what",
			Type:  ast.TypeString,
		},
	}
	if err := rc.Interpolate(vars); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual = rc.Config()
	expected = map[string]interface{}{
		"foo": "what",
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestRawConfigInterpolate_escaped(t *testing.T) {
	raw := map[string]interface{}{
		"foo": "bar-$${baz}",
	}

	rc, err := NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Before interpolate, Config() should be the raw
	if !reflect.DeepEqual(rc.Config(), raw) {
		t.Fatalf("bad: %#v", rc.Config())
	}

	if err := rc.Interpolate(nil); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := rc.Config()
	expected := map[string]interface{}{
		"foo": "bar-${baz}",
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
	if len(rc.UnknownKeys()) != 0 {
		t.Fatalf("bad: %#v", rc.UnknownKeys())
	}
}

func TestRawConfig_merge(t *testing.T) {
	raw1 := map[string]interface{}{
		"foo": "${var.foo}",
		"bar": "${var.bar}",
	}

	rc1, err := NewRawConfig(raw1)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	{
		vars := map[string]ast.Variable{
			"var.foo": ast.Variable{
				Value: "foovalue",
				Type:  ast.TypeString,
			},
			"var.bar": ast.Variable{
				Value: "nope",
				Type:  ast.TypeString,
			},
		}
		if err := rc1.Interpolate(vars); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	raw2 := map[string]interface{}{
		"bar": "${var.bar}",
		"baz": "${var.baz}",
	}

	rc2, err := NewRawConfig(raw2)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	{
		vars := map[string]ast.Variable{
			"var.bar": ast.Variable{
				Value: "barvalue",
				Type:  ast.TypeString,
			},
			"var.baz": ast.Variable{
				Value: hcl2shim.UnknownVariableValue,
				Type:  ast.TypeUnknown,
			},
		}
		if err := rc2.Interpolate(vars); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	// Merge the two
	rc3 := rc1.Merge(rc2)

	// Raw should be merged
	raw3 := map[string]interface{}{
		"foo": "${var.foo}",
		"bar": "${var.bar}",
		"baz": "${var.baz}",
	}
	if !reflect.DeepEqual(rc3.Raw, raw3) {
		t.Fatalf("bad: %#v", rc3.Raw)
	}

	actual := rc3.Config()
	expected := map[string]interface{}{
		"foo": "foovalue",
		"bar": "barvalue",
		"baz": hcl2shim.UnknownVariableValue,
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	expectedKeys := []string{"baz"}
	if !reflect.DeepEqual(rc3.UnknownKeys(), expectedKeys) {
		t.Fatalf("bad: %#v", rc3.UnknownKeys())
	}
}

func TestRawConfig_syntax(t *testing.T) {
	raw := map[string]interface{}{
		"foo": "${var",
	}

	if _, err := NewRawConfig(raw); err == nil {
		t.Fatal("should error")
	}
}

func TestRawConfig_unknown(t *testing.T) {
	raw := map[string]interface{}{
		"foo": "${var.bar}",
	}

	rc, err := NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	vars := map[string]ast.Variable{
		"var.bar": ast.Variable{
			Value: hcl2shim.UnknownVariableValue,
			Type:  ast.TypeUnknown,
		},
	}
	if err := rc.Interpolate(vars); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := rc.Config()
	expected := map[string]interface{}{"foo": hcl2shim.UnknownVariableValue}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	expectedKeys := []string{"foo"}
	if !reflect.DeepEqual(rc.UnknownKeys(), expectedKeys) {
		t.Fatalf("bad: %#v", rc.UnknownKeys())
	}
}

func TestRawConfig_unknownPartial(t *testing.T) {
	raw := map[string]interface{}{
		"foo": "${var.bar}/32",
	}

	rc, err := NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	vars := map[string]ast.Variable{
		"var.bar": ast.Variable{
			Value: hcl2shim.UnknownVariableValue,
			Type:  ast.TypeUnknown,
		},
	}
	if err := rc.Interpolate(vars); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := rc.Config()
	expected := map[string]interface{}{"foo": hcl2shim.UnknownVariableValue}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	expectedKeys := []string{"foo"}
	if !reflect.DeepEqual(rc.UnknownKeys(), expectedKeys) {
		t.Fatalf("bad: %#v", rc.UnknownKeys())
	}
}

func TestRawConfig_unknownPartialList(t *testing.T) {
	raw := map[string]interface{}{
		"foo": []interface{}{
			"${var.bar}/32",
		},
	}

	rc, err := NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	vars := map[string]ast.Variable{
		"var.bar": ast.Variable{
			Value: hcl2shim.UnknownVariableValue,
			Type:  ast.TypeUnknown,
		},
	}
	if err := rc.Interpolate(vars); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := rc.Config()
	expected := map[string]interface{}{"foo": []interface{}{hcl2shim.UnknownVariableValue}}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	expectedKeys := []string{"foo"}
	if !reflect.DeepEqual(rc.UnknownKeys(), expectedKeys) {
		t.Fatalf("bad: %#v", rc.UnknownKeys())
	}
}

// This tests a race found where we were not maintaining the "slice index"
// accounting properly. The result would be that some computed keys would
// look like they had no slice index when they in fact do. This test is not
// very reliable but it did fail before the fix and passed after.
func TestRawConfig_sliceIndexLoss(t *testing.T) {
	raw := map[string]interface{}{
		"slice": []map[string]interface{}{
			map[string]interface{}{
				"foo": []interface{}{"foo/${var.unknown}"},
				"bar": []interface{}{"bar"},
			},
		},
	}

	vars := map[string]ast.Variable{
		"var.unknown": ast.Variable{
			Value: hcl2shim.UnknownVariableValue,
			Type:  ast.TypeUnknown,
		},
		"var.known": ast.Variable{
			Value: "123456",
			Type:  ast.TypeString,
		},
	}

	// We run it a lot because its fast and we try to get a race out
	for i := 0; i < 50; i++ {
		rc, err := NewRawConfig(raw)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if err := rc.Interpolate(vars); err != nil {
			t.Fatalf("err: %s", err)
		}

		expectedKeys := []string{"slice.0.foo"}
		if !reflect.DeepEqual(rc.UnknownKeys(), expectedKeys) {
			t.Fatalf("bad: %#v", rc.UnknownKeys())
		}
	}
}

func TestRawConfigCopy(t *testing.T) {
	raw := map[string]interface{}{
		"foo": "${var.bar}",
	}

	rc, err := NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	rc.Key = "foo"
	if rc.Value() != "${var.bar}" {
		t.Fatalf("err: %#v", rc.Value())
	}

	// Interpolate the first one
	vars := map[string]ast.Variable{
		"var.bar": ast.Variable{
			Value: "baz",
			Type:  ast.TypeString,
		},
	}
	if err := rc.Interpolate(vars); err != nil {
		t.Fatalf("err: %s", err)
	}

	if rc.Value() != "baz" {
		t.Fatalf("bad: %#v", rc.Value())
	}

	// Copy and interpolate
	{
		rc2 := rc.Copy()
		if rc2.Value() != "${var.bar}" {
			t.Fatalf("err: %#v", rc2.Value())
		}

		vars := map[string]ast.Variable{
			"var.bar": ast.Variable{
				Value: "qux",
				Type:  ast.TypeString,
			},
		}
		if err := rc2.Interpolate(vars); err != nil {
			t.Fatalf("err: %s", err)
		}

		if rc2.Value() != "qux" {
			t.Fatalf("bad: %#v", rc2.Value())
		}
	}
}

func TestRawConfigCopyHCL2(t *testing.T) {
	rc := NewRawConfigHCL2(hcl2.EmptyBody())
	rc2 := rc.Copy()

	if rc.Body == nil {
		t.Errorf("RawConfig copy has a nil Body")
	}
	if rc2.Raw != nil {
		t.Errorf("RawConfig copy got a non-nil Raw")
	}
}

func TestRawConfigValue(t *testing.T) {
	raw := map[string]interface{}{
		"foo": "${var.bar}",
	}

	rc, err := NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	rc.Key = ""
	if rc.Value() != nil {
		t.Fatalf("bad: %#v", rc.Value())
	}

	rc.Key = "foo"
	if rc.Value() != "${var.bar}" {
		t.Fatalf("err: %#v", rc.Value())
	}

	vars := map[string]ast.Variable{
		"var.bar": ast.Variable{
			Value: "baz",
			Type:  ast.TypeString,
		},
	}
	if err := rc.Interpolate(vars); err != nil {
		t.Fatalf("err: %s", err)
	}

	if rc.Value() != "baz" {
		t.Fatalf("bad: %#v", rc.Value())
	}
}

func TestRawConfig_implGob(t *testing.T) {
	var _ gob.GobDecoder = new(RawConfig)
	var _ gob.GobEncoder = new(RawConfig)
}

// verify that RawMap returns an identical copy
func TestNewRawConfig_rawMap(t *testing.T) {
	raw := map[string]interface{}{
		"foo": "${var.bar}",
		"bar": `${file("boom.txt")}`,
	}

	rc, err := NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	rawCopy := rc.RawMap()
	if !reflect.DeepEqual(raw, rawCopy) {
		t.Fatalf("bad: %#v", rawCopy)
	}

	// make sure they aren't the same map
	raw["test"] = "value"
	if reflect.DeepEqual(raw, rawCopy) {
		t.Fatal("RawMap() didn't return a copy")
	}
}
