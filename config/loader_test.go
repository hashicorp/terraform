package config

import (
	"fmt"
	"path/filepath"
	"sort"
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

	actual = outputsStr(c.Outputs)
	if actual != strings.TrimSpace(basicOutputsStr) {
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

func outputsStr(os map[string]*Output) string {
	ns := make([]string, 0, len(os))
	for n, _ := range os {
		ns = append(ns, n)
	}
	sort.Strings(ns)

	result := ""
	for _, n := range ns {
		o := os[n]

		result += fmt.Sprintf("%s\n", n)

		if len(o.RawConfig.Variables) > 0 {
			result += fmt.Sprintf("  vars\n")
			for _, rawV := range o.RawConfig.Variables {
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

// This helper turns a provider configs field into a deterministic
// string value for comparison in tests.
func providerConfigsStr(pcs map[string]*ProviderConfig) string {
	result := ""

	ns := make([]string, 0, len(pcs))
	for n, _ := range pcs {
		ns = append(ns, n)
	}
	sort.Strings(ns)

	for _, n := range ns {
		pc := pcs[n]

		result += fmt.Sprintf("%s\n", n)

		keys := make([]string, 0, len(pc.RawConfig.Raw))
		for k, _ := range pc.RawConfig.Raw {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			result += fmt.Sprintf("  %s\n", k)
		}

		if len(pc.RawConfig.Variables) > 0 {
			result += fmt.Sprintf("  vars\n")
			for _, rawV := range pc.RawConfig.Variables {
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
			"%s[%s] (x%d)\n",
			r.Type,
			r.Name,
			r.Count)

		ks := make([]string, 0, len(r.RawConfig.Raw))
		for k, _ := range r.RawConfig.Raw {
			ks = append(ks, k)
		}
		sort.Strings(ks)

		for _, k := range ks {
			result += fmt.Sprintf("  %s\n", k)
		}

		if len(r.RawConfig.Variables) > 0 {
			result += fmt.Sprintf("  vars\n")

			ks := make([]string, 0, len(r.RawConfig.Variables))
			for k, _ := range r.RawConfig.Variables {
				ks = append(ks, k)
			}
			sort.Strings(ks)

			for _, k := range ks {
				rawV := r.RawConfig.Variables[k]
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
	ks := make([]string, 0, len(vs))
	for k, _ := range vs {
		ks = append(ks, k)
	}
	sort.Strings(ks)

	for _, k := range ks {
		v := vs[k]

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
aws_security_group[firewall] (x5)
aws_instance[web] (x1)
  ami
  network_interface
  security_groups
  vars
    resource: aws_security_group.firewall.foo
    user: var.foo
aws_instance[db] (x1)
  security_groups
  vars
    resource: aws_security_group.firewall.*.id
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
aws_security_group[db] (x1)
aws_security_group[web] (x1)
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
bar
  <>
  <>
baz
  foo
  <>
foo
  <>
  <>
`
