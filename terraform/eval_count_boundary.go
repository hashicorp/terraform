package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
)

// EvalCountFixZeroOneBoundaryGlobal is an EvalNode that fixes up the state
// when there is a resource count with zero/one boundary, i.e. fixing
// a resource named "aws_instance.foo" to "aws_instance.foo.0" and vice-versa.
//
// This works on the global state.
type EvalCountFixZeroOneBoundaryGlobal struct {
	Config *configs.Config
}

// TODO: test
func (n *EvalCountFixZeroOneBoundaryGlobal) Eval(ctx EvalContext) (interface{}, error) {
	// We'll temporarily lock the state to grab the modules, then work on each
	// one separately while taking a lock again for each separate resource.
	// This means that if another caller concurrently adds a module here while
	// we're working then we won't update it, but that's no worse than the
	// concurrent writer blocking for our entire fixup process and _then_
	// adding a new module, and in practice the graph node associated with
	// this eval depends on everything else in the graph anyway, so there
	// should not be concurrent writers.
	state := ctx.State().Lock()
	moduleAddrs := make([]addrs.ModuleInstance, 0, len(state.Modules))
	for _, m := range state.Modules {
		moduleAddrs = append(moduleAddrs, m.Addr)
	}
	ctx.State().Unlock()

	for _, addr := range moduleAddrs {
		cfg := n.Config.DescendentForInstance(addr)
		if cfg == nil {
			log.Printf("[WARN] Not fixing up EachModes for %s because it has no config", addr)
			continue
		}
		if err := n.fixModule(ctx, addr); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (n *EvalCountFixZeroOneBoundaryGlobal) fixModule(ctx EvalContext, moduleAddr addrs.ModuleInstance) error {
	ms := ctx.State().Module(moduleAddr)
	cfg := n.Config.DescendentForInstance(moduleAddr)
	if ms == nil {
		// Theoretically possible for a concurrent writer to delete a module
		// while we're running, but in practice the graph node that called us
		// depends on everything else in the graph and so there can never
		// be a concurrent writer.
		return fmt.Errorf("[WARN] no state found for %s while trying to fix up EachModes", moduleAddr)
	}
	if cfg == nil {
		return fmt.Errorf("[WARN] no config found for %s while trying to fix up EachModes", moduleAddr)
	}

	for _, r := range ms.Resources {
		addr := r.Addr.Absolute(moduleAddr)
		rCfg := cfg.Module.ResourceByAddr(r.Addr)
		if rCfg == nil {
			log.Printf("[WARN] Not fixing up EachModes for %s because it has no config", addr)
			continue
		}
		hasCount := rCfg.Count != nil
		fixResourceCountSetTransition(ctx, addr, hasCount)
	}

	return nil
}
