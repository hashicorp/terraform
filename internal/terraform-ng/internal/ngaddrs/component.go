package ngaddrs

import (
	"fmt"
)

type ComponentCall struct {
	Name string
}

func (addr ComponentCall) String() string {
	return "component." + addr.Name
}

func (addr ComponentCall) UniqueKey() UniqueKey {
	// This type is comparable, so it can be its own unique key
	return addr
}

func (addr ComponentCall) uniqueKeySigil() {}

type Component struct {
	Name string
	Key  InstanceKey
}

func (addr Component) String() string {
	if addr.Key == nil {
		return "component." + addr.Name
	}
	return fmt.Sprintf("component.%s%s", addr.Name, addr.Key)
}

func (addr Component) UniqueKey() UniqueKey {
	// type is comparable, so it can be its own unique key type
	return addr
}

func (addr Component) uniqueKeySigil() {}

// AbsComponentCall represents a component call in a specific component group
// that has already been expanded from its corresponding call.
type AbsComponentCall = Abs[ComponentCall]

// ConfigComponentCall represents a component call in the static component
// tree, before any instance expansion.
type ConfigComponentCall = Config[ComponentCall]

// AbsComponent represents a fully-qualified component inside a component
// group, where both are fully expanded.
//
// This is the address of the object that an execution engine is most
// interested in, because it's the specific object the engine should create
// a Terraform plan for, and ultimately report output values for in order
// to make progress towards a full result.
type AbsComponent = Abs[Component]
