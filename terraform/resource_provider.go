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

	// Apply applies a diff to a specific resource and returns the new
	// resource state along with an error.
	//
	// If the resource state given has an empty ID, then a new resource
	// is expected to be created.
	//Apply(ResourceState, ResourceDiff) (ResourceState, error)

	// Diff diffs a resource versus a desired state and returns
	// a diff.
	Diff(
		ResourceState,
		map[string]interface{}) (ResourceDiff, error)
}

// ResourceDiff is the diff of a resource from some state to another.
type ResourceDiff struct {
	Attributes map[string]ResourceDiffAttribute
}

// ResourceDiffAttribute is the diff of a single attribute of a resource.
type ResourceDiffAttribute struct {
	Old         string
	New         string
	RequiresNew bool
}

// ResourceState holds the state of a resource that is used so that
// a provider can find and manage an existing resource as well as for
// storing attributes that are uesd to populate variables of child
// resources.
type ResourceState struct {
	Type       string
	ID         string
	Attributes map[string]string
	Extra      map[string]interface{}
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
