// The config package is responsible for loading and validating the
// configuration.
package config

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/multierror"
	"github.com/mitchellh/mapstructure"
	"github.com/mitchellh/reflectwalk"
)

// Config is the configuration that comes from loading a collection
// of Terraform templates.
type Config struct {
	// Dir is the path to the directory where this configuration was
	// loaded from. If it is blank, this configuration wasn't loaded from
	// any meaningful directory.
	Dir string

	Modules         []*Module
	ProviderConfigs []*ProviderConfig
	Resources       []*Resource
	Variables       []*Variable
	Outputs         []*Output

	// The fields below can be filled in by loaders for validation
	// purposes.
	unknownKeys []string
}

// Module is a module used within a configuration.
//
// This does not represent a module itself, this represents a module
// call-site within an existing configuration.
type Module struct {
	Name      string
	Source    string
	RawConfig *RawConfig
}

// ProviderConfig is the configuration for a resource provider.
//
// For example, Terraform needs to set the AWS access keys for the AWS
// resource provider.
type ProviderConfig struct {
	Name      string
	RawConfig *RawConfig
}

// A resource represents a single Terraform resource in the configuration.
// A Terraform resource is something that represents some component that
// can be created and managed, and has some properties associated with it.
type Resource struct {
	Name                string
	Type                string
	Count               int
	RawConfig           *RawConfig
	Provisioners        []*Provisioner
	DependsOn           []string
	CreateBeforeDestroy bool
}

// Provisioner is a configured provisioner step on a resource.
type Provisioner struct {
	Type      string
	RawConfig *RawConfig
	ConnInfo  *RawConfig
}

// Variable is a variable defined within the configuration.
type Variable struct {
	Name        string
	Default     interface{}
	Description string
}

// Output is an output defined within the configuration. An output is
// resulting data that is highlighted by Terraform when finished.
type Output struct {
	Name      string
	RawConfig *RawConfig
}

// VariableType is the type of value a variable is holding, and returned
// by the Type() function on variables.
type VariableType byte

const (
	VariableTypeUnknown VariableType = iota
	VariableTypeString
	VariableTypeMap
)

// ProviderConfigName returns the name of the provider configuration in
// the given mapping that maps to the proper provider configuration
// for this resource.
func ProviderConfigName(t string, pcs []*ProviderConfig) string {
	lk := ""
	for _, v := range pcs {
		k := v.Name
		if strings.HasPrefix(t, k) && len(k) > len(lk) {
			lk = k
		}
	}

	return lk
}

// A unique identifier for this module.
func (r *Module) Id() string {
	return fmt.Sprintf("%s", r.Name)
}

// A unique identifier for this resource.
func (r *Resource) Id() string {
	return fmt.Sprintf("%s.%s", r.Type, r.Name)
}

// Validate does some basic semantic checking of the configuration.
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}

	var errs []error

	for _, k := range c.unknownKeys {
		errs = append(errs, fmt.Errorf(
			"Unknown root level key: %s", k))
	}

	vars := c.InterpolatedVariables()
	varMap := make(map[string]*Variable)
	for _, v := range c.Variables {
		varMap[v.Name] = v
	}

	for _, v := range c.Variables {
		if v.Type() == VariableTypeUnknown {
			errs = append(errs, fmt.Errorf(
				"Variable '%s': must be string or mapping",
				v.Name))
			continue
		}

		interp := false
		fn := func(i Interpolation) (string, error) {
			interp = true
			return "", nil
		}

		w := &interpolationWalker{F: fn}
		if v.Default != nil {
			if err := reflectwalk.Walk(v.Default, w); err == nil {
				if interp {
					errs = append(errs, fmt.Errorf(
						"Variable '%s': cannot contain interpolations",
						v.Name))
				}
			}
		}
	}

	// Check for references to user variables that do not actually
	// exist and record those errors.
	for source, vs := range vars {
		for _, v := range vs {
			uv, ok := v.(*UserVariable)
			if !ok {
				continue
			}

			if _, ok := varMap[uv.Name]; !ok {
				errs = append(errs, fmt.Errorf(
					"%s: unknown variable referenced: %s",
					source,
					uv.Name))
			}
		}
	}

	// Check that all references to modules are valid
	modules := make(map[string]*Module)
	dupped := make(map[string]struct{})
	for _, m := range c.Modules {
		if _, ok := modules[m.Id()]; ok {
			if _, ok := dupped[m.Id()]; !ok {
				dupped[m.Id()] = struct{}{}

				errs = append(errs, fmt.Errorf(
					"%s: module repeated multiple times",
					m.Id()))
			}
		}

		modules[m.Id()] = m
	}
	dupped = nil

	// Check that all variables for modules reference modules that
	// exist.
	for source, vs := range vars {
		for _, v := range vs {
			mv, ok := v.(*ModuleVariable)
			if !ok {
				continue
			}

			if _, ok := modules[mv.Name]; !ok {
				errs = append(errs, fmt.Errorf(
					"%s: unknown module referenced: %s",
					source,
					mv.Name))
			}
		}
	}

	// Check that all references to resources are valid
	resources := make(map[string]*Resource)
	dupped = make(map[string]struct{})
	for _, r := range c.Resources {
		if _, ok := resources[r.Id()]; ok {
			if _, ok := dupped[r.Id()]; !ok {
				dupped[r.Id()] = struct{}{}

				errs = append(errs, fmt.Errorf(
					"%s: resource repeated multiple times",
					r.Id()))
			}
		}

		resources[r.Id()] = r
	}
	dupped = nil

	// Validate resources
	for n, r := range resources {
		if r.Count < 1 {
			errs = append(errs, fmt.Errorf(
				"%s: count must be greater than or equal to 1",
				n))
		}

		for _, d := range r.DependsOn {
			if _, ok := resources[d]; !ok {
				errs = append(errs, fmt.Errorf(
					"%s: resource depends on non-existent resource '%s'",
					n, d))
			}
		}
	}

	for source, vs := range vars {
		for _, v := range vs {
			rv, ok := v.(*ResourceVariable)
			if !ok {
				continue
			}

			id := fmt.Sprintf("%s.%s", rv.Type, rv.Name)
			r, ok := resources[id]
			if !ok {
				errs = append(errs, fmt.Errorf(
					"%s: unknown resource '%s' referenced in variable %s",
					source,
					id,
					rv.FullKey()))
				continue
			}

			// If it is a multi reference and resource has a single
			// count, it is an error.
			if r.Count > 1 && !rv.Multi {
				errs = append(errs, fmt.Errorf(
					"%s: variable '%s' must specify index for multi-count "+
						"resource %s",
					source,
					rv.FullKey(),
					id))
				continue
			}
		}
	}

	// Check that all outputs are valid
	for _, o := range c.Outputs {
		invalid := false
		for k, _ := range o.RawConfig.Raw {
			if k != "value" {
				invalid = true
				break
			}
		}
		if invalid {
			errs = append(errs, fmt.Errorf(
				"%s: output should only have 'value' field", o.Name))
		}
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	return nil
}

// InterpolatedVariables is a helper that returns a mapping of all the interpolated
// variables within the configuration. This is used to verify references
// are valid in the Validate step.
func (c *Config) InterpolatedVariables() map[string][]InterpolatedVariable {
	result := make(map[string][]InterpolatedVariable)
	for _, pc := range c.ProviderConfigs {
		source := fmt.Sprintf("provider config '%s'", pc.Name)
		for _, v := range pc.RawConfig.Variables {
			result[source] = append(result[source], v)
		}
	}

	for _, rc := range c.Resources {
		source := fmt.Sprintf("resource '%s'", rc.Id())
		for _, v := range rc.RawConfig.Variables {
			result[source] = append(result[source], v)
		}
	}

	for _, o := range c.Outputs {
		source := fmt.Sprintf("output '%s'", o.Name)
		for _, v := range o.RawConfig.Variables {
			result[source] = append(result[source], v)
		}
	}

	return result
}

func (m *Module) mergerName() string {
	return m.Id()
}

func (m *Module) mergerMerge(other merger) merger {
	m2 := other.(*Module)

	result := *m
	result.Name = m2.Name
	result.RawConfig = result.RawConfig.merge(m2.RawConfig)

	if m2.Source != "" {
		result.Source = m2.Source
	}

	return &result
}

func (o *Output) mergerName() string {
	return o.Name
}

func (o *Output) mergerMerge(m merger) merger {
	o2 := m.(*Output)

	result := *o
	result.Name = o2.Name
	result.RawConfig = result.RawConfig.merge(o2.RawConfig)

	return &result
}

func (c *ProviderConfig) mergerName() string {
	return c.Name
}

func (c *ProviderConfig) mergerMerge(m merger) merger {
	c2 := m.(*ProviderConfig)

	result := *c
	result.Name = c2.Name
	result.RawConfig = result.RawConfig.merge(c2.RawConfig)

	return &result
}

func (r *Resource) mergerName() string {
	return fmt.Sprintf("%s.%s", r.Type, r.Name)
}

func (r *Resource) mergerMerge(m merger) merger {
	r2 := m.(*Resource)

	result := *r
	result.Name = r2.Name
	result.Type = r2.Type
	result.RawConfig = result.RawConfig.merge(r2.RawConfig)

	if r2.Count > 0 {
		result.Count = r2.Count
	}

	if len(r2.Provisioners) > 0 {
		result.Provisioners = r2.Provisioners
	}

	return &result
}

// DefaultsMap returns a map of default values for this variable.
func (v *Variable) DefaultsMap() map[string]string {
	if v.Default == nil {
		return nil
	}

	n := fmt.Sprintf("var.%s", v.Name)
	switch v.Type() {
	case VariableTypeString:
		return map[string]string{n: v.Default.(string)}
	case VariableTypeMap:
		result := flatmap.Flatten(map[string]interface{}{
			n: v.Default.(map[string]string),
		})
		result[n] = v.Name

		return result
	default:
		return nil
	}
}

// Merge merges two variables to create a new third variable.
func (v *Variable) Merge(v2 *Variable) *Variable {
	// Shallow copy the variable
	result := *v

	// The names should be the same, but the second name always wins.
	result.Name = v2.Name

	if v2.Default != nil {
		result.Default = v2.Default
	}
	if v2.Description != "" {
		result.Description = v2.Description
	}

	return &result
}

// Type returns the type of varialbe this is.
func (v *Variable) Type() VariableType {
	if v.Default == nil {
		return VariableTypeString
	}

	var strVal string
	if err := mapstructure.WeakDecode(v.Default, &strVal); err == nil {
		v.Default = strVal
		return VariableTypeString
	}

	var m map[string]string
	if err := mapstructure.WeakDecode(v.Default, &m); err == nil {
		v.Default = m
		return VariableTypeMap
	}

	return VariableTypeUnknown
}

func (v *Variable) mergerName() string {
	return v.Name
}

func (v *Variable) mergerMerge(m merger) merger {
	return v.Merge(m.(*Variable))
}

// Required tests whether a variable is required or not.
func (v *Variable) Required() bool {
	return v.Default == nil
}
