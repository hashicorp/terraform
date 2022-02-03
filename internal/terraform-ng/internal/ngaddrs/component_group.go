package ngaddrs

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

type ComponentGroupCall struct {
	Name string
}

type ComponentGroupCallInstance struct {
	Name string
	Key  addrs.InstanceKey
}

type AbsComponentGroupCall struct {
	Container AbsComponentGroup
	Local     ComponentGroupCall
}

type AbsComponentGroup []ComponentGroupCallInstance

// RootComponentGroupInstance is the topmost object in the tree of component groups
// and components, containing all other components and component groups.
var RootComponentGroup AbsComponentGroup

func (addr AbsComponentGroup) IsRoot() bool {
	return len(addr) == 0
}

func (addr AbsComponentGroup) Child(name string, key addrs.InstanceKey) AbsComponentGroup {
	return append(addr[0:len(addr):len(addr)], ComponentGroupCallInstance{
		Name: name,
		Key:  key,
	})
}

func (addr AbsComponentGroup) Parent() AbsComponentGroup {
	if addr.IsRoot() {
		panic("root component group has no parent")
	}
	return addr[:len(addr)-1]
}

func (addr AbsComponentGroup) Call() AbsComponentGroupCall {
	if addr.IsRoot() {
		panic("root component group has no call")
	}
	return AbsComponentGroupCall{
		Container: addr.Parent(),
		Local:     ComponentGroupCall{Name: addr[len(addr)-1].Name},
	}
}

func (addr AbsComponentGroupCall) ComponentGroup(key addrs.InstanceKey) AbsComponentGroup {
	return addr.Container.Child(addr.Local.Name, key)
}
