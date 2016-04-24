package config

import (
	"encoding/gob"
	"reflect"
	"testing"

	"github.com/hashicorp/hil/ast"
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

func TestRawConfig(t *testing.T) {
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
				Value: UnknownVariableValue,
				Type:  ast.TypeString,
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
			Value: UnknownVariableValue,
			Type:  ast.TypeString,
		},
	}
	if err := rc.Interpolate(vars); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := rc.Config()
	expected := map[string]interface{}{}

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
			Value: UnknownVariableValue,
			Type:  ast.TypeString,
		},
	}
	if err := rc.Interpolate(vars); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := rc.Config()
	expected := map[string]interface{}{}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	expectedKeys := []string{"foo"}
	if !reflect.DeepEqual(rc.UnknownKeys(), expectedKeys) {
		t.Fatalf("bad: %#v", rc.UnknownKeys())
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
