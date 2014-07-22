package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

func TestConfigValidate(t *testing.T) {
	c := testConfig(t, "validate-good")
	if err := c.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestConfigValidate_badMultiResource(t *testing.T) {
	c := testConfig(t, "validate-bad-multi-resource")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_dupResource(t *testing.T) {
	c := testConfig(t, "validate-dup-resource")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_outputBadField(t *testing.T) {
	c := testConfig(t, "validate-output-bad-field")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_unknownThing(t *testing.T) {
	c := testConfig(t, "validate-unknownthing")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_unknownResourceVar(t *testing.T) {
	c := testConfig(t, "validate-unknown-resource-var")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_unknownResourceVar_output(t *testing.T) {
	c := testConfig(t, "validate-unknown-resource-var-output")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_unknownVar(t *testing.T) {
	c := testConfig(t, "validate-unknownvar")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_varDefault(t *testing.T) {
	c := testConfig(t, "validate-var-default")
	if err := c.Validate(); err != nil {
		t.Fatalf("should be valid: %s", err)
	}
}

func TestConfigValidate_varDefaultBadType(t *testing.T) {
	c := testConfig(t, "validate-var-default-bad-type")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestProviderConfigName(t *testing.T) {
	pcs := []*ProviderConfig{
		&ProviderConfig{Name: "aw"},
		&ProviderConfig{Name: "aws"},
		&ProviderConfig{Name: "a"},
		&ProviderConfig{Name: "gce_"},
	}

	n := ProviderConfigName("aws_instance", pcs)
	if n != "aws" {
		t.Fatalf("bad: %s", n)
	}
}

func TestVariableDefaultsMap(t *testing.T) {
	cases := []struct {
		Default interface{}
		Output  map[string]string
	}{
		{
			nil,
			nil,
		},

		{
			"foo",
			map[string]string{"var.foo": "foo"},
		},

		{
			map[interface{}]interface{}{
				"foo": "bar",
				"bar": "baz",
			},
			map[string]string{
				"var.foo":     "foo",
				"var.foo.foo": "bar",
				"var.foo.bar": "baz",
			},
		},
	}

	for i, tc := range cases {
		v := &Variable{Name: "foo", Default: tc.Default}
		actual := v.DefaultsMap()
		if !reflect.DeepEqual(actual, tc.Output) {
			t.Fatalf("%d: bad: %#v", i, actual)
		}
	}
}

func testConfig(t *testing.T, name string) *Config {
	c, err := Load(filepath.Join(fixtureDir, name, "main.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return c
}
