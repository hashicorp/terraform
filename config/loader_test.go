package config

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestErrNoConfigsFound_impl(t *testing.T) {
	var _ error = new(ErrNoConfigsFound)
}

func TestIsEmptyDir(t *testing.T) {
	val, err := IsEmptyDir(fixtureDir)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if val {
		t.Fatal("should not be empty")
	}
}

func TestIsEmptyDir_noExist(t *testing.T) {
	val, err := IsEmptyDir(filepath.Join(fixtureDir, "nopenopenope"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !val {
		t.Fatal("should be empty")
	}
}

func TestIsEmptyDir_noConfigs(t *testing.T) {
	val, err := IsEmptyDir(filepath.Join(fixtureDir, "dir-empty"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !val {
		t.Fatal("should be empty")
	}
}

func TestLoadFile_badType(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "bad_type.tf.nope"))
	if err == nil {
		t.Fatal("should have error")
	}
}

func TestLoadFile_gitCrypt(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "git-crypt.tf"))
	if err == nil {
		t.Fatal("should have error")
	}

	t.Logf("err: %s", err)
}

func TestLoadFile_lifecycleKeyCheck(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "lifecycle_cbd_typo.tf"))
	if err == nil {
		t.Fatal("should have error")
	}

	t.Logf("err: %s", err)
}

func TestLoadFile_varInvalidKey(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "var-invalid-key.tf"))
	if err == nil {
		t.Fatal("should have error")
	}
}

func TestLoadFile_resourceArityMistake(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "resource-arity-mistake.tf"))
	if err == nil {
		t.Fatal("should have error")
	}
	expected := "Error loading test-fixtures/resource-arity-mistake.tf: position 2:10: resource must be followed by exactly two strings, a type and a name"
	if err.Error() != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, err)
	}
}

func TestLoadFile_resourceMultiLifecycle(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "resource-multi-lifecycle.tf"))
	if err == nil {
		t.Fatal("should have error")
	}
}

func TestLoadFile_dataSourceArityMistake(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "data-source-arity-mistake.tf"))
	if err == nil {
		t.Fatal("should have error")
	}
	expected := "Error loading test-fixtures/data-source-arity-mistake.tf: position 2:6: 'data' must be followed by exactly two strings: a type and a name"
	if err.Error() != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, err)
	}
}

func TestLoadFileWindowsLineEndings(t *testing.T) {
	testFile := filepath.Join(fixtureDir, "windows-line-endings.tf")

	contents, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(string(contents), "\r\n") {
		t.Fatalf("Windows line endings test file %s contains no windows line endings - this may be an autocrlf related issue.", testFile)
	}

	c, err := LoadFile(testFile)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	if c.Dir != "" {
		t.Fatalf("bad: %#v", c.Dir)
	}

	actual := resourcesStr(c.Resources)
	if actual != strings.TrimSpace(windowsHeredocResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadFileHeredoc(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "heredoc.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	if c.Dir != "" {
		t.Fatalf("bad: %#v", c.Dir)
	}

	actual := providerConfigsStr(c.ProviderConfigs)
	if actual != strings.TrimSpace(heredocProvidersStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = resourcesStr(c.Resources)
	if actual != strings.TrimSpace(heredocResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadFileEscapedQuotes(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "escapedquotes.tf"))
	if err == nil {
		t.Fatalf("expected syntax error as escaped quotes are no longer supported")
	}

	if !strings.Contains(err.Error(), "parse error") {
		t.Fatalf("expected \"syntax error\", got: %s", err)
	}
}

func TestLoadFileBasic(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "basic.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	if c.Dir != "" {
		t.Fatalf("wrong dir %#v; want %#v", c.Dir, "")
	}

	expectedTF := &Terraform{RequiredVersion: "foo"}
	if !reflect.DeepEqual(c.Terraform, expectedTF) {
		t.Fatalf("wrong terraform block %#v; want %#v", c.Terraform, expectedTF)
	}

	expectedAtlas := &AtlasConfig{Name: "mitchellh/foo"}
	if !reflect.DeepEqual(c.Atlas, expectedAtlas) {
		t.Fatalf("wrong atlas config %#v; want %#v", c.Atlas, expectedAtlas)
	}

	actual := variablesStr(c.Variables)
	if actual != strings.TrimSpace(basicVariablesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = providerConfigsStr(c.ProviderConfigs)
	if actual != strings.TrimSpace(basicProvidersStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = resourcesStr(c.Resources)
	if actual != strings.TrimSpace(basicResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	if actual, want := localsStr(c.Locals), strings.TrimSpace(basicLocalsStr); actual != want {
		t.Fatalf("wrong locals:\n%s\nwant:\n%s", actual, want)
	}

	actual = outputsStr(c.Outputs)
	if actual != strings.TrimSpace(basicOutputsStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadFileBasic_empty(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "empty.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}
}

func TestLoadFileBasic_import(t *testing.T) {
	// Skip because we disabled importing
	t.Skip()

	c, err := LoadFile(filepath.Join(fixtureDir, "import.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	actual := variablesStr(c.Variables)
	if actual != strings.TrimSpace(importVariablesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = providerConfigsStr(c.ProviderConfigs)
	if actual != strings.TrimSpace(importProvidersStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = resourcesStr(c.Resources)
	if actual != strings.TrimSpace(importResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadFileBasic_json(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "basic.tf.json"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	if c.Dir != "" {
		t.Fatalf("bad: %#v", c.Dir)
	}

	expectedAtlas := &AtlasConfig{Name: "mitchellh/foo"}
	if !reflect.DeepEqual(c.Atlas, expectedAtlas) {
		t.Fatalf("bad: %#v", c.Atlas)
	}

	actual := variablesStr(c.Variables)
	if actual != strings.TrimSpace(basicVariablesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = providerConfigsStr(c.ProviderConfigs)
	if actual != strings.TrimSpace(basicProvidersStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = resourcesStr(c.Resources)
	if actual != strings.TrimSpace(basicResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	if actual, want := localsStr(c.Locals), strings.TrimSpace(basicLocalsStr); actual != want {
		t.Fatalf("wrong locals:\n%s\nwant:\n%s", actual, want)
	}

	actual = outputsStr(c.Outputs)
	if actual != strings.TrimSpace(basicOutputsStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadFileBasic_modules(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "modules.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	if c.Dir != "" {
		t.Fatalf("bad: %#v", c.Dir)
	}

	actual := modulesStr(c.Modules)
	if actual != strings.TrimSpace(modulesModulesStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadFile_unnamedModule(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "module-unnamed.tf"))
	if err == nil {
		t.Fatalf("bad: expected error")
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, `"module" must be followed`) {
		t.Fatalf("bad: expected error has wrong text: %s", errorStr)
	}
}

func TestLoadFile_outputDependsOn(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "output-depends-on.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	if c.Dir != "" {
		t.Fatalf("bad: %#v", c.Dir)
	}

	actual := outputsStr(c.Outputs)
	if actual != strings.TrimSpace(outputDependsOnStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadFile_terraformBackend(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "terraform-backend.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	if c.Dir != "" {
		t.Fatalf("bad: %#v", c.Dir)
	}

	{
		actual := terraformStr(c.Terraform)
		expected := strings.TrimSpace(`
backend (s3)
  foo`)
		if actual != expected {
			t.Fatalf("bad:\n%s", actual)
		}
	}
}

func TestLoadFile_terraformBackendJSON(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "terraform-backend.tf.json"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	if c.Dir != "" {
		t.Fatalf("bad: %#v", c.Dir)
	}

	{
		actual := terraformStr(c.Terraform)
		expected := strings.TrimSpace(`
backend (s3)
  foo`)
		if actual != expected {
			t.Fatalf("bad:\n%s", actual)
		}
	}
}

// test that the alternate, more obvious JSON format also decodes properly
func TestLoadFile_terraformBackendJSON2(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "terraform-backend-2.tf.json"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	if c.Dir != "" {
		t.Fatalf("bad: %#v", c.Dir)
	}

	{
		actual := terraformStr(c.Terraform)
		expected := strings.TrimSpace(`
backend (s3)
  foo`)
		if actual != expected {
			t.Fatalf("bad:\n%s", actual)
		}
	}
}

func TestLoadFile_terraformBackendMulti(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "terraform-backend-multi.tf"))
	if err == nil {
		t.Fatal("expected error")
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, "only one 'backend'") {
		t.Fatalf("bad: expected error has wrong text: %s", errorStr)
	}
}

func TestLoadJSONBasic(t *testing.T) {
	raw, err := ioutil.ReadFile(filepath.Join(fixtureDir, "basic.tf.json"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	c, err := LoadJSON(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	if c.Dir != "" {
		t.Fatalf("bad: %#v", c.Dir)
	}

	expectedAtlas := &AtlasConfig{Name: "mitchellh/foo"}
	if !reflect.DeepEqual(c.Atlas, expectedAtlas) {
		t.Fatalf("bad: %#v", c.Atlas)
	}

	actual := variablesStr(c.Variables)
	if actual != strings.TrimSpace(basicVariablesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = providerConfigsStr(c.ProviderConfigs)
	if actual != strings.TrimSpace(basicProvidersStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = resourcesStr(c.Resources)
	if actual != strings.TrimSpace(basicResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = outputsStr(c.Outputs)
	if actual != strings.TrimSpace(basicOutputsStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadJSONAmbiguous(t *testing.T) {
	js := `
{
  "variable": {
    "first": {
      "default": {
        "key": "val"
      }
    },
    "second": {
      "description": "Described",
      "default": {
        "key": "val"
      }
    }
  }
}
`

	c, err := LoadJSON([]byte(js))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(c.Variables) != 2 {
		t.Fatal("config should have 2 variables, found", len(c.Variables))
	}

	first := &Variable{
		Name:    "first",
		Default: map[string]interface{}{"key": "val"},
	}
	second := &Variable{
		Name:        "second",
		Description: "Described",
		Default:     map[string]interface{}{"key": "val"},
	}

	if !reflect.DeepEqual(first, c.Variables[0]) {
		t.Fatalf("\nexpected: %#v\ngot:      %#v", first, c.Variables[0])
	}

	if !reflect.DeepEqual(second, c.Variables[1]) {
		t.Fatalf("\nexpected: %#v\ngot:      %#v", second, c.Variables[1])
	}
}

func TestLoadFileBasic_jsonNoName(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "resource-no-name.tf.json"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	actual := resourcesStr(c.Resources)
	if actual != strings.TrimSpace(basicJsonNoNameResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadFile_variables(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "variables.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if c == nil {
		t.Fatal("config should not be nil")
	}

	if c.Dir != "" {
		t.Fatalf("bad: %#v", c.Dir)
	}

	actual := variablesStr(c.Variables)
	if actual != strings.TrimSpace(variablesVariablesStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadDir_basic(t *testing.T) {
	dir := filepath.Join(fixtureDir, "dir-basic")
	c, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if c.Dir != dirAbs {
		t.Fatalf("bad: %#v", c.Dir)
	}

	actual := variablesStr(c.Variables)
	if actual != strings.TrimSpace(dirBasicVariablesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = providerConfigsStr(c.ProviderConfigs)
	if actual != strings.TrimSpace(dirBasicProvidersStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = resourcesStr(c.Resources)
	if actual != strings.TrimSpace(dirBasicResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = outputsStr(c.Outputs)
	if actual != strings.TrimSpace(dirBasicOutputsStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadDir_file(t *testing.T) {
	_, err := LoadDir(filepath.Join(fixtureDir, "variables.tf"))
	if err == nil {
		t.Fatal("should error")
	}
}

func TestLoadDir_noConfigs(t *testing.T) {
	_, err := LoadDir(filepath.Join(fixtureDir, "dir-empty"))
	if err == nil {
		t.Fatal("should error")
	}
}

func TestLoadDir_noMerge(t *testing.T) {
	c, err := LoadDir(filepath.Join(fixtureDir, "dir-merge"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	if err := c.Validate(); err == nil {
		t.Fatal("should not be valid")
	}
}

func TestLoadDir_override(t *testing.T) {
	c, err := LoadDir(filepath.Join(fixtureDir, "dir-override"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	actual := variablesStr(c.Variables)
	if actual != strings.TrimSpace(dirOverrideVariablesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = providerConfigsStr(c.ProviderConfigs)
	if actual != strings.TrimSpace(dirOverrideProvidersStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = resourcesStr(c.Resources)
	if actual != strings.TrimSpace(dirOverrideResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	actual = outputsStr(c.Outputs)
	if actual != strings.TrimSpace(dirOverrideOutputsStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadDir_overrideVar(t *testing.T) {
	c, err := LoadDir(filepath.Join(fixtureDir, "dir-override-var"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	actual := variablesStr(c.Variables)
	if actual != strings.TrimSpace(dirOverrideVarsVariablesStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadFile_mismatchedVariableTypes(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "variable-mismatched-type.tf"))
	if err == nil {
		t.Fatalf("bad: expected error")
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, "'not_a_map' has a default value which is not of type 'string'") {
		t.Fatalf("bad: expected error has wrong text: %s", errorStr)
	}
}

func TestLoadFile_badVariableTypes(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "bad-variable-type.tf"))
	if err == nil {
		t.Fatalf("bad: expected error")
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, "'bad_type' type must be one of") {
		t.Fatalf("bad: expected error has wrong text: %s", errorStr)
	}
}

func TestLoadFile_variableNoName(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "variable-no-name.tf"))
	if err == nil {
		t.Fatalf("bad: expected error")
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, `"variable" must be followed`) {
		t.Fatalf("bad: expected error has wrong text: %s", errorStr)
	}
}

func TestLoadFile_provisioners(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "provisioners.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	actual := resourcesStr(c.Resources)
	if actual != strings.TrimSpace(provisionerResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadFile_provisionersDestroy(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "provisioners-destroy.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	actual := resourcesStr(c.Resources)
	if actual != strings.TrimSpace(provisionerDestroyResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestLoadFile_unnamedOutput(t *testing.T) {
	_, err := LoadFile(filepath.Join(fixtureDir, "output-unnamed.tf"))
	if err == nil {
		t.Fatalf("bad: expected error")
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, `"output" must be followed`) {
		t.Fatalf("bad: expected error has wrong text: %s", errorStr)
	}
}

func TestLoadFile_connections(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "connection.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	actual := resourcesStr(c.Resources)
	if actual != strings.TrimSpace(connectionResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	// Check for the connection info
	r := c.Resources[0]
	if r.Name != "web" && r.Type != "aws_instance" {
		t.Fatalf("Bad: %#v", r)
	}

	p1 := r.Provisioners[0]
	if p1.ConnInfo == nil || len(p1.ConnInfo.Raw) != 2 {
		t.Fatalf("Bad: %#v", p1.ConnInfo)
	}
	if p1.ConnInfo.Raw["user"] != "nobody" {
		t.Fatalf("Bad: %#v", p1.ConnInfo)
	}

	p2 := r.Provisioners[1]
	if p2.ConnInfo == nil || len(p2.ConnInfo.Raw) != 2 {
		t.Fatalf("Bad: %#v", p2.ConnInfo)
	}
	if p2.ConnInfo.Raw["user"] != "root" {
		t.Fatalf("Bad: %#v", p2.ConnInfo)
	}
}

func TestLoadFile_createBeforeDestroy(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "create-before-destroy.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	actual := resourcesStr(c.Resources)
	if actual != strings.TrimSpace(createBeforeDestroyResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	// Check for the flag value
	r := c.Resources[0]
	if r.Name != "web" && r.Type != "aws_instance" {
		t.Fatalf("Bad: %#v", r)
	}

	// Should enable create before destroy
	if !r.Lifecycle.CreateBeforeDestroy {
		t.Fatalf("Bad: %#v", r)
	}

	r = c.Resources[1]
	if r.Name != "bar" && r.Type != "aws_instance" {
		t.Fatalf("Bad: %#v", r)
	}

	// Should not enable create before destroy
	if r.Lifecycle.CreateBeforeDestroy {
		t.Fatalf("Bad: %#v", r)
	}
}

func TestLoadFile_ignoreChanges(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "ignore-changes.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	actual := resourcesStr(c.Resources)
	print(actual)
	if actual != strings.TrimSpace(ignoreChangesResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	// Check for the flag value
	r := c.Resources[0]
	if r.Name != "web" && r.Type != "aws_instance" {
		t.Fatalf("Bad: %#v", r)
	}

	// Should populate ignore changes
	if len(r.Lifecycle.IgnoreChanges) == 0 {
		t.Fatalf("Bad: %#v", r)
	}

	r = c.Resources[1]
	if r.Name != "bar" && r.Type != "aws_instance" {
		t.Fatalf("Bad: %#v", r)
	}

	// Should not populate ignore changes
	if len(r.Lifecycle.IgnoreChanges) > 0 {
		t.Fatalf("Bad: %#v", r)
	}

	r = c.Resources[2]
	if r.Name != "baz" && r.Type != "aws_instance" {
		t.Fatalf("Bad: %#v", r)
	}

	// Should not populate ignore changes
	if len(r.Lifecycle.IgnoreChanges) > 0 {
		t.Fatalf("Bad: %#v", r)
	}
}

func TestLoad_preventDestroyString(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "prevent-destroy-string.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	actual := resourcesStr(c.Resources)
	if actual != strings.TrimSpace(createBeforeDestroyResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	// Check for the flag value
	r := c.Resources[0]
	if r.Name != "web" && r.Type != "aws_instance" {
		t.Fatalf("Bad: %#v", r)
	}

	// Should enable create before destroy
	if !r.Lifecycle.PreventDestroy {
		t.Fatalf("Bad: %#v", r)
	}

	r = c.Resources[1]
	if r.Name != "bar" && r.Type != "aws_instance" {
		t.Fatalf("Bad: %#v", r)
	}

	// Should not enable create before destroy
	if r.Lifecycle.PreventDestroy {
		t.Fatalf("Bad: %#v", r)
	}
}

func TestLoad_temporary_files(t *testing.T) {
	_, err := LoadDir(filepath.Join(fixtureDir, "dir-temporary-files"))
	if err == nil {
		t.Fatalf("Expected to see an error stating no config files found")
	}
}

func TestLoad_hclAttributes(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "attributes.tf"))
	if err != nil {
		t.Fatalf("Bad: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	actual := resourcesStr(c.Resources)
	print(actual)
	if actual != strings.TrimSpace(jsonAttributeStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	r := c.Resources[0]
	if r.Name != "test" && r.Type != "cloudstack_firewall" {
		t.Fatalf("Bad: %#v", r)
	}

	raw := r.RawConfig
	if raw.Raw["ipaddress"] != "192.168.0.1" {
		t.Fatalf("Bad: %s", raw.Raw["ipAddress"])
	}

	rule := raw.Raw["rule"].([]map[string]interface{})[0]
	if rule["protocol"] != "tcp" {
		t.Fatalf("Bad: %s", rule["protocol"])
	}

	if rule["source_cidr"] != "10.0.0.0/8" {
		t.Fatalf("Bad: %s", rule["source_cidr"])
	}

	ports := rule["ports"].([]interface{})

	if ports[0] != "80" {
		t.Fatalf("Bad ports: %s", ports[0])
	}
	if ports[1] != "1000-2000" {
		t.Fatalf("Bad ports: %s", ports[1])
	}
}

func TestLoad_jsonAttributes(t *testing.T) {
	c, err := LoadFile(filepath.Join(fixtureDir, "attributes.tf.json"))
	if err != nil {
		t.Fatalf("Bad: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}

	actual := resourcesStr(c.Resources)
	print(actual)
	if actual != strings.TrimSpace(jsonAttributeStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	r := c.Resources[0]
	if r.Name != "test" && r.Type != "cloudstack_firewall" {
		t.Fatalf("Bad: %#v", r)
	}

	raw := r.RawConfig
	if raw.Raw["ipaddress"] != "192.168.0.1" {
		t.Fatalf("Bad: %s", raw.Raw["ipAddress"])
	}

	rule := raw.Raw["rule"].([]map[string]interface{})[0]
	if rule["protocol"] != "tcp" {
		t.Fatalf("Bad: %s", rule["protocol"])
	}

	if rule["source_cidr"] != "10.0.0.0/8" {
		t.Fatalf("Bad: %s", rule["source_cidr"])
	}

	ports := rule["ports"].([]interface{})

	if ports[0] != "80" {
		t.Fatalf("Bad ports: %s", ports[0])
	}
	if ports[1] != "1000-2000" {
		t.Fatalf("Bad ports: %s", ports[1])
	}
}

const jsonAttributeStr = `
cloudstack_firewall.test (x1)
  ipaddress
  rule
`

const windowsHeredocResourcesStr = `
aws_instance.test (x1)
  user_data
`

const heredocProvidersStr = `
aws
  access_key
  secret_key
`

const heredocResourcesStr = `
aws_iam_policy.policy (x1)
  description
  name
  path
  policy
aws_instance.heredocwithnumbers (x1)
  ami
  provisioners
    local-exec
      command
aws_instance.test (x1)
  ami
  provisioners
    remote-exec
      inline
`

const basicOutputsStr = `
web_ip
  vars
    resource: aws_instance.web.private_ip
`

const basicLocalsStr = `
literal
literal_list
literal_map
security_group_ids
  vars
    resource: aws_security_group.firewall.*.id
web_ip
  vars
    resource: aws_instance.web.private_ip
`

const basicProvidersStr = `
aws
  access_key
  secret_key
do
  api_key
  vars
    user: var.foo
`

const basicResourcesStr = `
aws_instance.db (x1)
  VPC
  security_groups
  provisioners
    file
      destination
      source
  dependsOn
    aws_instance.web
  vars
    resource: aws_security_group.firewall.*.id
aws_instance.web (x1)
  ami
  network_interface
  security_groups
  provisioners
    file
      destination
      source
  vars
    resource: aws_security_group.firewall.foo
    user: var.foo
aws_security_group.firewall (x5)
data.do.depends (x1)
  dependsOn
    data.do.simple
data.do.simple (x1)
  foo
`

const basicVariablesStr = `
bar (required) (string)
  <>
  <>
baz (map)
  map[key:value]
  <>
foo
  bar
  bar
`

const basicJsonNoNameResourcesStr = `
aws_security_group.allow_external_http_https (x1)
  tags
`

const dirBasicOutputsStr = `
web_ip
  vars
    resource: aws_instance.web.private_ip
`

const dirBasicProvidersStr = `
aws
  access_key
  secret_key
do
  api_key
  vars
    user: var.foo
`

const dirBasicResourcesStr = `
aws_instance.db (x1)
  security_groups
  vars
    resource: aws_security_group.firewall.*.id
aws_instance.web (x1)
  ami
  network_interface
  security_groups
  vars
    resource: aws_security_group.firewall.foo
    user: var.foo
aws_security_group.firewall (x5)
data.do.depends (x1)
  dependsOn
    data.do.simple
data.do.simple (x1)
  foo
`

const dirBasicVariablesStr = `
foo
  bar
  bar
`

const dirOverrideOutputsStr = `
web_ip
  vars
    resource: aws_instance.web.private_ip
`

const dirOverrideProvidersStr = `
aws
  access_key
  secret_key
do
  api_key
  vars
    user: var.foo
`

const dirOverrideResourcesStr = `
aws_instance.db (x1)
  ami
  security_groups
aws_instance.web (x1)
  ami
  foo
  network_interface
  security_groups
  vars
    resource: aws_security_group.firewall.foo
    user: var.foo
aws_security_group.firewall (x5)
data.do.depends (x1)
  hello
  dependsOn
    data.do.simple
data.do.simple (x1)
  foo
`

const dirOverrideVariablesStr = `
foo
  bar
  bar
`

const dirOverrideVarsVariablesStr = `
foo
  baz
  bar
`

const importProvidersStr = `
aws
  bar
  foo
`

const importResourcesStr = `
aws_security_group.db (x1)
aws_security_group.web (x1)
`

const importVariablesStr = `
bar (required)
  <>
  <>
foo
  bar
  bar
`

const modulesModulesStr = `
bar
  source = baz
  memory
`

const provisionerResourcesStr = `
aws_instance.web (x1)
  ami
  security_groups
  provisioners
    shell
      path
  vars
    resource: aws_security_group.firewall.foo
    user: var.foo
`

const provisionerDestroyResourcesStr = `
aws_instance.web (x1)
  provisioners
    shell
    shell (destroy)
      path
    shell (destroy)
      on_failure = continue
      path
`

const connectionResourcesStr = `
aws_instance.web (x1)
  ami
  security_groups
  provisioners
    shell
      path
    shell
      path
  vars
    resource: aws_security_group.firewall.foo
    user: var.foo
`

const outputDependsOnStr = `
value
  dependsOn
    foo
`

const variablesVariablesStr = `
bar
  <>
  <>
baz
  foo
  <>
foo (required)
  <>
  <>
`

const createBeforeDestroyResourcesStr = `
aws_instance.bar (x1)
  ami
aws_instance.web (x1)
  ami
`

const ignoreChangesResourcesStr = `
aws_instance.bar (x1)
  ami
aws_instance.baz (x1)
  ami
aws_instance.web (x1)
  ami
`
