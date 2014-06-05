package terraform

// State keeps track of a snapshot state-of-the-world that Terraform
// can use to keep track of what real world resources it is actually
// managing.
type State struct {
	resources map[string]ResourceState
}

// ResourceState holds the state of a resource that is used so that
// a provider can find and manage an existing resource as well as for
// storing attributes that are uesd to populate variables of child
// resources.
//
// Attributes has attributes about the created resource that are
// queryable in interpolation: "${type.id.attr}"
//
// Extra is just extra data that a provider can return that we store
// for later, but is not exposed in any way to the user.
type ResourceState struct {
	ID         string
	Attributes map[string]string
	Extra      map[string]interface{}
}

// MergeDiff takes a ResourceDiff and merges the attributes into
// this resource state in order to generate a new state. This new
// state can be used to provide updated attribute lookups for
// variable interpolation.
func (s *ResourceState) MergeDiff(
	d map[string]*ResourceAttrDiff) ResourceState {
	result := *s
	result.Attributes = make(map[string]string)
	for k, v := range s.Attributes {
		result.Attributes[k] = v
	}
	for k, diff := range d {
		result.Attributes[k] = diff.New
	}

	return result
}
