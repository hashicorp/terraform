package resource

import (
	"fmt"

	"github.com/hashicorp/terraform/terraform"
)

// Map is a map of resources that are supported, and provides helpers for
// more easily implementing a ResourceProvider.
type Map struct {
	Mapping map[string]Resource
}

// Apply performs a create or update depending on the diff, and calls
// the proper function on the matching Resource.
func (m *Map) Apply(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	r, ok := m.Mapping[s.Type]
	if !ok {
		return nil, fmt.Errorf("Unknown resource type: %s", s.Type)
	}

	if s.ID == "" {
		return r.Create(s, d, meta)
	} else {
		panic("update no implemented yet")
		//return r.Update(s, d, meta)
	}
}

// Refresh performs a Refresh on the proper resource type.
//
// Refresh on the Resource won't be called if the state represents a
// non-created resource (ID is blank).
//
// An error is returned if the resource isn't registered.
func (m *Map) Refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	// If the resource isn't created, don't refresh.
	if s.ID == "" {
		return s, nil
	}

	r, ok := m.Mapping[s.Type]
	if !ok {
		return nil, fmt.Errorf("Unknown resource type: %s", s.Type)
	}

	return r.Refresh(s, meta)
}

// Resources returns all the resources that are supported by this
// resource map and can be used to satisfy the Resources method of
// a ResourceProvider.
func (m *Map) Resources() []terraform.ResourceType {
	rs := make([]terraform.ResourceType, 0, len(m.Mapping))
	for k, _ := range m.Mapping {
		rs = append(rs, terraform.ResourceType{
			Name: k,
		})
	}

	return rs
}
