package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

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
}

func TestLoadBasic_import(t *testing.T) {
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

func TestLoad_variables(t *testing.T) {
	c, err := Load(filepath.Join(fixtureDir, "variables.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if c == nil {
		t.Fatal("config should not be nil")
	}

	actual := variablesStr(c.Variables)
	if actual != strings.TrimSpace(variablesVariablesStr) {
		t.Fatalf("bad:\n%s", actual)
	}

	if !c.Variables["foo"].Required() {
		t.Fatal("foo should be required")
	}
	if c.Variables["bar"].Required() {
		t.Fatal("bar should not be required")
	}
	if c.Variables["baz"].Required() {
		t.Fatal("baz should not be required")
	}
}

// This helper turns a provider configs field into a deterministic
// string value for comparison in tests.
func providerConfigsStr(pcs map[string]*ProviderConfig) string {
	result := ""
	for n, pc := range pcs {
		result += fmt.Sprintf("%s\n", n)

		for k, _ := range pc.Config {
			result += fmt.Sprintf("  %s\n", k)
		}

		if len(pc.Variables) > 0 {
			result += fmt.Sprintf("  vars\n")
			for _, rawV := range pc.Variables {
				kind := "unknown"
				str := rawV.FullKey()

				switch rawV.(type) {
				case *ResourceVariable:
					kind = "resource"
				case *UserVariable:
					kind = "user"
				}

				result += fmt.Sprintf("    %s: %s\n", kind, str)
			}
		}
	}

	return strings.TrimSpace(result)
}

// This helper turns a resources field into a deterministic
// string value for comparison in tests.
func resourcesStr(rs []*Resource) string {
	result := ""
	for _, r := range rs {
		result += fmt.Sprintf(
			"%s[%s]\n",
			r.Type,
			r.Name)

		for k, _ := range r.Config {
			result += fmt.Sprintf("  %s\n", k)
		}

		if len(r.Variables) > 0 {
			result += fmt.Sprintf("  vars\n")
			for _, rawV := range r.Variables {
				kind := "unknown"
				str := rawV.FullKey()

				switch rawV.(type) {
				case *ResourceVariable:
					kind = "resource"
				case *UserVariable:
					kind = "user"
				}

				result += fmt.Sprintf("    %s: %s\n", kind, str)
			}
		}
	}

	return strings.TrimSpace(result)
}

// This helper turns a variables field into a deterministic
// string value for comparison in tests.
func variablesStr(vs map[string]*Variable) string {
	result := ""
	for k, v := range vs {
		if v.Default == "" {
			v.Default = "<>"
		}
		if v.Description == "" {
			v.Description = "<>"
		}

		result += fmt.Sprintf(
			"%s\n  %s\n  %s\n",
			k,
			v.Default,
			v.Description)
	}

	return strings.TrimSpace(result)
}

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
aws_security_group[firewall]
aws_instance[web]
  ami
  security_groups
  vars
    user: var.foo
    resource: aws_security_group.firewall.foo
`

const basicVariablesStr = `
foo
  bar
  bar
`

const importProvidersStr = `
aws
  foo
`

const importResourcesStr = `
aws_security_group[db]
aws_security_group[web]
`

const importVariablesStr = `
bar
  <>
  <>
foo
  bar
  bar
`

const variablesVariablesStr = `
foo
  <>
  <>
bar
  <>
  <>
baz
  foo
  <>
`
