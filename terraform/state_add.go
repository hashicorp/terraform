package terraform

import (
	"fmt"
)

// Add adds the item in the state at the given address.
//
// The item can be a ModuleState, ResourceState, or InstanceState. Depending
// on the item type, the address may or may not be valid. For example, a
// module cannot be moved to a resource address, however a resource can be
// moved to a module address (it retains the same name, under that resource).
//
// The full semantics of Add:
//
//                         ┌───────────────────────┬───────────────────────┬───────────────────────┐
//                         │    Module Address     │   Resource Address    │   Instance Address    │
// ┌───────────────────────┼───────────────────────┼───────────────────────┼───────────────────────┤
// │      ModuleState      │           ✓           │           x           │           x           │
// ├───────────────────────┼───────────────────────┼───────────────────────┼───────────────────────┤
// │     ResourceState     │           ✓           │           ✓           │        maybe*         │
// ├───────────────────────┼───────────────────────┼───────────────────────┼───────────────────────┤
// │    Instance State     │           ✓           │           ✓           │           ✓           │
// └───────────────────────┴───────────────────────┴───────────────────────┴───────────────────────┘
//
// *maybe - Resources can be added at an instance address only if the resource
//          represents a single instance (primary). Example:
//          "aws_instance.foo" can be moved to "aws_instance.bar.tainted"
//
func (s *State) Add(fromAddrRaw string, toAddrRaw string, raw interface{}) error {
	// Parse the address
	toAddr, err := ParseResourceAddress(toAddrRaw)
	if err != nil {
		return err
	}

	// Parse the from address
	fromAddr, err := ParseResourceAddress(fromAddrRaw)
	if err != nil {
		return err
	}

	// Determine the types
	from := detectValueAddLoc(raw)
	to := detectAddrAddLoc(toAddr)

	// Find the function to do this
	fromMap, ok := stateAddFuncs[from]
	if !ok {
		return fmt.Errorf("invalid source to add to state: %T", raw)
	}
	f, ok := fromMap[to]
	if !ok {
		return fmt.Errorf("invalid destination: %s (%d)", toAddr, to)
	}

	// Call the migrator
	if err := f(s, fromAddr, toAddr, raw); err != nil {
		return err
	}

	// Prune the state
	s.prune()
	return nil
}

func stateAddFunc_Module_Module(s *State, fromAddr, addr *ResourceAddress, raw interface{}) error {
	src := raw.(*ModuleState).deepcopy()

	// If the target module exists, it is an error
	path := append([]string{"root"}, addr.Path...)
	if s.ModuleByPath(path) != nil {
		return fmt.Errorf("module target is not empty: %s", addr)
	}

	// Create it and copy our outputs and dependencies
	mod := s.AddModule(path)
	mod.Outputs = src.Outputs
	mod.Dependencies = src.Dependencies

	// Go through the resources perform an add for each of those
	for k, v := range src.Resources {
		resourceKey, err := ParseResourceStateKey(k)
		if err != nil {
			return err
		}

		// Update the resource address for this
		addrCopy := *addr
		addrCopy.Type = resourceKey.Type
		addrCopy.Name = resourceKey.Name
		addrCopy.Index = resourceKey.Index

		// Perform an add
		if err := s.Add(fromAddr.String(), addrCopy.String(), v); err != nil {
			return err
		}
	}

	return nil
}

func stateAddFunc_Resource_Module(
	s *State, from, to *ResourceAddress, raw interface{}) error {
	// Build the more specific to addr
	addr := *to
	addr.Type = from.Type
	addr.Name = from.Name

	return s.Add(from.String(), addr.String(), raw)
}

func stateAddFunc_Resource_Resource(s *State, fromAddr, addr *ResourceAddress, raw interface{}) error {
	src := raw.(*ResourceState).deepcopy()

	// Initialize the resource
	resourceRaw, exists := stateAddInitAddr(s, addr)
	if exists {
		return fmt.Errorf("resource exists and not empty: %s", addr)
	}
	resource := resourceRaw.(*ResourceState)
	resource.Type = src.Type
	resource.Dependencies = src.Dependencies
	resource.Provider = src.Provider

	// Move the primary
	if src.Primary != nil {
		addrCopy := *addr
		addrCopy.InstanceType = TypePrimary
		addrCopy.InstanceTypeSet = true
		if err := s.Add(fromAddr.String(), addrCopy.String(), src.Primary); err != nil {
			return err
		}
	}

	// Move all deposed
	if len(src.Deposed) > 0 {
		resource.Deposed = src.Deposed
	}

	return nil
}

func stateAddFunc_Instance_Instance(s *State, fromAddr, addr *ResourceAddress, raw interface{}) error {
	src := raw.(*InstanceState).DeepCopy()

	// Create the instance
	instanceRaw, _ := stateAddInitAddr(s, addr)
	instance := instanceRaw.(*InstanceState)

	// Set it
	*instance = *src

	return nil
}

func stateAddFunc_Instance_Module(
	s *State, from, to *ResourceAddress, raw interface{}) error {
	addr := *to
	addr.Type = from.Type
	addr.Name = from.Name

	return s.Add(from.String(), addr.String(), raw)
}

func stateAddFunc_Instance_Resource(
	s *State, from, to *ResourceAddress, raw interface{}) error {
	addr := *to
	addr.InstanceType = TypePrimary
	addr.InstanceTypeSet = true

	return s.Add(from.String(), addr.String(), raw)
}

// stateAddFunc is the type of function for adding an item to a state
type stateAddFunc func(s *State, from, to *ResourceAddress, item interface{}) error

// stateAddFuncs has the full matrix mapping of the state adders.
var stateAddFuncs map[stateAddLoc]map[stateAddLoc]stateAddFunc

func init() {
	stateAddFuncs = map[stateAddLoc]map[stateAddLoc]stateAddFunc{
		stateAddModule: {
			stateAddModule: stateAddFunc_Module_Module,
		},
		stateAddResource: {
			stateAddModule:   stateAddFunc_Resource_Module,
			stateAddResource: stateAddFunc_Resource_Resource,
		},
		stateAddInstance: {
			stateAddInstance: stateAddFunc_Instance_Instance,
			stateAddModule:   stateAddFunc_Instance_Module,
			stateAddResource: stateAddFunc_Instance_Resource,
		},
	}
}

// stateAddLoc is an enum to represent the location where state is being
// moved from/to. We use this for quick lookups in a function map.
type stateAddLoc uint

const (
	stateAddInvalid stateAddLoc = iota
	stateAddModule
	stateAddResource
	stateAddInstance
)

// detectAddrAddLoc detects the state type for the given address. This
// function is specifically not unit tested since we consider the State.Add
// functionality to be comprehensive enough to cover this.
func detectAddrAddLoc(addr *ResourceAddress) stateAddLoc {
	if addr.Name == "" {
		return stateAddModule
	}

	if !addr.InstanceTypeSet {
		return stateAddResource
	}

	return stateAddInstance
}

// detectValueAddLoc determines the stateAddLoc value from the raw value
// that is some State structure.
func detectValueAddLoc(raw interface{}) stateAddLoc {
	switch raw.(type) {
	case *ModuleState:
		return stateAddModule
	case *ResourceState:
		return stateAddResource
	case *InstanceState:
		return stateAddInstance
	default:
		return stateAddInvalid
	}
}

// stateAddInitAddr takes a ResourceAddress and creates the non-existing
// resources up to that point, returning the empty (or existing) interface
// at that address.
func stateAddInitAddr(s *State, addr *ResourceAddress) (interface{}, bool) {
	addType := detectAddrAddLoc(addr)

	// Get the module
	path := append([]string{"root"}, addr.Path...)
	exists := true
	mod := s.ModuleByPath(path)
	if mod == nil {
		mod = s.AddModule(path)
		exists = false
	}
	if addType == stateAddModule {
		return mod, exists
	}

	// Add the resource
	resourceKey := (&ResourceStateKey{
		Name:  addr.Name,
		Type:  addr.Type,
		Index: addr.Index,
	}).String()
	exists = true
	resource, ok := mod.Resources[resourceKey]
	if !ok {
		resource = &ResourceState{Type: addr.Type}
		resource.init()
		mod.Resources[resourceKey] = resource
		exists = false
	}
	if addType == stateAddResource {
		return resource, exists
	}

	// Get the instance
	exists = true
	instance := &InstanceState{}
	switch addr.InstanceType {
	case TypePrimary, TypeTainted:
		if v := resource.Primary; v != nil {
			instance = resource.Primary
		} else {
			exists = false
		}
	case TypeDeposed:
		idx := addr.Index
		if addr.Index < 0 {
			idx = 0
		}
		if len(resource.Deposed) > idx {
			instance = resource.Deposed[idx]
		} else {
			resource.Deposed = append(resource.Deposed, instance)
			exists = false
		}
	}

	return instance, exists
}
