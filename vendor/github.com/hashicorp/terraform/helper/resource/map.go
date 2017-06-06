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
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	r, ok := m.Mapping[info.Type]
	if !ok {
		return nil, fmt.Errorf("Unknown resource type: %s", info.Type)
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

	var result *terraform.InstanceState
	var err error
	if s.ID == "" {
		result, err = r.Create(s, d, meta)
	} else {
		if r.Update == nil {
			return s, fmt.Errorf(
				"Resource type '%s' doesn't support update",
				info.Type)
		}

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

// Diff performs a diff on the proper resource type.
func (m *Map) Diff(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.InstanceDiff, error) {
	r, ok := m.Mapping[info.Type]
	if !ok {
		return nil, fmt.Errorf("Unknown resource type: %s", info.Type)
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
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	meta interface{}) (*terraform.InstanceState, error) {
	// If the resource isn't created, don't refresh.
	if s.ID == "" {
		return s, nil
	}

	r, ok := m.Mapping[info.Type]
	if !ok {
		return nil, fmt.Errorf("Unknown resource type: %s", info.Type)
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
