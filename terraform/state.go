package terraform

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/config"
)

// State keeps track of a snapshot state-of-the-world that Terraform
// can use to keep track of what real world resources it is actually
// managing.
type State struct {
	Resources map[string]*ResourceState

	once sync.Once
}

func (s *State) init() {
	s.once.Do(func() {
		s.Resources = make(map[string]*ResourceState)
	})
}

func (s *State) String() string {
	var buf bytes.Buffer

	for k, rs := range s.Resources {
		buf.WriteString(fmt.Sprintf("%s:\n", k))
		buf.WriteString(fmt.Sprintf("  ID = %s\n", rs.ID))

		for ak, av := range rs.Attributes {
			buf.WriteString(fmt.Sprintf("  %s = %s\n", ak, av))
		}
	}

	return buf.String()
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
	Type       string
	Attributes map[string]string
	Extra      map[string]interface{}
}

// MergeDiff takes a ResourceDiff and merges the attributes into
// this resource state in order to generate a new state. This new
// state can be used to provide updated attribute lookups for
// variable interpolation.
//
// If the diff attribute requires computing the value, and hence
// won't be available until apply, the value is replaced with the
// computeID.
func (s *ResourceState) MergeDiff(
	d map[string]*ResourceAttrDiff) *ResourceState {
	var result ResourceState
	if s != nil {
		result = *s
	}

	result.Attributes = make(map[string]string)
	if s != nil {
		for k, v := range s.Attributes {
			result.Attributes[k] = v
		}
	}
	for k, diff := range d {
		if diff.NewComputed {
			result.Attributes[k] = config.UnknownVariableValue
			continue
		}

		result.Attributes[k] = diff.New
	}

	return &result
}
