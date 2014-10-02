package terraform

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/config"
)

// The format byte is prefixed into the state file format so that we have
// the ability in the future to change the file format if we want for any
// reason.
const (
	stateFormatMagic        = "tfstate"
	stateFormatVersion byte = 1
)

// StateV1 is used to represent the state of Terraform files before
// 0.3. It is automatically upgraded to a modern State representation
// on start.
type StateV1 struct {
	Outputs   map[string]string
	Resources map[string]*ResourceStateV1
	Tainted   map[string]struct{}

	once sync.Once
}

func (s *StateV1) init() {
	s.once.Do(func() {
		if s.Resources == nil {
			s.Resources = make(map[string]*ResourceStateV1)
		}

		if s.Tainted == nil {
			s.Tainted = make(map[string]struct{})
		}
	})
}

func (s *StateV1) deepcopy() *StateV1 {
	result := new(StateV1)
	result.init()
	if s != nil {
		for k, v := range s.Resources {
			result.Resources[k] = v
		}
		for k, v := range s.Tainted {
			result.Tainted[k] = v
		}
	}

	return result
}

// prune is a helper that removes any empty IDs from the state
// and cleans it up in general.
func (s *StateV1) prune() {
	for k, v := range s.Resources {
		if v.ID == "" {
			delete(s.Resources, k)
		}
	}
}

// Orphans returns a list of keys of resources that are in the State
// but aren't present in the configuration itself. Hence, these keys
// represent the state of resources that are orphans.
func (s *StateV1) Orphans(c *config.Config) []string {
	keys := make(map[string]struct{})
	for k, _ := range s.Resources {
		keys[k] = struct{}{}
	}

	for _, r := range c.Resources {
		delete(keys, r.Id())

		for k, _ := range keys {
			if strings.HasPrefix(k, r.Id()+".") {
				delete(keys, k)
			}
		}
	}

	result := make([]string, 0, len(keys))
	for k, _ := range keys {
		result = append(result, k)
	}

	return result
}

func (s *StateV1) String() string {
	if len(s.Resources) == 0 {
		return "<no state>"
	}

	var buf bytes.Buffer

	names := make([]string, 0, len(s.Resources))
	for name, _ := range s.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, k := range names {
		rs := s.Resources[k]
		id := rs.ID
		if id == "" {
			id = "<not created>"
		}

		taintStr := ""
		if _, ok := s.Tainted[k]; ok {
			taintStr = " (tainted)"
		}

		buf.WriteString(fmt.Sprintf("%s:%s\n", k, taintStr))
		buf.WriteString(fmt.Sprintf("  ID = %s\n", id))

		attrKeys := make([]string, 0, len(rs.Attributes))
		for ak, _ := range rs.Attributes {
			if ak == "id" {
				continue
			}

			attrKeys = append(attrKeys, ak)
		}
		sort.Strings(attrKeys)

		for _, ak := range attrKeys {
			av := rs.Attributes[ak]
			buf.WriteString(fmt.Sprintf("  %s = %s\n", ak, av))
		}

		if len(rs.Dependencies) > 0 {
			buf.WriteString(fmt.Sprintf("\n  Dependencies:\n"))
			for _, dep := range rs.Dependencies {
				buf.WriteString(fmt.Sprintf("    %s\n", dep.ID))
			}
		}
	}

	if len(s.Outputs) > 0 {
		buf.WriteString("\nOutputs:\n\n")

		ks := make([]string, 0, len(s.Outputs))
		for k, _ := range s.Outputs {
			ks = append(ks, k)
		}
		sort.Strings(ks)

		for _, k := range ks {
			v := s.Outputs[k]
			buf.WriteString(fmt.Sprintf("%s = %s\n", k, v))
		}
	}

	return buf.String()
}

/// ResourceState holds the state of a resource that is used so that
// a provider can find and manage an existing resource as well as for
// storing attributes that are uesd to populate variables of child
// resources.
//
// Attributes has attributes about the created resource that are
// queryable in interpolation: "${type.id.attr}"
//
// Extra is just extra data that a provider can return that we store
// for later, but is not exposed in any way to the user.
type ResourceStateV1 struct {
	// This is filled in and managed by Terraform, and is the resource
	// type itself such as "mycloud_instance". If a resource provider sets
	// this value, it won't be persisted.
	Type string

	// The attributes below are all meant to be filled in by the
	// resource providers themselves. Documentation for each are above
	// each element.

	// A unique ID for this resource. This is opaque to Terraform
	// and is only meant as a lookup mechanism for the providers.
	ID string

	// Attributes are basic information about the resource. Any keys here
	// are accessible in variable format within Terraform configurations:
	// ${resourcetype.name.attribute}.
	Attributes map[string]string

	// ConnInfo is used for the providers to export information which is
	// used to connect to the resource for provisioning. For example,
	// this could contain SSH or WinRM credentials.
	ConnInfo map[string]string

	// Extra information that the provider can store about a resource.
	// This data is opaque, never shown to the user, and is sent back to
	// the provider as-is for whatever purpose appropriate.
	Extra map[string]interface{}

	// Dependencies are a list of things that this resource relies on
	// existing to remain intact. For example: an AWS instance might
	// depend on a subnet (which itself might depend on a VPC, and so
	// on).
	//
	// Terraform uses this information to build valid destruction
	// orders and to warn the user if they're destroying a resource that
	// another resource depends on.
	//
	// Things can be put into this list that may not be managed by
	// Terraform. If Terraform doesn't find a matching ID in the
	// overall state, then it assumes it isn't managed and doesn't
	// worry about it.
	Dependencies []ResourceDependency
}

// MergeDiff takes a ResourceDiff and merges the attributes into
// this resource state in order to generate a new state. This new
// state can be used to provide updated attribute lookups for
// variable interpolation.
//
// If the diff attribute requires computing the value, and hence
// won't be available until apply, the value is replaced with the
// computeID.
func (s *ResourceStateV1) MergeDiff(d *InstanceDiff) *ResourceStateV1 {
	var result ResourceStateV1
	if s != nil {
		result = *s
	}

	result.Attributes = make(map[string]string)
	if s != nil {
		for k, v := range s.Attributes {
			result.Attributes[k] = v
		}
	}
	if d != nil {
		for k, diff := range d.Attributes {
			if diff.NewRemoved {
				delete(result.Attributes, k)
				continue
			}
			if diff.NewComputed {
				result.Attributes[k] = config.UnknownVariableValue
				continue
			}

			result.Attributes[k] = diff.New
		}
	}

	return &result
}

func (s *ResourceStateV1) GoString() string {
	return fmt.Sprintf("*%#v", *s)
}

// ResourceDependency maps a resource to another resource that it
// depends on to remain intact and uncorrupted.
type ResourceDependency struct {
	// ID of the resource that we depend on. This ID should map
	// directly to another ResourceState's ID.
	ID string
}

// ReadStateV1 reads a state structure out of a reader in the format that
// was written by WriteState.
func ReadStateV1(src io.Reader) (*StateV1, error) {
	var result *StateV1
	var err error
	n := 0

	// Verify the magic bytes
	magic := make([]byte, len(stateFormatMagic))
	for n < len(magic) {
		n, err = src.Read(magic[n:])
		if err != nil {
			return nil, fmt.Errorf("error while reading magic bytes: %s", err)
		}
	}
	if string(magic) != stateFormatMagic {
		return nil, fmt.Errorf("not a valid state file")
	}

	// Verify the version is something we can read
	var formatByte [1]byte
	n, err = src.Read(formatByte[:])
	if err != nil {
		return nil, err
	}
	if n != len(formatByte) {
		return nil, errors.New("failed to read state version byte")
	}

	if formatByte[0] != stateFormatVersion {
		return nil, fmt.Errorf("unknown state file version: %d", formatByte[0])
	}

	// Decode
	dec := gob.NewDecoder(src)
	if err := dec.Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
