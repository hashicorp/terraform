package terraform

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/hashicorp/terraform/config"
)

// State keeps track of a snapshot state-of-the-world that Terraform
// can use to keep track of what real world resources it is actually
// managing.
type State struct {
	Outputs   map[string]string
	Resources map[string]*ResourceState

	once sync.Once
}

func (s *State) init() {
	s.once.Do(func() {
		s.Resources = make(map[string]*ResourceState)
	})
}

func (s *State) deepcopy() *State {
	result := new(State)
	result.init()
	if s != nil {
		for k, v := range s.Resources {
			result.Resources[k] = v
		}
	}

	return result
}

// Orphans returns a list of keys of resources that are in the State
// but aren't present in the configuration itself. Hence, these keys
// represent the state of resources that are orphans.
func (s *State) Orphans(c *config.Config) []string {
	keys := make(map[string]struct{})
	for k, _ := range s.Resources {
		keys[k] = struct{}{}
	}

	for _, r := range c.Resources {
		delete(keys, r.Id())

		// If there is only one of this instance, then we alias that
		// to the ".0" version as well so that it can count
		if r.Count == 1 {
			delete(keys, r.Id()+".0")
		}
	}

	result := make([]string, 0, len(keys))
	for k, _ := range keys {
		result = append(result, k)
	}

	return result
}

func (s *State) String() string {
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

		buf.WriteString(fmt.Sprintf("%s:\n", k))
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

// sensitiveState is used to store sensitive state information
// that should not be serialized. This is only used temporarily
// and is restored into the state.
type sensitiveState struct {
	ConnInfo map[string]*ResourceConnectionInfo

	once sync.Once
}

func (s *sensitiveState) init() {
	s.once.Do(func() {
		s.ConnInfo = make(map[string]*ResourceConnectionInfo)
	})
}

// The format byte is prefixed into the state file format so that we have
// the ability in the future to change the file format if we want for any
// reason.
const stateFormatMagic = "tfstate"
const stateFormatVersion byte = 1

// ReadState reads a state structure out of a reader in the format that
// was written by WriteState.
func ReadState(src io.Reader) (*State, error) {
	var result *State
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

// WriteState writes a state somewhere in a binary format.
func WriteState(d *State, dst io.Writer) error {
	// Write the magic bytes so we can determine the file format later
	n, err := dst.Write([]byte(stateFormatMagic))
	if err != nil {
		return err
	}
	if n != len(stateFormatMagic) {
		return errors.New("failed to write state format magic bytes")
	}

	// Write a version byte so we can iterate on version at some point
	n, err = dst.Write([]byte{stateFormatVersion})
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("failed to write state version byte")
	}

	// Prevent sensitive information from being serialized
	sensitive := &sensitiveState{}
	sensitive.init()
	for name, r := range d.Resources {
		if r.ConnInfo != nil {
			sensitive.ConnInfo[name] = r.ConnInfo
			r.ConnInfo = nil
		}
	}

	// Serialize the state
	err = gob.NewEncoder(dst).Encode(d)

	// Restore the state
	for name, info := range sensitive.ConnInfo {
		d.Resources[name].ConnInfo = info
	}

	return err
}

// ResourceConnectionInfo holds addresses, credentials and configuration
// information require to connect to a resource. This is populated
// by a provider so that provisioners can connect and run on the
// resource.
type ResourceConnectionInfo struct {
	// Type is set so that an appropriate connection can be formed.
	// As an example, for a Linux machine, the Type may be "ssh"
	Type string

	// Raw is used to store any relevant keys for the given Type
	// so that a provisioner can connect to the resource. This could
	// contain credentials or address information.
	Raw map[string]string
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
	ConnInfo *ResourceConnectionInfo

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
func (s *ResourceState) MergeDiff(d *ResourceDiff) *ResourceState {
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

// ResourceDependency maps a resource to another resource that it
// depends on to remain intact and uncorrupted.
type ResourceDependency struct {
	// ID of the resource that we depend on. This ID should map
	// directly to another ResourceState's ID.
	ID string
}
