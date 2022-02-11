package ngaddrs

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

type ComponentCall struct {
	Name string
}

func (addr ComponentCall) String() string {
	return "component." + addr.Name
}

func (addr ComponentCall) UniqueKey() addrs.UniqueKey {
	return addr // A ComponentCall can be its own UniqueKey, because it's ==-compatible
}

type Component struct {
	Name string
	Key  addrs.InstanceKey
}

type AbsComponentCall struct {
	Container AbsComponentGroup
	Local     ComponentCall
}

type AbsComponent struct {
	Container AbsComponentGroup
	Local     Component
}

func (addr AbsComponent) RootModule() AbsModuleInstance {
	return AbsModuleInstance{
		Component:  addr,
		ModuleInst: addrs.RootModuleInstance,
	}
}
