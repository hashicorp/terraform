package stackaddrs

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
)

// Component is the address of a "component" block within a stack config.
type Component struct {
	Name string
}

func (Component) referenceableSigil()   {}
func (Component) inStackConfigSigil()   {}
func (Component) inStackInstanceSigil() {}

func (c Component) String() string {
	return "component." + c.Name
}

func (c Component) UniqueKey() collections.UniqueKey[Component] {
	return c
}

// A Component is its own [collections.UniqueKey].
func (Component) IsUniqueKey(Component) {}

// ConfigComponent places a [Component] in the context of a particular [Stack].
type ConfigComponent = InStackConfig[Component]

// AbsComponent places a [Component] in the context of a particular [StackInstance].
type AbsComponent = InStackInstance[Component]

// ComponentInstance is the address of a dynamic instance of a component.
type ComponentInstance struct {
	Component Component
	Key       addrs.InstanceKey
}

func (ComponentInstance) inStackConfigSigil()   {}
func (ComponentInstance) inStackInstanceSigil() {}

func (c ComponentInstance) String() string {
	if c.Key == nil {
		return c.Component.String()
	}
	return c.Component.String() + c.Key.String()
}

func (c ComponentInstance) UniqueKey() collections.UniqueKey[ComponentInstance] {
	return c
}

// A ComponentInstance is its own [collections.UniqueKey].
func (ComponentInstance) IsUniqueKey(ComponentInstance) {}

// ConfigComponentInstance places a [ComponentInstance] in the context of a
// particular [Stack].
type ConfigComponentInstance = InStackConfig[ComponentInstance]

// AbsComponentInstance places a [ComponentInstance] in the context of a
// particular [StackInstance].
type AbsComponentInstance = InStackInstance[ComponentInstance]
