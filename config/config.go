// The config package is responsible for loading and validating the
// configuration.
package config

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/depgraph"
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

// Graph returns a dependency graph of the resources from this
// Terraform configuration.
//
// The graph can contain both *Resource and *ProviderConfig. When consuming
// the graph, you'll have to use type inference to determine what it is
// and the proper behavior.
func (c *Config) Graph() *depgraph.Graph {
	// This tracks all the resource nouns
	nouns := make(map[string]*depgraph.Noun)
	for _, r := range c.Resources {
		noun := &depgraph.Noun{
			Name: r.Id(),
			Meta: r,
		}
		nouns[noun.Name] = noun
	}

	// Build the list of nouns that we iterate over
	nounsList := make([]*depgraph.Noun, 0, len(nouns))
	for _, n := range nouns {
		nounsList = append(nounsList, n)
	}

	// This tracks the provider configs that are nouns in our dep graph
	pcNouns := make(map[string]*depgraph.Noun)

	i := 0
	for i < len(nounsList) {
		noun := nounsList[i]
		i += 1

		// Determine depenencies based on variables. Both resources
		// and provider configurations have dependencies in this case.
		var vars map[string]InterpolatedVariable
		switch n := noun.Meta.(type) {
		case *Resource:
			vars = n.RawConfig.Variables
		case *ProviderConfig:
			vars = n.RawConfig.Variables
		}
		for _, v := range vars {
			// Only resource variables impose dependencies
			rv, ok := v.(*ResourceVariable)
			if !ok {
				continue
			}

			// Build the dependency
			dep := &depgraph.Dependency{
				Name:   rv.ResourceId(),
				Source: noun,
				Target: nouns[rv.ResourceId()],
			}

			noun.Deps = append(noun.Deps, dep)
		}

		// If this is a Resource, then check if we have to also
		// depend on a provider configuration.
		if r, ok := noun.Meta.(*Resource); ok {
			// If there is a provider config that matches this resource
			// then we add that as a dependency.
			if pcName := ProviderConfigName(r.Type, c.ProviderConfigs); pcName != "" {
				pcNoun, ok := pcNouns[pcName]
				if !ok {
					pcNoun = &depgraph.Noun{
						Name: fmt.Sprintf("provider.%s", pcName),
						Meta: c.ProviderConfigs[pcName],
					}
					pcNouns[pcName] = pcNoun
					nounsList = append(nounsList, pcNoun)
				}

				dep := &depgraph.Dependency{
					Name:   pcName,
					Source: noun,
					Target: pcNoun,
				}

				noun.Deps = append(noun.Deps, dep)
			}
		}
	}

	// Create a root that just depends on everything else finishing.
	root := &depgraph.Noun{Name: "root"}
	for _, n := range nounsList {
		root.Deps = append(root.Deps, &depgraph.Dependency{
			Name:   n.Name,
			Source: root,
			Target: n,
		})
	}
	nounsList = append(nounsList, root)

	return &depgraph.Graph{
		Name:  "resources",
		Nouns: nounsList,
	}
}

// Validate does some basic semantic checking of the configuration.
func (c *Config) Validate() error {
	// TODO(mitchellh): make sure all referenced variables exist
	// TODO(mitchellh): make sure types/names have valid values (characters)

	return nil
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
