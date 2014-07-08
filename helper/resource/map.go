package resource

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/terraform"
)

// Map is a map of resources that are supported, and provides helpers for
// more easily implementing a ResourceProvider.
type Map struct {
	Mapping map[string]Resource
}

func (m *Map) Validate(
	t string, c *terraform.ResourceConfig) ([]string, []error) {
	r, ok := m.Mapping[t]
	if !ok {
		return nil, []error{fmt.Errorf("Unknown resource type: %s", t)}
	}

	// If there is no validator set, then it is valid
	if r.ConfigValidator == nil {
		return nil, nil
	}

	return r.ConfigValidator.Validate(c)
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

	if d.Destroy || d.RequiresNew() {
		if s.ID != "" {
			// Destroy the resource if it is created
			err := r.Destroy(s, meta)
			if err != nil {
				return s, err
			}

			s.ID = ""
		}

		// If we're only destroying, and not creating, then return now.
		// Otherwise, we continue so that we can create a new resource.
		if !d.RequiresNew() {
			return nil, nil
		}
	}

	var result *terraform.ResourceState
	var err error
	if s.ID == "" {
		result, err = r.Create(s, d, meta)
	} else {
		result, err = r.Update(s, d, meta)
	}
	if result != nil {
		if result.Attributes == nil {
			result.Attributes = make(map[string]string)
		}

		result.Attributes["id"] = result.ID
	}

	return result, err
}

// Diff peforms a diff on the proper resource type.
func (m *Map) Diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {
	r, ok := m.Mapping[s.Type]
	if !ok {
		return nil, fmt.Errorf("Unknown resource type: %s", s.Type)
	}

	return r.Diff(s, c, meta)
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
	ks := make([]string, 0, len(m.Mapping))
	for k, _ := range m.Mapping {
		ks = append(ks, k)
	}
	sort.Strings(ks)

	rs := make([]terraform.ResourceType, 0, len(m.Mapping))
	for _, k := range ks {
		rs = append(rs, terraform.ResourceType{
			Name: k,
		})
	}

	return rs
}
