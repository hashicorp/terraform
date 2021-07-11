package planner

import (
	"context"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
)

type moduleInstance struct {
	planner *planner
	addr    addrs.ModuleInstance
}

func (mi moduleInstance) Addr() addrs.ModuleInstance {
	return mi.addr
}

func (mi moduleInstance) InstanceKey() addrs.InstanceKey {
	if mi.addr.IsRoot() {
		return addrs.NoKey
	}
	return mi.addr[len(mi.addr)-1].InstanceKey
}

func (mi moduleInstance) IsRoot() bool {
	return mi.addr.IsRoot()
}

func (mi moduleInstance) Parent() moduleInstance {
	return moduleInstance{
		planner: mi.planner,
		addr:    mi.addr.Parent(),
	}
}

func (m moduleInstance) Child(name string, instKey addrs.InstanceKey) moduleInstance {
	return moduleInstance{
		planner: m.planner,
		addr:    m.addr.Child(name, instKey),
	}
}

func (mi moduleInstance) ChildModuleCall(addr addrs.ModuleCall) moduleCall {
	callAddr := addrs.AbsModuleCall{
		Module: mi.Addr(),
		Call:   addr,
	}
	return moduleCall{
		planner: mi.planner,
		addr:    callAddr,
	}
}

func (mi moduleInstance) Call() moduleCall {
	callerAddr, callAddr := mi.Addr().Call()
	return moduleCall{
		planner: mi.planner,
		addr: addrs.AbsModuleCall{
			Module: callerAddr,
			Call:   callAddr,
		},
	}
}

func (mi moduleInstance) RepetitionData(ctx context.Context) instances.RepetitionData {
	modAddr := mi.Addr()
	if modAddr.IsRoot() {
		return instances.RepetitionData{}
	}
	instKey := modAddr[len(modAddr)-1].InstanceKey
	return repetitionDataForInstance(ctx, instKey, mi.Call().EachValueForInstance)
}

func (mi moduleInstance) ExprScope() *lang.Scope {
	return mi.planner.ModuleInstanceExprScope(mi.addr)
}
