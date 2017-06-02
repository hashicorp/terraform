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
const fixtureDir = "./test-fixtures"

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
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			c := testConfig(t, tc.Fixture)
			err := c.Validate()
			if (err != nil) != tc.Err {
				t.Fatalf("err: %s", err)
			}
			if err != nil {
				if tc.ErrString != "" && !strings.Contains(err.Error(), tc.ErrString) {
					t.Fatalf("expected err to contain: %s\n\ngot: %s", tc.ErrString, err)
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

func TestConfigValidate_countNotInt(t *testing.T) {
	c := testConfig(t, "validate-count-not-int")
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

func testConfig(t *testing.T, name string) *Config {
	c, err := LoadFile(filepath.Join(fixtureDir, name, "main.tf"))
	if err != nil {
		t.Fatalf("file: %s\n\nerr: %s", name, err)
	}

	return c
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
