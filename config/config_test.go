package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

func TestConfigCount(t *testing.T) {
	c := testConfig(t, "count-int")
	actual, err := c.Resources[0].Count()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != 5 {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestConfigCount_string(t *testing.T) {
	c := testConfig(t, "count-string")
	actual, err := c.Resources[0].Count()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != 5 {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestConfigCount_var(t *testing.T) {
	c := testConfig(t, "count-var")
	_, err := c.Resources[0].Count()
	if err == nil {
		t.Fatalf("should error")
	}
}

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

func TestConfigValidate_countInt(t *testing.T) {
	c := testConfig(t, "validate-count-int")
	if err := c.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestConfigValidate_countBadContext(t *testing.T) {
	c := testConfig(t, "validate-count-bad-context")

	err := c.Validate()

	expected := []string{
		"no_count_in_output: count variables are only valid within resources",
		"no_count_in_module: count variables are only valid within resources",
	}
	for _, exp := range expected {
		if !strings.Contains(err.Error(), exp) {
			t.Fatalf("expected: %q,\nto contain: %q", err, exp)
		}
	}
}

func TestConfigValidate_countCountVar(t *testing.T) {
	c := testConfig(t, "validate-count-count-var")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_countModuleVar(t *testing.T) {
	c := testConfig(t, "validate-count-module-var")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_countNotInt(t *testing.T) {
	c := testConfig(t, "validate-count-not-int")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_countResourceVar(t *testing.T) {
	c := testConfig(t, "validate-count-resource-var")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_countUserVar(t *testing.T) {
	c := testConfig(t, "validate-count-user-var")
	if err := c.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestConfigValidate_countVar(t *testing.T) {
	c := testConfig(t, "validate-count-var")
	if err := c.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestConfigValidate_countVarInvalid(t *testing.T) {
	c := testConfig(t, "validate-count-var-invalid")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_dependsOnVar(t *testing.T) {
	c := testConfig(t, "validate-depends-on-var")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_dupModule(t *testing.T) {
	c := testConfig(t, "validate-dup-module")
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

func TestConfigValidate_moduleNameBad(t *testing.T) {
	c := testConfig(t, "validate-module-name-bad")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_moduleSourceVar(t *testing.T) {
	c := testConfig(t, "validate-module-source-var")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_moduleVarInt(t *testing.T) {
	c := testConfig(t, "validate-module-var-int")
	if err := c.Validate(); err != nil {
		t.Fatalf("should be valid: %s", err)
	}
}

func TestConfigValidate_moduleVarMap(t *testing.T) {
	c := testConfig(t, "validate-module-var-map")
	if err := c.Validate(); err == nil {
		t.Fatal("should be invalid")
	}
}

func TestConfigValidate_moduleVarSelf(t *testing.T) {
	c := testConfig(t, "validate-module-var-self")
	if err := c.Validate(); err == nil {
		t.Fatal("should be invalid")
	}
}

func TestConfigValidate_nil(t *testing.T) {
	var c Config
	if err := c.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestConfigValidate_outputBadField(t *testing.T) {
	c := testConfig(t, "validate-output-bad-field")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_pathVar(t *testing.T) {
	c := testConfig(t, "validate-path-var")
	if err := c.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestConfigValidate_pathVarInvalid(t *testing.T) {
	c := testConfig(t, "validate-path-var-invalid")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_providerMulti(t *testing.T) {
	c := testConfig(t, "validate-provider-multi")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_providerMultiGood(t *testing.T) {
	c := testConfig(t, "validate-provider-multi-good")
	if err := c.Validate(); err != nil {
		t.Fatalf("should be valid: %s", err)
	}
}

func TestConfigValidate_providerMultiRefGood(t *testing.T) {
	c := testConfig(t, "validate-provider-multi-ref-good")
	if err := c.Validate(); err != nil {
		t.Fatalf("should be valid: %s", err)
	}
}

func TestConfigValidate_providerMultiRefBad(t *testing.T) {
	c := testConfig(t, "validate-provider-multi-ref-bad")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_provConnSplatOther(t *testing.T) {
	c := testConfig(t, "validate-prov-conn-splat-other")
	if err := c.Validate(); err != nil {
		t.Fatalf("should be valid: %s", err)
	}
}

func TestConfigValidate_provConnSplatSelf(t *testing.T) {
	c := testConfig(t, "validate-prov-conn-splat-self")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_provSplatOther(t *testing.T) {
	c := testConfig(t, "validate-prov-splat-other")
	if err := c.Validate(); err != nil {
		t.Fatalf("should be valid: %s", err)
	}
}

func TestConfigValidate_provSplatSelf(t *testing.T) {
	c := testConfig(t, "validate-prov-splat-self")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_resourceProvVarSelf(t *testing.T) {
	c := testConfig(t, "validate-resource-prov-self")
	if err := c.Validate(); err != nil {
		t.Fatalf("should be valid: %s", err)
	}
}

func TestConfigValidate_resourceVarSelf(t *testing.T) {
	c := testConfig(t, "validate-resource-self")
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

func TestConfigValidate_unknownVarCount(t *testing.T) {
	c := testConfig(t, "validate-unknownvar-count")
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

func TestConfigValidate_varMultiExactNonSlice(t *testing.T) {
	c := testConfig(t, "validate-var-multi-exact-non-slice")
	if err := c.Validate(); err != nil {
		t.Fatalf("should be valid: %s", err)
	}
}

func TestConfigValidate_varMultiNonSlice(t *testing.T) {
	c := testConfig(t, "validate-var-multi-non-slice")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_varMultiNonSliceProvisioner(t *testing.T) {
	c := testConfig(t, "validate-var-multi-non-slice-provisioner")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_varMultiFunctionCall(t *testing.T) {
	c := testConfig(t, "validate-var-multi-func")
	if err := c.Validate(); err != nil {
		t.Fatalf("should be valid: %s", err)
	}
}

func TestConfigValidate_varModule(t *testing.T) {
	c := testConfig(t, "validate-var-module")
	if err := c.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestConfigValidate_varModuleInvalid(t *testing.T) {
	c := testConfig(t, "validate-var-module-invalid")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestNameRegexp(t *testing.T) {
	cases := []struct {
		Input string
		Match bool
	}{
		{"hello", true},
		{"foo-bar", true},
		{"foo_bar", true},
		{"_hello", true},
		{"foo bar", false},
		{"foo.bar", false},
	}

	for _, tc := range cases {
		if NameRegexp.Match([]byte(tc.Input)) != tc.Match {
			t.Fatalf("Input: %s\n\nExpected: %#v", tc.Input, tc.Match)
		}
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
	c, err := LoadFile(filepath.Join(fixtureDir, name, "main.tf"))
	if err != nil {
		t.Fatalf("file: %s\n\nerr: %s", name, err)
	}

	return c
}
