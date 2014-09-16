package terraform

// ResourceProvider is an interface that must be implemented by any
// resource provider: the thing that creates and manages the resources in
// a Terraform configuration.
type ResourceProvider interface {
	// Validate is called once at the beginning with the raw configuration
	// (no interpolation done) and can return a list of warnings and/or
	// errors.
	//
	// This is called once with the provider configuration only. It may not
	// be called at all if no provider configuration is given.
	//
	// This should not assume that any values of the configurations are valid.
	// The primary use case of this call is to check that required keys are
	// set.
	Validate(*ResourceConfig) ([]string, []error)

	// ValidateResource is called once at the beginning with the raw
	// configuration (no interpolation done) and can return a list of warnings
	// and/or errors.
	//
	// This is called once per resource.
	//
	// This should not assume any of the values in the resource configuration
	// are valid since it is possible they have to be interpolated still.
	// The primary use case of this call is to check that the required keys
	// are set and that the general structure is correct.
	ValidateResource(string, *ResourceConfig) ([]string, []error)

	// Configure configures the provider itself with the configuration
	// given. This is useful for setting things like access keys.
	//
	// This won't be called at all if no provider configuration is given.
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
	Apply(
		*InstanceInfo,
		*InstanceState,
		*ResourceDiff) (*InstanceState, error)

	// Diff diffs a resource versus a desired state and returns
	// a diff.
	Diff(
		*InstanceInfo,
		*InstanceState,
		*ResourceConfig) (*ResourceDiff, error)

	// Refresh refreshes a resource and updates all of its attributes
	// with the latest information.
	Refresh(*InstanceInfo, *InstanceState) (*InstanceState, error)
}

// InstanceInfo is used to hold information about the instance and/or
// resource being modified.
type InstanceInfo struct {
	// Type is the resource type of this instance
	Type string
}

// ResourceType is a type of resource that a resource provider can manage.
type ResourceType struct {
	Name string
}

// ResourceProviderFactory is a function type that creates a new instance
// of a resource provider.
type ResourceProviderFactory func() (ResourceProvider, error)

// ResourceProviderFactoryFixed is a helper that creates a
// ResourceProviderFactory that just returns some fixed provider.
func ResourceProviderFactoryFixed(p ResourceProvider) ResourceProviderFactory {
	return func() (ResourceProvider, error) {
		return p, nil
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
