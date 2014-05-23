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

	/*
	actual = resourcesStr(c.Resources)
	if actual != strings.TrimSpace(basicResourcesStr) {
		t.Fatalf("bad:\n%s", actual)
	}
	*/
}

// This helper turns a resources field into a deterministic
// string value for comparison in tests.
func resourcesStr(rs []Resource) string {
	result := ""
	for _, r := range rs {
		result += fmt.Sprintf(
			"%s[%s]\n",
			r.Type,
			r.Name)

		for k, _ := range r.Config {
			result += fmt.Sprintf("  %s\n", k)
		}
	}

	return strings.TrimSpace(result)
}

// This helper turns a variables field into a deterministic
// string value for comparison in tests.
func variablesStr(vs map[string]Variable) string {
	result := ""
	for k, v := range vs {
		result += fmt.Sprintf(
			"%s\n  %s\n  %s\n",
			k,
			v.Default,
			v.Description)
	}

	return strings.TrimSpace(result)
}

const basicResourcesStr = `
aws_security_group[firewall]
aws_instance[web]
  ami
  security_groups
`

const basicVariablesStr = `
foo
  bar
  bar
`

const importVariablesStr = `
foo
  bar
  bar
bar
`
