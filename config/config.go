// The config package is responsible for loading and validating the
// configuration.
package config

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/multierror"
)

// Config is the configuration that comes from loading a collection
// of Terraform templates.
type Config struct {
	ProviderConfigs map[string]*ProviderConfig
	Resources       []*Resource
	Variables       map[string]*Variable
}

// ProviderConfig is the configuration for a resource provider.
//
// For example, Terraform needs to set the AWS access keys for the AWS
// resource provider.
type ProviderConfig struct {
	RawConfig *RawConfig
}

// A resource represents a single Terraform resource in the configuration.
// A Terraform resource is something that represents some component that
// can be created and managed, and has some properties associated with it.
type Resource struct {
	Name      string
	Type      string
	Count     int
	RawConfig *RawConfig
}

// Variable is a variable defined within the configuration.
type Variable struct {
	Default     string
	Description string
	defaultSet  bool
}

// An InterpolatedVariable is a variable that is embedded within a string
// in the configuration, such as "hello ${world}" (world in this case is
// an interpolated variable).
//
// These variables can come from a variety of sources, represented by
// implementations of this interface.
type InterpolatedVariable interface {
	FullKey() string
}

// A ResourceVariable is a variable that is referencing the field
// of a resource, such as "${aws_instance.foo.ami}"
type ResourceVariable struct {
	Type  string
	Name  string
	Field string

	key string
}

// A UserVariable is a variable that is referencing a user variable
// that is inputted from outside the configuration. This looks like
// "${var.foo}"
type UserVariable struct {
	Name string

	key string
}

// ProviderConfigName returns the name of the provider configuration in
// the given mapping that maps to the proper provider configuration
// for this resource.
func ProviderConfigName(t string, pcs map[string]*ProviderConfig) string {
	lk := ""
	for k, _ := range pcs {
		if strings.HasPrefix(t, k) && len(k) > len(lk) {
			lk = k
		}
	}

	return lk
}

// A unique identifier for this resource.
func (r *Resource) Id() string {
	return fmt.Sprintf("%s.%s", r.Type, r.Name)
}

// Validate does some basic semantic checking of the configuration.
func (c *Config) Validate() error {
	var errs []error

	vars := c.allVariables()

	// Check for references to user variables that do not actually
	// exist and record those errors.
	for source, v := range vars {
		uv, ok := v.(*UserVariable)
		if !ok {
			continue
		}

		if _, ok := c.Variables[uv.Name]; !ok {
			errs = append(errs, fmt.Errorf(
				"%s: unknown variable referenced: %s",
				source,
				uv.Name))
		}
	}

	// Check that all references to resources are valid
	resources := make(map[string]struct{})
	for _, r := range c.Resources {
		resources[r.Id()] = struct{}{}
	}
	for source, v := range vars {
		rv, ok := v.(*ResourceVariable)
		if !ok {
			continue
		}

		id := fmt.Sprintf("%s.%s", rv.Type, rv.Name)
		if _, ok := resources[id]; !ok {
			errs = append(errs, fmt.Errorf(
				"%s: unknown resource '%s' referenced in variable %s",
				source,
				id,
				rv.FullKey()))
		}
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	return nil
}

// allVariables is a helper that returns a mapping of all the interpolated
// variables within the configuration. This is used to verify references
// are valid in the Validate step.
func (c *Config) allVariables() map[string]InterpolatedVariable {
	result := make(map[string]InterpolatedVariable)
	for n, pc := range c.ProviderConfigs {
		source := fmt.Sprintf("provider config '%s'", n)
		for _, v := range pc.RawConfig.Variables {
			result[source] = v
		}
	}

	for _, rc := range c.Resources {
		source := fmt.Sprintf("resource '%s'", rc.Id())
		for _, v := range rc.RawConfig.Variables {
			result[source] = v
		}
	}

	return result
}

// Required tests whether a variable is required or not.
func (v *Variable) Required() bool {
	return !v.defaultSet
}

func NewResourceVariable(key string) (*ResourceVariable, error) {
	parts := strings.SplitN(key, ".", 3)
	return &ResourceVariable{
		Type:  parts[0],
		Name:  parts[1],
		Field: parts[2],
		key:   key,
	}, nil
}

func (v *ResourceVariable) ResourceId() string {
	return fmt.Sprintf("%s.%s", v.Type, v.Name)
}

func (v *ResourceVariable) FullKey() string {
	return v.key
}

func NewUserVariable(key string) (*UserVariable, error) {
	name := key[len("var."):]
	return &UserVariable{
		key:  key,
		Name: name,
	}, nil
}

func (v *UserVariable) FullKey() string {
	return v.key
}
