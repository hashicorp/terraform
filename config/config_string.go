package config

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// TestString is a Stringer-like function that outputs a string that can
// be used to easily compare multiple Config structures in unit tests.
//
// This function has no practical use outside of unit tests and debugging.
func (c *Config) TestString() string {
	if c == nil {
		return "<nil config>"
	}

	var buf bytes.Buffer
	if len(c.Modules) > 0 {
		buf.WriteString("Modules:\n\n")
		buf.WriteString(modulesStr(c.Modules))
		buf.WriteString("\n\n")
	}

	if len(c.Variables) > 0 {
		buf.WriteString("Variables:\n\n")
		buf.WriteString(variablesStr(c.Variables))
		buf.WriteString("\n\n")
	}

	if len(c.ProviderConfigs) > 0 {
		buf.WriteString("Provider Configs:\n\n")
		buf.WriteString(providerConfigsStr(c.ProviderConfigs))
		buf.WriteString("\n\n")
	}

	if len(c.Resources) > 0 {
		buf.WriteString("Resources:\n\n")
		buf.WriteString(resourcesStr(c.Resources))
		buf.WriteString("\n\n")
	}

	if len(c.Outputs) > 0 {
		buf.WriteString("Outputs:\n\n")
		buf.WriteString(outputsStr(c.Outputs))
		buf.WriteString("\n")
	}

	return strings.TrimSpace(buf.String())
}

func modulesStr(ms []*Module) string {
	result := ""
	order := make([]int, 0, len(ms))
	ks := make([]string, 0, len(ms))
	mapping := make(map[string]int)
	for i, m := range ms {
		k := m.Id()
		ks = append(ks, k)
		mapping[k] = i
	}
	sort.Strings(ks)
	for _, k := range ks {
		order = append(order, mapping[k])
	}

	for _, i := range order {
		m := ms[i]
		result += fmt.Sprintf("%s\n", m.Id())

		ks := make([]string, 0, len(m.RawConfig.Raw))
		for k, _ := range m.RawConfig.Raw {
			ks = append(ks, k)
		}
		sort.Strings(ks)

		result += fmt.Sprintf("  source = %s\n", m.Source)

		for _, k := range ks {
			result += fmt.Sprintf("  %s\n", k)
		}
	}

	return strings.TrimSpace(result)
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
			"%s[%s] (x%s)\n",
			r.Type,
			r.Name,
			r.RawCount.Value())

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

		if len(r.DependsOn) > 0 {
			result += fmt.Sprintf("  dependsOn\n")
			for _, d := range r.DependsOn {
				result += fmt.Sprintf("    %s\n", d)
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

		required := ""
		if v.Required() {
			required = " (required)"
		}

		if v.Default == nil || v.Default == "" {
			v.Default = "<>"
		}
		if v.Description == "" {
			v.Description = "<>"
		}

		result += fmt.Sprintf(
			"%s%s\n  %v\n  %s\n",
			k,
			required,
			v.Default,
			v.Description)
	}

	return strings.TrimSpace(result)
}
