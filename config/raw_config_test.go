package config

import (
	"reflect"
	"testing"
)

func TestNewRawConfig(t *testing.T) {
	raw := map[string]interface{}{
		"foo": "${var.bar}",
	}

	rc, err := NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
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

	vars := map[string]string{"var.bar": "baz"}
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

func TestRawConfig_unknown(t *testing.T) {
	raw := map[string]interface{}{
		"foo": "${var.bar}",
	}

	rc, err := NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	vars := map[string]string{"var.bar": UnknownVariableValue}
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
