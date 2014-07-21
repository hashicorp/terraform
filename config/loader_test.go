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

func TestLoadBasic_json(t *testing.T) {
	c, err := Load(filepath.Join(fixtureDir, "basic.tf.json"))
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
}

func TestLoadDir_basic(t *testing.T) {
	c, err := LoadDir(filepath.Join(fixtureDir, "dir-basic"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if c == nil {
		t.Fatal("config should not be nil")
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

func outputsStr(os []*Output) string {
	ns := make([]string, 0, len(os))
	m := make(map[string]*Output)
	for _, o := range os {
		ns = append(ns, o.Name)
		m[o.Name] = o
	}
	sort.Strings(ns)

	result := ""
	for _, n := range ns {
		o := m[n]

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

// This helper turns a provider configs field into a deterministic
// string value for comparison in tests.
func providerConfigsStr(pcs []*ProviderConfig) string {
	result := ""

	ns := make([]string, 0, len(pcs))
	m := make(map[string]*ProviderConfig)
	for _, n := range pcs {
		ns = append(ns, n.Name)
		m[n.Name] = n
	}
	sort.Strings(ns)

	for _, n := range ns {
		pc := m[n]

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
	order := make([]int, 0, len(rs))
	ks := make([]string, 0, len(rs))
	mapping := make(map[string]int)
	for i, r := range rs {
		k := fmt.Sprintf("%s[%s]", r.Type, r.Name)
		ks = append(ks, k)
		mapping[k] = i
	}
	sort.Strings(ks)
	for _, k := range ks {
		order = append(order, mapping[k])
	}

	for _, i := range order {
		r := rs[i]
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

		if len(r.Provisioners) > 0 {
			result += fmt.Sprintf("  provisioners\n")
			for _, p := range r.Provisioners {
				result += fmt.Sprintf("    %s\n", p.Type)

				ks := make([]string, 0, len(p.RawConfig.Raw))
				for k, _ := range p.RawConfig.Raw {
					ks = append(ks, k)
				}
				sort.Strings(ks)

				for _, k := range ks {
					result += fmt.Sprintf("      %s\n", k)
				}
			}
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
func variablesStr(vs []*Variable) string {
	result := ""
	ks := make([]string, 0, len(vs))
	m := make(map[string]*Variable)
	for _, v := range vs {
		ks = append(ks, v.Name)
		m[v.Name] = v
	}
	sort.Strings(ks)

	for _, k := range ks {
		v := m[k]

		if v.Default == "" {
			v.Default = "<>"
		}
		if v.Description == "" {
			v.Description = "<>"
		}

		required := ""
		if v.Required() {
			required = " (required)"
		}

		result += fmt.Sprintf(
			"%s%s\n  %s\n  %s\n",
			k,
			required,
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
