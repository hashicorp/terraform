package terraform

// ResourceProvider is an interface that must be implemented by any
// resource provider: the thing that creates and manages the resources in
// a Terraform configuration.
type ResourceProvider interface {
	// Configure configures the provider itself with the configuration
	// given. This is useful for setting things like access keys.
	//
	// Configure returns a list of warnings and a potential error.
	Configure(config map[string]interface{}) ([]string, error)

	// Resources returns all the available resource types that this provider
	// knows how to manage.
	Resources() []ResourceType
}

// ResourceType is a type of resource that a resource provider can manage.
type ResourceType struct {
	Name string
}

// ResourceProviderFactory is a function type that creates a new instance
// of a resource provider.
type ResourceProviderFactory func() (ResourceProvider, error)

func ProviderSatisfies(p ResourceProvider, n string) bool {
	for _, rt := range p.Resources() {
		if rt.Name == n {
			return true
		}
	}

	return false
}
