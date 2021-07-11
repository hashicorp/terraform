package planner

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	opentracing "github.com/opentracing/opentracing-go"
	tracelog "github.com/opentracing/opentracing-go/log"
)

func Plan(ctx context.Context, opts *Options, config *configs.Config, prevRunState *states.State, providerFactory func(addrs.Provider) (providers.Interface, error)) (*plans.Plan, tfdiags.Diagnostics) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Plan")
	defer span.Finish()

	//coalescedSpan, coalescedCtx := opentracing.StartSpanFromContext(ctx, "coalesced")
	//defer coalescedSpan.Finish()
	coalescedCtx := ctx

	pnr := planner{
		opts:            opts,
		config:          config,
		prevRunState:    prevRunState,
		providerFactory: providerFactory,

		coalescedCtx: coalescedCtx,
	}
	pnr.agglomerator = &agglomerator{planner: &pnr}

	plan := pnr.Plan(ctx)
	pnr.diags.Sort()
	return plan, pnr.diags
}

func (p *planner) Plan(ctx context.Context) *plans.Plan {
	changes := plans.NewChanges()
	priorState := states.NewState()

	// We'll walk the configuration and find all of the configured resources
	// and resource instances and, if they are targeted, collect planned
	// changes for each of them.
	p.planModule(ctx, p.config, changes.SyncWrapper())

	// We also need to start planning any resource instances in the state
	// that didn't appear in the configuration, which will typically
	// (but not necessarily) lead to adding a "destroy" action.
	// TODO: actually do that

	log.Printf("[TRACE] Planning is complete")

	return &plans.Plan{
		PrevRunState: p.prevRunState,
		PriorState:   priorState,
		Changes:      changes,

		// TODO: and all of the other plan fields
	}
}

func (p *planner) planModule(ctx context.Context, modCfg *configs.Config, changes *plans.ChangesSync) {
	log.Printf("[TRACE] starting to plan %s", modCfg.Path)
	var wg sync.WaitGroup
	wg.Add(len(modCfg.Module.ManagedResources) + len(modCfg.Module.DataResources) + len(modCfg.Children))

	span, ctx := opentracing.StartSpanFromContext(ctx, "planModule")
	defer span.Finish()
	span.LogFields(
		tracelog.String("module", modCfg.Path.String()),
	)

	for _, rc := range modCfg.Module.ManagedResources {
		r := p.ResourceInConfig(addrs.ConfigResource{
			Module:   modCfg.Path,
			Resource: rc.Addr(),
		})
		go func(r resourceInConfig) {
			p.planResource(ctx, r, changes)
			wg.Done()
		}(r)
	}

	// We also need to start planning the data resources.
	// TODO: actually do that

	// We also need to start planning the root module outputs.
	// TODO: actually do that

	for _, cc := range modCfg.Children {
		go func(cc *configs.Config) {
			p.planModule(ctx, cc, changes)
			wg.Done()
		}(cc)
	}

	wg.Wait()
}

func (p *planner) planResource(ctx context.Context, r resourceInConfig, changes *plans.ChangesSync) {
	log.Printf("[TRACE] starting to plan resource %s", r.Addr())

	if !r.IsTargeted() {
		return
	}

	span, ctx := opentracing.StartSpanFromContext(ctx, "planResource")
	defer span.Finish()
	span.LogFields(
		tracelog.String("resource", r.Addr().String()),
	)

	insts := r.Instances(ctx)
	var wg sync.WaitGroup
	wg.Add(len(insts))
	for _, inst := range insts {
		go func(inst resourceInstance) {
			p.planResourceInstance(ctx, inst, changes)
			wg.Done()
		}(inst)
	}

	wg.Wait()
}

func (p *planner) planResourceInstance(ctx context.Context, inst resourceInstance, changes *plans.ChangesSync) {
	log.Printf("[TRACE] starting to plan resource instance %s", inst.Addr())

	if !inst.IsTargeted() {
		return
	}

	span, ctx := opentracing.StartSpanFromContext(ctx, "planResourceInstance")
	defer span.Finish()
	span.LogFields(
		tracelog.String("resourceInstance", inst.Addr().String()),
	)

	change := inst.PlannedChange(ctx)
	if change != nil {
		schema, _ := inst.Resource().Schema(ctx)
		if schema == nil {
			p.AddDiagnostics(fmt.Errorf("no schema for %s", inst.Addr()))
			return
		}

		changeSrc, err := change.Encode(schema.ImpliedType())
		if err != nil {
			p.AddDiagnostics(err)
			return
		}

		changes.AppendResourceInstanceChange(changeSrc)
	}

}
