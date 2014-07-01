package terraform

import (
	"fmt"
)

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

// Vars returns the mapping of variables that should be replaced in
// configuration based on the attributes of this resource.
func (r *Resource) Vars() map[string]string {
	if r.State == nil {
		return nil
	}

	vars := make(map[string]string)
	for ak, av := range r.State.Attributes {
		vars[fmt.Sprintf("%s.%s", r.Id, ak)] = av
	}

	return vars
}
