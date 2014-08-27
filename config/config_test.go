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

func TestConfigValidate_badDependsOn(t *testing.T) {
	c := testConfig(t, "validate-bad-depends-on")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_badMultiResource(t *testing.T) {
	c := testConfig(t, "validate-bad-multi-resource")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_countBelowZero(t *testing.T) {
	c := testConfig(t, "validate-count-below-zero")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_countZero(t *testing.T) {
	c := testConfig(t, "validate-count-zero")
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

func TestConfigValidate_varDefaultInterpolate(t *testing.T) {
	c := testConfig(t, "validate-var-default-interpolate")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_resourceTemplate(t *testing.T) {
	c := testConfig(t, "validate-resource-template")
	if err := c.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestConfigValidate_resourceTemplateBadName(t *testing.T) {
	c := testConfig(t, "validate-resource-template-bad-name")
	if err := c.Validate(); err == nil {
		t.Fatalf("should not be valid")
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

func TestResourceTemplate_Apply(t *testing.T) {
	provisioners := []*Provisioner{
		&Provisioner{
			Type: "ssh",
			ConnInfo: &RawConfig{
				Raw: map[string]interface{}{
					"user": "test",
				},
			},
		},
	}

	resourceTemplate := &ResourceTemplate{
		Name:         "test-template",
		Count:        3,
		Provisioners: provisioners,
		DependsOn:    []string{"aws_instance.web"},
		RawConfig: &RawConfig{
			Raw: map[string]interface{}{
				"keyA": "bad",
				"keyC": []string{"good"},
			},
		},
	}

	resource1 := &Resource{
		RawConfig: &RawConfig{
			Raw: map[string]interface{}{
				"keyA": "good",
				"keyB": "good",
			},
		},
	}

	resource2 := &Resource{
		Count:        2,
		countSet:     true,
		Provisioners: []*Provisioner{&Provisioner{}},
		DependsOn:    []string{"aws_instance.db"},
		RawConfig: &RawConfig{
			Raw: map[string]interface{}{},
		},
	}

	resource1.ApplyTemplate(resourceTemplate)
	resource2.ApplyTemplate(resourceTemplate)

	cases := [][]interface{}{
		// Config items should be properly merged where appropriate
		[]interface{}{resource1.RawConfig.Raw["keyA"], "good"},
		[]interface{}{resource1.RawConfig.Raw["keyB"], "good"},
		[]interface{}{resource1.RawConfig.Raw["keyC"], []string{"good"}},
		[]interface{}{resource1.Count, resourceTemplate.Count},
		[]interface{}{resource1.Provisioners, resourceTemplate.Provisioners},
		[]interface{}{resource1.DependsOn, resourceTemplate.DependsOn},

		// Config items which are present within the resource definition should
		// preserved after template application
		[]interface{}{resource2.Count, 2},
		[]interface{}{resource2.Provisioners, []*Provisioner{&Provisioner{}}},
		[]interface{}{resource2.DependsOn, []string{"aws_instance.db"}},
	}
	for _, c := range cases {
		if !reflect.DeepEqual(c[0], c[1]) {
			t.Fatalf("bad:\n%#v", c[0])
		}
	}
}

func testConfig(t *testing.T, name string) *Config {
	c, err := Load(filepath.Join(fixtureDir, name, "main.tf"))
	if err != nil {
		t.Fatalf("file: %s\n\nerr: %s", name, err)
	}

	return c
}
