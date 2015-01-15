package config

import (
	"encoding/gob"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config/lang/ast"
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
