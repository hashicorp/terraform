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
func (s *State) Add(addrRaw string, raw interface{}) error {
	// Parse the address
	addr, err := ParseResourceAddress(addrRaw)
	if err != nil {
		return err
	}

	// Determine the types
	from := detectValueAddLoc(raw)
	to := detectAddrAddLoc(addr)

	// Find the function to do this
	fromMap, ok := stateAddFuncs[from]
	if !ok {
		return fmt.Errorf("invalid source to add to state: %T", raw)
	}
	f, ok := fromMap[to]
	if !ok {
		return fmt.Errorf("invalid destination: %s (%d)", addr, to)
	}

	// Call the migrator
	if err := f(s, addr, raw); err != nil {
		return err
	}

	// Prune the state
	s.prune()
	return nil
}

func stateAddFunc_Module_Module(s *State, addr *ResourceAddress, raw interface{}) error {
	src := raw.(*ModuleState)

	// If the target module exists, it is an error
	path := append([]string{"root"}, addr.Path...)
	if s.ModuleByPath(path) != nil {
		return fmt.Errorf("module target is not empty: %s", addr)
	}

	// TODO: outputs
	// TODO: dependencies

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
		if err := s.Add(addrCopy.String(), v); err != nil {
			return err
		}
	}

	return nil
}

func stateAddFunc_Resource_Resource(s *State, addr *ResourceAddress, raw interface{}) error {
	src := raw.(*ResourceState)

	// TODO: Dependencies
	// TODO: Provider?

	// Move the primary
	if src.Primary != nil {
		addrCopy := *addr
		addrCopy.InstanceType = TypePrimary
		addrCopy.InstanceTypeSet = true
		if err := s.Add(addrCopy.String(), src.Primary); err != nil {
			return err
		}
	}

	// TODO: Move all tainted
	// TODO: Move all deposed

	return nil
}

func stateAddFunc_Instance_Instance(s *State, addr *ResourceAddress, raw interface{}) error {
	src := raw.(*InstanceState).deepcopy()

	// Create the module up to this point
	path := append([]string{"root"}, addr.Path...)
	mod := s.ModuleByPath(path)
	if mod == nil {
		mod = s.AddModule(path)
	}

	// Create the resources
	resourceKey := (&ResourceStateKey{
		Name:  addr.Name,
		Type:  addr.Type,
		Index: addr.Index,
	}).String()
	resource, ok := mod.Resources[resourceKey]
	if !ok {
		resource = &ResourceState{Type: addr.Type}
		resource.init()
		mod.Resources[resourceKey] = resource
	}

	// Depending on the instance type, set it
	switch addr.InstanceType {
	case TypePrimary:
		resource.Primary = src
	default:
		return fmt.Errorf("can't move instance state to %s", addr.InstanceType)
	}

	return nil
}

// stateAddFunc is the type of function for adding an item to a state
type stateAddFunc func(s *State, addr *ResourceAddress, item interface{}) error

// stateAddFuncs has the full matrix mapping of the state adders.
var stateAddFuncs map[stateAddLoc]map[stateAddLoc]stateAddFunc

func init() {
	stateAddFuncs = map[stateAddLoc]map[stateAddLoc]stateAddFunc{
		stateAddModule: {
			stateAddModule: stateAddFunc_Module_Module,
		},
		stateAddResource: {
			stateAddResource: stateAddFunc_Resource_Resource,
		},
		stateAddInstance: {
			stateAddInstance: stateAddFunc_Instance_Instance,
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
