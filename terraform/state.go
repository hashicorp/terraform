package terraform

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
)

// State keeps track of a snapshot state-of-the-world that Terraform
// can use to keep track of what real world resources it is actually
// managing. This is the latest format as of Terraform 0.3
type State struct {
	// Version is the protocol version. Currently only "1".
	Version int `json:"version"`

	// Serial is incremented on any operation that modifies
	// the State file. It is used to detect potentially conflicting
	// updates.
	Serial int64 `json:"serial"`

	// Modules contains all the modules in a breadth-first order
	Modules []*ModuleState `json:"modules"`
}

// ModuleState is used to track all the state relevant to a single
// module. Previous to Terraform 0.3, all state belonged to the "root"
// module.
type ModuleState struct {
	// Path is the import path from the root module. Modules imports are
	// always disjoint, so the path represents amodule tree
	Path []string `json:"path"`

	// Outputs declared by the module and maintained for each module
	// even though only the root module technically needs to be kept.
	// This allows operators to inspect values at the boundaries.
	Outputs map[string]string `json:"outputs"`

	// Resources is a mapping of the logically named resource to
	// the state of the resource. Each resource may actually have
	// N instances underneath, although a user only needs to think
	// about the 1:1 case.
	Resources map[string]*ResourceState `json:"resources"`
}

// ResourceState holds the state of a resource that is used so that
// a provider can find and manage an existing resource as well as for
// storing attributes that are used to populate variables of child
// resources.
//
// Attributes has attributes about the created resource that are
// queryable in interpolation: "${type.id.attr}"
//
// Extra is just extra data that a provider can return that we store
// for later, but is not exposed in any way to the user.
//
type ResourceState struct {
	// This is filled in and managed by Terraform, and is the resource
	// type itself such as "mycloud_instance". If a resource provider sets
	// this value, it won't be persisted.
	Type string `json:"type"`

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
	Dependencies []string `json:"depends_on,omitempty"`

	// Instances is used to track all of the underlying instances
	// have been created as part of this logical resource. In the
	// standard case, there is only a single underlying instance.
	// However, in pathological cases, it is possible for the number
	// of instances to accumulate. The first instance in the list is
	// the "primary" and the others should be removed on subsequent
	// apply operations.
	Instances []*InstanceState `json:"instances"`
}

// InstanceState is used to track the unique state information belonging
// to a given instance.
type InstanceState struct {
	// A unique ID for this resource. This is opaque to Terraform
	// and is only meant as a lookup mechanism for the providers.
	ID string `json:"id"`

	// Tainted is used to mark a resource as existing but being in
	// an unknown or errored state. Hence, it is 'tainted' and should
	// be destroyed and replaced on the next fun.
	Tainted bool `json:"tainted,omitempty"`

	// Attributes are basic information about the resource. Any keys here
	// are accessible in variable format within Terraform configurations:
	// ${resourcetype.name.attribute}.
	Attributes map[string]string `json:"attributes,omitempty"`

	// Ephemeral is used to store any state associated with this instance
	// that is necessary for the Terraform run to complete, but is not
	// persisted to a state file.
	Ephemeral EphemeralState `json:"-"`
}

// EphemeralState is used for transient state that is only kept in-memory
type EphemeralState struct {
	// ConnInfo is used for the providers to export information which is
	// used to connect to the resource for provisioning. For example,
	// this could contain SSH or WinRM credentials.
	ConnInfo map[string]string `json:"-"`
}

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
