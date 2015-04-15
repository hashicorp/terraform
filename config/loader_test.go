package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

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

func TestLoad_badType(t *testing.T) {
	_, err := Load(filepath.Join(fixtureDir, "bad_type.tf.nope"))
	if err == nil {
		t.Fatal("should have error")
	}
}

func TestLoadBasic(t *testing.T) {
	c, err := Load(filepath.Join(fixtureDir, "basic.tf"))
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

func TestLoadBasic_empty(t *testing.T) {
	c, err := Load(filepath.Join(fixtureDir, "empty.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
	}
}

func TestLoadBasic_import(t *testing.T) {
	// Skip because we disabled importing
	t.Skip()

	c, err := Load(filepath.Join(fixtureDir, "import.tf"))
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

func TestLoadBasic_json(t *testing.T) {
	c, err := Load(filepath.Join(fixtureDir, "basic.tf.json"))
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

func TestLoadBasic_modules(t *testing.T) {
	c, err := Load(filepath.Join(fixtureDir, "modules.tf"))
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

func TestLoad_variables(t *testing.T) {
	c, err := Load(filepath.Join(fixtureDir, "variables.tf"))
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

func TestLoad_provisioners(t *testing.T) {
	c, err := Load(filepath.Join(fixtureDir, "provisioners.tf"))
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

func TestLoad_connections(t *testing.T) {
	c, err := Load(filepath.Join(fixtureDir, "connection.tf"))
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

func TestLoad_createBeforeDestroy(t *testing.T) {
	c, err := Load(filepath.Join(fixtureDir, "create-before-destroy.tf"))
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

func TestLoad_temporary_files(t *testing.T) {
	_, err := LoadDir(filepath.Join(fixtureDir, "dir-temporary-files"))
	if err == nil {
		t.Fatalf("Expected to see an error stating no config files found")
	}
}

const basicOutputsStr = `
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
aws_instance[db] (x1)
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
aws_instance[web] (x1)
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
aws_security_group[firewall] (x5)
`

const basicVariablesStr = `
foo
  bar
  bar
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
aws_instance[db] (x1)
  security_groups
  vars
    resource: aws_security_group.firewall.*.id
aws_instance[web] (x1)
  ami
  network_interface
  security_groups
  vars
    resource: aws_security_group.firewall.foo
    user: var.foo
aws_security_group[firewall] (x5)
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
aws_instance[db] (x1)
  ami
  security_groups
aws_instance[web] (x1)
  ami
  foo
  network_interface
  security_groups
  vars
    resource: aws_security_group.firewall.foo
    user: var.foo
aws_security_group[firewall] (x5)
`

const dirOverrideVariablesStr = `
foo
  bar
  bar
`

const importProvidersStr = `
aws
  bar
  foo
`

const importResourcesStr = `
aws_security_group[db] (x1)
aws_security_group[web] (x1)
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
aws_instance[web] (x1)
  ami
  security_groups
  provisioners
    shell
      path
  vars
    resource: aws_security_group.firewall.foo
    user: var.foo
`

const connectionResourcesStr = `
aws_instance[web] (x1)
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
aws_instance[bar] (x1)
  ami
aws_instance[web] (x1)
  ami
`
