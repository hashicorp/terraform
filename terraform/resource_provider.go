package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// ResourceProvider is an interface that must be implemented by any
// resource provider: the thing that creates and manages the resources in
// a Terraform configuration.
type ResourceProvider interface {
	// Validate is called once at the beginning with the raw configuration
	// (no interpolation done) and can return a list of warnings and/or
	// errors.
	//
	// This should not assume that any values of the configurations are valid.
	// The primary use case of this call is to check that required keys are
	// set.
	Validate(*ResourceConfig) ([]string, []error)

	// Configure configures the provider itself with the configuration
	// given. This is useful for setting things like access keys.
	//
	// Configure returns an error if it occurred.
	Configure(*ResourceConfig) error

	// Resources returns all the available resource types that this provider
	// knows how to manage.
	Resources() []ResourceType

	// Apply applies a diff to a specific resource and returns the new
	// resource state along with an error.
	//
	// If the resource state given has an empty ID, then a new resource
	// is expected to be created.
	//Apply(ResourceState, ResourceDiff) (ResourceState, error)

	// Diff diffs a resource versus a desired state and returns
	// a diff.
	Diff(
		*ResourceState,
		*ResourceConfig) (*ResourceDiff, error)
}

// ResourceConfig holds the configuration given for a resource. This is
// done instead of a raw `map[string]interface{}` type so that rich
// methods can be added to it to make dealing with it easier.
type ResourceConfig struct {
	ComputedKeys []string
	Raw          map[string]interface{}
}

// ResourceType is a type of resource that a resource provider can manage.
type ResourceType struct {
	Name string
}

// ResourceProviderFactory is a function type that creates a new instance
// of a resource provider.
type ResourceProviderFactory func() (ResourceProvider, error)

// NewResourceConfig creates a new ResourceConfig from a config.RawConfig.
func NewResourceConfig(c *config.RawConfig) *ResourceConfig {
	return &ResourceConfig{
		ComputedKeys: c.UnknownKeys(),
		Raw:          c.Raw,
	}
}

func ProviderSatisfies(p ResourceProvider, n string) bool {
	for _, rt := range p.Resources() {
		if rt.Name == n {
			return true
		}
	}

	return false
}
