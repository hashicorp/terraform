package ngaddrs

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

type AbsModuleInstance struct {
	Component  AbsComponent
	ModuleInst addrs.ModuleInstance
}

func (addr AbsModuleInstance) Child(name string, key addrs.InstanceKey) AbsModuleInstance {
	return AbsModuleInstance{
		Component:  addr.Component,
		ModuleInst: addr.ModuleInst.Child(name, key),
	}
}

func (addr AbsModuleInstance) Parent() AbsModuleInstance {
	if addr.ModuleInst.IsRoot() {
		panic("root module of component has no parent")
	}
	return AbsModuleInstance{
		Component:  addr.Component,
		ModuleInst: addr.ModuleInst.Parent(),
	}
}
