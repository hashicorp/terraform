package ngaddrs

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

type ComponentCall struct {
	Name string
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
