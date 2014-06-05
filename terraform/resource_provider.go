package terraform

// ComputedPlaceholderKey is the configuration key given to Configure
// in a ResourceProvider that contains the value for the computed value
// placeholder when diffs are being done.
const ComputedPlaceholderKey = "tf_computed_placeholder"

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
		map[string]interface{}) (ResourceDiff, error)
}

// ResourceProviderCommonConfig contains the common configuration
// keys that are sent with every resource provider configuration.
// This can be used with something like mapstructure to extract
// the proper value.
type ResourceProviderCommonConfig struct {
	TFComputedPlaceholder string `mapstructure:"tf_computed_placeholder"`
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
