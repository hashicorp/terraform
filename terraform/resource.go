package terraform

// Resource encapsulates a resource, its configuration, its provider,
// its current state, and potentially a desired diff from the state it
// wants to reach.
type Resource struct {
	Id       string
	Config   *ResourceConfig
	Diff     *ResourceDiff
	Provider ResourceProvider
	State    *ResourceState
}
