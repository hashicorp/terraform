package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mitchellh/copystructure"
)

// stateV1 keeps track of a snapshot state-of-the-world that Terraform
// can use to keep track of what real world resources it is actually
// managing.
//
// stateV1 is _only used for the purposes of backwards compatibility
// and is no longer used in Terraform.
type stateV1 struct {
	// Version is the protocol version. "1" for a StateV1.
	Version int `json:"version"`

	// Serial is incremented on any operation that modifies
	// the State file. It is used to detect potentially conflicting
	// updates.
	Serial int64 `json:"serial"`

	// Remote is used to track the metadata required to
	// pull and push state files from a remote storage endpoint.
	Remote *remoteStateV1 `json:"remote,omitempty"`

	// Modules contains all the modules in a breadth-first order
	Modules []*moduleStateV1 `json:"modules"`
}

// upgrade is used to upgrade a V1 state representation
// into a State (current) representation.
func (old *stateV1) upgrade() (*State, error) {
	remote, err := old.Remote.upgrade()
	if err != nil {
		return nil, fmt.Errorf("Error upgrading State V1: %v", err)
	}

	modules := make([]*ModuleState, len(old.Modules))
	for i, module := range old.Modules {
		upgraded, err := module.upgrade()
		if err != nil {
			return nil, fmt.Errorf("Error upgrading State V1: %v", err)
		}
		modules[i] = upgraded
	}
	if len(modules) == 0 {
		modules = nil
	}

	newState := &State{
		Version: 2,
		Serial:  old.Serial,
		Remote:  remote,
		Modules: modules,
	}

	newState.sort()

	return newState, nil
}

type remoteStateV1 struct {
	// Type controls the client we use for the remote state
	Type string `json:"type"`

	// Config is used to store arbitrary configuration that
	// is type specific
	Config map[string]string `json:"config"`
}

func (old *remoteStateV1) upgrade() (*RemoteState, error) {
	config, err := copystructure.Copy(old.Config)
	if err != nil {
		return nil, fmt.Errorf("Error upgrading RemoteState V1: %v", err)
	}

	return &RemoteState{
		Type:   old.Type,
		Config: config.(map[string]string),
	}, nil
}

func (source *RemoteState) downgradeToV1() (*remoteStateV1, bool, error) {
	config, err := copystructure.Copy(source.Config)
	if err != nil {
		return nil, false, fmt.Errorf("Error upgrading RemoteState V1: %v", err)
	}

	return &remoteStateV1{
		Type:   source.Type,
		Config: config.(map[string]string),
	}, false, nil
}

type moduleStateV1 struct {
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
	Resources map[string]*resourceStateV1 `json:"resources"`

	// Dependencies are a list of things that this module relies on
	// existing to remain intact. For example: an module may depend
	// on a VPC ID given by an aws_vpc resource.
	//
	// Terraform uses this information to build valid destruction
	// orders and to warn the user if they're destroying a module that
	// another resource depends on.
	//
	// Things can be put into this list that may not be managed by
	// Terraform. If Terraform doesn't find a matching ID in the
	// overall state, then it assumes it isn't managed and doesn't
	// worry about it.
	Dependencies []string `json:"depends_on,omitempty"`
}

func (old *moduleStateV1) upgrade() (*ModuleState, error) {
	path, err := copystructure.Copy(old.Path)
	if err != nil {
		return nil, fmt.Errorf("Error upgrading ModuleState V1: %v", err)
	}

	// Outputs needs upgrading to use the new structure
	outputs := make(map[string]*OutputState)
	for key, output := range old.Outputs {
		outputs[key] = &OutputState{
			Type:      "string",
			Value:     output,
			Sensitive: false,
		}
	}
	if len(outputs) == 0 {
		outputs = nil
	}

	resources := make(map[string]*ResourceState)
	for key, oldResource := range old.Resources {
		upgraded, err := oldResource.upgrade()
		if err != nil {
			return nil, fmt.Errorf("Error upgrading ModuleState V1: %v", err)
		}
		resources[key] = upgraded
	}
	if len(resources) == 0 {
		resources = nil
	}

	dependencies, err := copystructure.Copy(old.Dependencies)
	if err != nil {
		return nil, fmt.Errorf("Error upgrading ModuleState V1: %v", err)
	}

	return &ModuleState{
		Path:         path.([]string),
		Outputs:      outputs,
		Resources:    resources,
		Dependencies: dependencies.([]string),
	}, nil
}

func (source *ModuleState) downgradeToV1() (*moduleStateV1, bool, error) {
	conversionWasLossy := false

	path, err := copystructure.Copy(source.Path)
	if err != nil {
		return nil, false, fmt.Errorf("Error downgrading ModuleState to V1: %v", err)
	}

	dependencies, err := copystructure.Copy(source.Dependencies)
	if err != nil {
		return nil, false, fmt.Errorf("Error downgrading ModuleState to V1: %v", err)
	}

	resources := make(map[string]*resourceStateV1)
	for key, oldResource := range source.Resources {
		downgraded, lossy, err := oldResource.downgradeToV1()
		if err != nil {
			return nil, false, fmt.Errorf("Error downgrading ModuleState to V1: %v", err)
		}
		if lossy {
			conversionWasLossy = true
		}
		resources[key] = downgraded
	}
	if len(resources) == 0 {
		resources = nil
	}

	outputs := make(map[string]string)
	for key, newOutput := range source.Outputs {
		if newOutput.Type != "string" {
			conversionWasLossy = true
			continue
		}
		if newOutput.Sensitive {
			conversionWasLossy = true
		}

		if targetOutput, ok := newOutput.Value.(string); ok {
			outputs[key] = targetOutput
		} else {
			conversionWasLossy = true
		}
	}

	return &moduleStateV1{
		Path:         path.([]string),
		Outputs:      outputs,
		Resources:    resources,
		Dependencies: dependencies.([]string),
	}, conversionWasLossy, nil
}

type resourceStateV1 struct {
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

	// Primary is the current active instance for this resource.
	// It can be replaced but only after a successful creation.
	// This is the instances on which providers will act.
	Primary *instanceStateV1 `json:"primary"`

	// Tainted is used to track any underlying instances that
	// have been created but are in a bad or unknown state and
	// need to be cleaned up subsequently.  In the
	// standard case, there is only at most a single instance.
	// However, in pathological cases, it is possible for the number
	// of instances to accumulate.
	Tainted []*instanceStateV1 `json:"tainted,omitempty"`

	// Deposed is used in the mechanics of CreateBeforeDestroy: the existing
	// Primary is Deposed to get it out of the way for the replacement Primary to
	// be created by Apply. If the replacement Primary creates successfully, the
	// Deposed instance is cleaned up. If there were problems creating the
	// replacement, the instance remains in the Deposed list so it can be
	// destroyed in a future run. Functionally, Deposed instances are very
	// similar to Tainted instances in that Terraform is only tracking them in
	// order to remember to destroy them.
	Deposed []*instanceStateV1 `json:"deposed,omitempty"`

	// Provider is used when a resource is connected to a provider with an alias.
	// If this string is empty, the resource is connected to the default provider,
	// e.g. "aws_instance" goes with the "aws" provider.
	// If the resource block contained a "provider" key, that value will be set here.
	Provider string `json:"provider,omitempty"`
}

func (old *resourceStateV1) upgrade() (*ResourceState, error) {
	dependencies, err := copystructure.Copy(old.Dependencies)
	if err != nil {
		return nil, fmt.Errorf("Error upgrading ResourceState V1: %v", err)
	}

	primary, err := old.Primary.upgrade()
	if err != nil {
		return nil, fmt.Errorf("Error upgrading ResourceState V1: %v", err)
	}

	tainted := make([]*InstanceState, len(old.Tainted))
	for i, v := range old.Tainted {
		upgraded, err := v.upgrade()
		if err != nil {
			return nil, fmt.Errorf("Error upgrading ResourceState V1: %v", err)
		}
		tainted[i] = upgraded
	}
	if len(tainted) == 0 {
		tainted = nil
	}

	deposed := make([]*InstanceState, len(old.Deposed))
	for i, v := range old.Deposed {
		upgraded, err := v.upgrade()
		if err != nil {
			return nil, fmt.Errorf("Error upgrading ResourceState V1: %v", err)
		}
		deposed[i] = upgraded
	}
	if len(deposed) == 0 {
		deposed = nil
	}

	return &ResourceState{
		Type:         old.Type,
		Dependencies: dependencies.([]string),
		Primary:      primary,
		Tainted:      tainted,
		Deposed:      deposed,
		Provider:     old.Provider,
	}, nil
}

func (source *ResourceState) downgradeToV1() (*resourceStateV1, bool, error) {
	conversionWasLossy := false

	dependencies, err := copystructure.Copy(source.Dependencies)
	if err != nil {
		return nil, false, fmt.Errorf("Error downgrading ResourceState to V1: %v", err)
	}

	primary, primaryLossy, err := source.Primary.downgradeToV1()
	if err != nil {
		return nil, false, fmt.Errorf("Error downgrading ResourceState to V1: %v", err)
	}
	if primaryLossy {
		conversionWasLossy = true
	}

	tainted := make([]*instanceStateV1, len(source.Tainted))
	for i, v := range source.Tainted {
		downgraded, taintedLossy, err := v.downgradeToV1()
		if err != nil {
			return nil, false, fmt.Errorf("Error downgrading ResourceState to V1: %v", err)
		}
		if taintedLossy {
			conversionWasLossy = true
		}
		tainted[i] = downgraded
	}
	if len(tainted) == 0 {
		tainted = nil
	}

	deposed := make([]*instanceStateV1, len(source.Deposed))
	for i, v := range source.Deposed {
		downgraded, deposedLossy, err := v.downgradeToV1()
		if err != nil {
			return nil, false, fmt.Errorf("Error downgrading ResourceState to V1: %v", err)
		}
		if deposedLossy {
			conversionWasLossy = true
		}
		deposed[i] = downgraded
	}
	if len(deposed) == 0 {
		deposed = nil
	}

	return &resourceStateV1{
		Type:         source.Type,
		Dependencies: dependencies.([]string),
		Primary:      primary,
		Tainted:      tainted,
		Deposed:      deposed,
		Provider:     source.Provider,
	}, conversionWasLossy, nil
}

type instanceStateV1 struct {
	// A unique ID for this resource. This is opaque to Terraform
	// and is only meant as a lookup mechanism for the providers.
	ID string `json:"id"`

	// Attributes are basic information about the resource. Any keys here
	// are accessible in variable format within Terraform configurations:
	// ${resourcetype.name.attribute}.
	Attributes map[string]string `json:"attributes,omitempty"`

	// Meta is a simple K/V map that is persisted to the State but otherwise
	// ignored by Terraform core. It's meant to be used for accounting by
	// external client code.
	Meta map[string]string `json:"meta,omitempty"`
}

func (old *instanceStateV1) upgrade() (*InstanceState, error) {
	attributes, err := copystructure.Copy(old.Attributes)
	if err != nil {
		return nil, fmt.Errorf("Error upgrading InstanceState V1: %v", err)
	}
	meta, err := copystructure.Copy(old.Meta)
	if err != nil {
		return nil, fmt.Errorf("Error upgrading InstanceState V1: %v", err)
	}

	return &InstanceState{
		ID:         old.ID,
		Attributes: attributes.(map[string]string),
		Meta:       meta.(map[string]string),
	}, nil
}

func (source *InstanceState) downgradeToV1() (*instanceStateV1, bool, error) {
	attributes, err := copystructure.Copy(source.Attributes)
	if err != nil {
		return nil, false, fmt.Errorf("Error downgrading InstanceState to V1: %v", err)
	}

	meta, err := copystructure.Copy(source.Meta)
	if err != nil {
		return nil, false, fmt.Errorf("Error downgrading InstanceState to V1: %v", err)
	}

	return &instanceStateV1{
		ID:         source.ID,
		Attributes: attributes.(map[string]string),
		Meta:       meta.(map[string]string),
	}, false, nil
}

// downgradeToV1 will downgrade a state from the current revision to
// version 1. It will track loss of information and return true for the
// second parameter if loss occurred. If it is not possible to downgrade
// the state, an error will be returned. Losing information however is
// not considered an error and should be checked explicitly if it is
// important in a given context.
func (source *State) downgradeToV1() (*stateV1, bool, error) {
	downgradeWasLossy := false

	var err error
	var remote *remoteStateV1

	if source.Remote != nil {
		var lossy bool
		remote, lossy, err = source.Remote.downgradeToV1()
		if err != nil {
			return nil, false, fmt.Errorf("Error downgrading RemoveState to V1: %v", err)
		}
		if lossy {
			downgradeWasLossy = true
		}
	}

	modules := make([]*moduleStateV1, len(source.Modules))
	for i, mod := range source.Modules {
		downgraded, lossy, err := mod.downgradeToV1()
		if err != nil {
			return nil, false, fmt.Errorf("Error downgrading RemoveState to V1: %v", err)
		}
		if lossy {
			downgradeWasLossy = true
		}
		modules[i] = downgraded
	}
	if len(modules) == 0 {
		modules = nil
	}

	target := &stateV1{
		Version: 1,
		Serial:  source.Serial,
		Remote:  remote,
		Modules: modules,
	}
	return target, downgradeWasLossy, nil
}

type moduleStateV1Sort []*moduleStateV1

func (s moduleStateV1Sort) Len() int {
	return len(s)
}

func (s moduleStateV1Sort) Less(i, j int) bool {
	a := s[i]
	b := s[j]

	// If the lengths are different, then the shorter one always wins
	if len(a.Path) != len(b.Path) {
		return len(a.Path) < len(b.Path)
	}

	// Otherwise, compare lexically
	return strings.Join(a.Path, ".") < strings.Join(b.Path, ".")
}

func (s moduleStateV1Sort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s *stateV1) sort() {
	sort.Sort(moduleStateV1Sort(s.Modules))

	// Allow modules to be sorted
	for _, m := range s.Modules {
		m.sort()
	}
}

func (r *resourceStateV1) sort() {
	sort.Strings(r.Dependencies)
}

func (m *moduleStateV1) sort() {
	for _, v := range m.Resources {
		v.sort()
	}
}

// WriteState writes a state somewhere in a binary format.
func (d *stateV1) WriteState(dst io.Writer) error {
	// Make sure it is sorted
	d.sort()

	// Ensure the version is set
	d.Version = 1

	// Encode the data in a human-friendly way
	data, err := json.MarshalIndent(d, "", "    ")
	if err != nil {
		return fmt.Errorf("Failed to encode state: %s", err)
	}

	// We append a newline to the data because MarshalIndent doesn't
	data = append(data, '\n')

	// Write the data out to the dst
	if _, err := io.Copy(dst, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("Failed to write state: %v", err)
	}

	return nil
}
