package planner

import (
	"context"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

type module struct {
	planner *planner
	addr    addrs.Module
}

func (m module) Addr() addrs.Module {
	return m.addr
}

func (m module) IsRoot() bool {
	return m.addr.IsRoot()
}

func (m module) Parent() module {
	return module{
		planner: m.planner,
		addr:    m.addr.Parent(),
	}
}

func (m module) Child(name string) module {
	return module{
		planner: m.planner,
		addr:    m.addr.Child(name),
	}
}

func (m module) CallConfig() *configs.ModuleCall {
	if m.addr.IsRoot() {
		return nil
	}
	callerAddr, callAddr := m.addr.Call()
	cfg := m.planner.Config()
	cc := cfg.Descendent(callerAddr)
	if cc == nil {
		return nil
	}
	return cc.Module.ModuleCalls[callAddr.Name]
}

func (m module) ContentConfig() *configs.Module {
	cc := m.planner.Config().Descendent(m.addr)
	if cc != nil {
		return nil
	}
	return cc.Module
}

func (m module) InstanceKeys(ctx context.Context) map[addrs.InstanceKey]struct{} {
	// TODO: Resolve count and/or for_each, and also get the parent module's
	// instances, and expand out. For now, we'll just assume no repetition.
	return map[addrs.InstanceKey]struct{}{
		addrs.NoKey: {},
	}
}

// Calls returns all of the module calls the receiver expands to as a result
// of any expansion of the module where it was configured.
//
// This doesn't consider any repetition arguments within the call itself:
// To go directly to full module instances, use method Instances.
func (m module) Calls(ctx context.Context) map[addrs.UniqueKey]moduleCall {
	if m.addr.IsRoot() {
		// There are no calls to the root module, because it's implicitly
		// "called" by the parent process when it runs Terraform.
		return nil
	}

	parentInsts := m.Parent().Instances(ctx)
	ret := make(map[addrs.UniqueKey]moduleCall, len(parentInsts))
	_, callAddr := m.Addr().Call()
	for _, pmi := range parentInsts {
		call := pmi.ChildModuleCall(callAddr)
		ret[call.Addr().UniqueKey()] = call
	}
	return ret
}

func (m module) Instances(ctx context.Context) map[addrs.UniqueKey]moduleInstance {
	ret := make(map[addrs.UniqueKey]moduleInstance)
	if m.addr.IsRoot() {
		// There's always exactly one instance of the root module.
		inst := addrs.RootModuleInstance
		ret[inst.UniqueKey()] = moduleInstance{
			planner: m.planner,
			addr:    inst,
		}
		return ret
	}

	for _, call := range m.Calls(ctx) {
		insts := call.Instances(ctx)
		for _, inst := range insts {
			ret[inst.Addr().UniqueKey()] = inst
		}
	}
	return ret
}
