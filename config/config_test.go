package config

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/helper/logging"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./testdata"

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Verbose() {
		// if we're verbose, use the logging requested by TF_LOG
		logging.SetOutput()
	} else {
		// otherwise silence all logs
		log.SetOutput(ioutil.Discard)
	}

	os.Exit(m.Run())
}

func TestConfigCopy(t *testing.T) {
	c := testConfig(t, "copy-basic")
	rOrig := c.Resources[0]
	rCopy := rOrig.Copy()

	if rCopy.Name != rOrig.Name {
		t.Fatalf("Expected names to equal: %q <=> %q", rCopy.Name, rOrig.Name)
	}

	if rCopy.Type != rOrig.Type {
		t.Fatalf("Expected types to equal: %q <=> %q", rCopy.Type, rOrig.Type)
	}

	origCount := rOrig.RawCount.Config()["count"]
	rCopy.RawCount.Config()["count"] = "5"
	if rOrig.RawCount.Config()["count"] != origCount {
		t.Fatalf("Expected RawCount to be copied, but it behaves like a ref!")
	}

	rCopy.RawConfig.Config()["newfield"] = "hello"
	if rOrig.RawConfig.Config()["newfield"] == "hello" {
		t.Fatalf("Expected RawConfig to be copied, but it behaves like a ref!")
	}

	rCopy.Provisioners = append(rCopy.Provisioners, &Provisioner{})
	if len(rOrig.Provisioners) == len(rCopy.Provisioners) {
		t.Fatalf("Expected Provisioners to be copied, but it behaves like a ref!")
	}

	if rCopy.Provider != rOrig.Provider {
		t.Fatalf("Expected providers to equal: %q <=> %q",
			rCopy.Provider, rOrig.Provider)
	}

	rCopy.DependsOn[0] = "gotchya"
	if rOrig.DependsOn[0] == rCopy.DependsOn[0] {
		t.Fatalf("Expected DependsOn to be copied, but it behaves like a ref!")
	}

	rCopy.Lifecycle.IgnoreChanges[0] = "gotchya"
	if rOrig.Lifecycle.IgnoreChanges[0] == rCopy.Lifecycle.IgnoreChanges[0] {
		t.Fatalf("Expected Lifecycle to be copied, but it behaves like a ref!")
	}

}

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

// Terraform GH-11800
func TestConfigCount_list(t *testing.T) {
	c := testConfig(t, "count-list")

	// The key is to interpolate so it doesn't fail parsing
	c.Resources[0].RawCount.Interpolate(map[string]ast.Variable{
		"var.list": ast.Variable{
			Value: []ast.Variable{},
			Type:  ast.TypeList,
		},
	})

	_, err := c.Resources[0].Count()
	if err == nil {
		t.Fatal("should error")
	}
}

func TestConfigCount_var(t *testing.T) {
	c := testConfig(t, "count-var")
	_, err := c.Resources[0].Count()
	if err == nil {
		t.Fatalf("should error")
	}
}

func TestConfig_emptyCollections(t *testing.T) {
	c := testConfig(t, "empty-collections")
	if len(c.Variables) != 3 {
		t.Fatalf("bad: expected 3 variables, got %d", len(c.Variables))
	}
	for _, variable := range c.Variables {
		switch variable.Name {
		case "empty_string":
			if variable.Default != "" {
				t.Fatalf("bad: wrong default %q for variable empty_string", variable.Default)
			}
		case "empty_map":
			if !reflect.DeepEqual(variable.Default, map[string]interface{}{}) {
				t.Fatalf("bad: wrong default %#v for variable empty_map", variable.Default)
			}
		case "empty_list":
			if !reflect.DeepEqual(variable.Default, []interface{}{}) {
				t.Fatalf("bad: wrong default %#v for variable empty_list", variable.Default)
			}
		default:
			t.Fatalf("Unexpected variable: %s", variable.Name)
		}
	}
}

// This table test is the preferred way to test validation of configuration.
// There are dozens of functions below which do not follow this that are
// there mostly historically. They should be converted at some point.
func TestConfigValidate_table(t *testing.T) {
	cases := []struct {
		Name      string
		Fixture   string
		Err       bool
		ErrString string
	}{
		{
			"basic good",
			"validate-good",
			false,
			"",
		},

		{
			"depends on module",
			"validate-depends-on-module",
			false,
			"",
		},

		{
			"depends on non-existent module",
			"validate-depends-on-bad-module",
			true,
			"non-existent module 'foo'",
		},

		{
			"data source with provisioners",
			"validate-data-provisioner",
			true,
			"data sources cannot have",
		},

		{
			"basic provisioners",
			"validate-basic-provisioners",
			false,
			"",
		},

		{
			"backend config with interpolations",
			"validate-backend-interpolate",
			true,
			"cannot contain interp",
		},
		{
			"nested types in variable default",
			"validate-var-nested",
			false,
			"",
		},
		{
			"provider with valid version constraint",
			"provider-version",
			false,
			"",
		},
		{
			"provider with invalid version constraint",
			"provider-version-invalid",
			true,
			"not a valid version constraint",
		},
		{
			"invalid provider name in module block",
			"validate-missing-provider",
			true,
			"cannot pass non-existent provider",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			c := testConfig(t, tc.Fixture)
			diags := c.Validate()
			if diags.HasErrors() != tc.Err {
				t.Fatalf("err: %s", diags.Err().Error())
			}
			if diags.HasErrors() {
				gotErr := diags.Err().Error()
				if tc.ErrString != "" && !strings.Contains(gotErr, tc.ErrString) {
					t.Fatalf("expected err to contain: %s\n\ngot: %s", tc.ErrString, gotErr)
				}

				return
			}
		})
	}

}

func TestConfigValidate_tfVersion(t *testing.T) {
	c := testConfig(t, "validate-tf-version")
	if err := c.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestConfigValidate_tfVersionBad(t *testing.T) {
	c := testConfig(t, "validate-bad-tf-version")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_tfVersionInterpolations(t *testing.T) {
	c := testConfig(t, "validate-tf-version-interp")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
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

func TestConfigValidate_countInt_HCL2(t *testing.T) {
	c := testConfigHCL2(t, "validate-count-int")
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestConfigValidate_countBadContext(t *testing.T) {
	c := testConfig(t, "validate-count-bad-context")

	diags := c.Validate()

	expected := []string{
		"output \"no_count_in_output\": count variables are only valid within resources",
		"module \"no_count_in_module\": count variables are only valid within resources",
	}
	for _, exp := range expected {
		errStr := diags.Err().Error()
		if !strings.Contains(errStr, exp) {
			t.Errorf("expected: %q,\nto contain: %q", errStr, exp)
		}
	}
}

func TestConfigValidate_countCountVar(t *testing.T) {
	c := testConfig(t, "validate-count-count-var")
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

func TestConfigValidate_countNotInt_HCL2(t *testing.T) {
	c := testConfigHCL2(t, "validate-count-not-int-const")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_countNotIntUnknown_HCL2(t *testing.T) {
	c := testConfigHCL2(t, "validate-count-not-int")
	// In HCL2 this is not an error because the unknown variable interpolates
	// to produce an unknown string, which we assume (incorrectly, it turns out)
	// will become a string containing only digits. This is okay because
	// the config validation is only a "best effort" and we'll get a definitive
	// result during the validation graph walk.
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestConfigValidate_countUserVar(t *testing.T) {
	c := testConfig(t, "validate-count-user-var")
	if err := c.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestConfigValidate_countUserVar_HCL2(t *testing.T) {
	c := testConfigHCL2(t, "validate-count-user-var")
	if err := c.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestConfigValidate_countLocalValue(t *testing.T) {
	c := testConfig(t, "validate-local-value-count")
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

func TestConfigValidate_countVarUnknown(t *testing.T) {
	c := testConfig(t, "validate-count-var-unknown")
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

func TestConfigValidate_ignoreChanges(t *testing.T) {
	c := testConfig(t, "validate-ignore-changes")
	if err := c.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestConfigValidate_ignoreChangesBad(t *testing.T) {
	c := testConfig(t, "validate-ignore-changes-bad")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_ignoreChangesInterpolate(t *testing.T) {
	c := testConfig(t, "validate-ignore-changes-interpolate")
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
	if err := c.Validate(); err != nil {
		t.Fatalf("should be valid: %s", err)
	}
}

func TestConfigValidate_moduleVarList(t *testing.T) {
	c := testConfig(t, "validate-module-var-list")
	if err := c.Validate(); err != nil {
		t.Fatalf("should be valid: %s", err)
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

func TestConfigValidate_outputDescription(t *testing.T) {
	c := testConfig(t, "validate-output-description")
	if err := c.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
	if len(c.Outputs) != 1 {
		t.Fatalf("got %d outputs; want 1", len(c.Outputs))
	}
	if got, want := "Number 5", c.Outputs[0].Description; got != want {
		t.Fatalf("got description %q; want %q", got, want)
	}
}

func TestConfigValidate_outputDuplicate(t *testing.T) {
	c := testConfig(t, "validate-output-dup")
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

func TestConfigValidate_varDefaultListType(t *testing.T) {
	c := testConfig(t, "validate-var-default-list-type")
	if err := c.Validate(); err != nil {
		t.Fatalf("should be valid: %s", err)
	}
}

func TestConfigValidate_varDefaultInterpolate(t *testing.T) {
	c := testConfig(t, "validate-var-default-interpolate")
	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestConfigValidate_varDefaultInterpolateEscaped(t *testing.T) {
	c := testConfig(t, "validate-var-default-interpolate-escaped")
	if err := c.Validate(); err != nil {
		t.Fatalf("should be valid, but got err: %s", err)
	}
}

func TestConfigValidate_varDup(t *testing.T) {
	c := testConfig(t, "validate-var-dup")
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

func TestConfigValidate_varProviderVersionInvalid(t *testing.T) {
	c := testConfig(t, "validate-provider-version-invalid")
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

func TestConfigValidate_localValuesMultiFile(t *testing.T) {
	c, err := LoadDir(filepath.Join(fixtureDir, "validate-local-multi-file"))
	if err != nil {
		t.Fatalf("unexpected error during load: %s", err)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error from validate: %s", err)
	}
	if len(c.Locals) != 1 {
		t.Fatalf("got 0 locals; want 1")
	}
	if got, want := c.Locals[0].Name, "test"; got != want {
		t.Errorf("wrong local name\ngot:  %#v\nwant: %#v", got, want)
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

func testConfig(t *testing.T, name string) *Config {
	c, err := LoadFile(filepath.Join(fixtureDir, name, "main.tf"))
	if err != nil {
		t.Fatalf("file: %s\n\nerr: %s", name, err)
	}

	return c
}

// testConfigHCL loads a config, forcing it to be processed with the HCL2
// loader even if it doesn't explicitly opt in to the HCL2 experiment.
func testConfigHCL2(t *testing.T, name string) *Config {
	t.Helper()
	cer, _, err := globalHCL2Loader.loadFile(filepath.Join(fixtureDir, name, "main.tf"))
	if err != nil {
		t.Fatalf("failed to load %s: %s", name, err)
	}

	cfg, err := cer.Config()
	if err != nil {
		t.Fatalf("failed to decode %s: %s", name, err)
	}

	return cfg
}

func TestConfigDataCount(t *testing.T) {
	c := testConfig(t, "data-count")
	actual, err := c.Resources[0].Count()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != 5 {
		t.Fatalf("bad: %#v", actual)
	}

	// we need to make sure "count" has been removed from the RawConfig, since
	// it's not a real key and won't validate.
	if _, ok := c.Resources[0].RawConfig.Raw["count"]; ok {
		t.Fatal("count key still exists in RawConfig")
	}
}

func TestConfigProviderVersion(t *testing.T) {
	c := testConfig(t, "provider-version")

	if len(c.ProviderConfigs) != 1 {
		t.Fatal("expected 1 provider")
	}

	p := c.ProviderConfigs[0]
	if p.Name != "aws" {
		t.Fatalf("expected provider name 'aws', got %q", p.Name)
	}

	if p.Version != "0.0.1" {
		t.Fatalf("expected providers version '0.0.1', got %q", p.Version)
	}

	if _, ok := p.RawConfig.Raw["version"]; ok {
		t.Fatal("'version' should not exist in raw config")
	}
}

func TestResourceProviderFullName(t *testing.T) {
	type testCase struct {
		ResourceName string
		Alias        string
		Expected     string
	}

	tests := []testCase{
		{
			// If no alias is provided, the first underscore-separated segment
			// is assumed to be the provider name.
			ResourceName: "aws_thing",
			Alias:        "",
			Expected:     "aws",
		},
		{
			// If we have more than one underscore then it's the first one that we'll use.
			ResourceName: "aws_thingy_thing",
			Alias:        "",
			Expected:     "aws",
		},
		{
			// A provider can export a resource whose name is just the bare provider name,
			// e.g. because the provider only has one resource and so any additional
			// parts would be redundant.
			ResourceName: "external",
			Alias:        "",
			Expected:     "external",
		},
		{
			// Alias always overrides the default extraction of the name
			ResourceName: "aws_thing",
			Alias:        "tls.baz",
			Expected:     "tls.baz",
		},
	}

	for _, test := range tests {
		got := ResourceProviderFullName(test.ResourceName, test.Alias)
		if got != test.Expected {
			t.Errorf(
				"(%q, %q) produced %q; want %q",
				test.ResourceName, test.Alias,
				got,
				test.Expected,
			)
		}
	}
}

func TestConfigModuleProviders(t *testing.T) {
	c := testConfig(t, "module-providers")

	if len(c.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(c.Modules))
	}

	expected := map[string]string{
		"aws": "aws.foo",
	}

	got := c.Modules[0].Providers

	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("exptected providers %#v, got providers %#v", expected, got)
	}
}

func TestValidateOutputErrorWarnings(t *testing.T) {
	// TODO: remove this in 0.12
	c := testConfig(t, "output-warnings")

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatal("config should not have errors:", diags)
	}
	if len(diags) != 2 {
		t.Fatalf("should have 2 warnings, got %d:\n%s", len(diags), diags)
	}

	// this fixture has no explicit count, and should have no warning
	c = testConfig(t, "output-no-warnings")
	if err := c.Validate(); err != nil {
		t.Fatal("config should have no warnings or errors")
	}
}
