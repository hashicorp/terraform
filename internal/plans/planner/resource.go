package planner

import (
	"context"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	opentracing "github.com/opentracing/opentracing-go"
	tracelog "github.com/opentracing/opentracing-go/log"
	"github.com/zclconf/go-cty/cty"
)

type resource struct {
	planner *planner
	addr    addrs.AbsResource
}

func (r resource) Addr() addrs.AbsResource {
	return r.addr
}

func (r resource) ModuleInstance() moduleInstance {
	return r.planner.ModuleInstance(r.addr.Module)
}

func (r resource) InConfig() resourceInConfig {
	modAddr := r.addr.Module.Module()
	rAddr := r.addr.Resource.InModule(modAddr)
	return resourceInConfig{
		planner: r.planner,
		addr:    rAddr,
	}
}

func (r resource) repetitionProxyValue(ctx context.Context) cty.Value {
	return r.planner.DataRequest(ctx, resourceExpansionRequest{r}).(cty.Value)
}

func (r resource) InstanceKeys(ctx context.Context) map[addrs.InstanceKey]struct{} {
	proxyVal := r.repetitionProxyValue(ctx)
	return instanceKeysForRepetition(proxyVal)
}

func (r resource) Instance(key addrs.InstanceKey) resourceInstance {
	return r.planner.ResourceInstance(r.Addr().Instance(key))
}

func (r resource) Instances(ctx context.Context) map[addrs.UniqueKey]resourceInstance {
	instKeys := r.InstanceKeys(ctx)
	ret := make(map[addrs.UniqueKey]resourceInstance, len(instKeys))
	for ik := range instKeys {
		instAddr := r.addr.Instance(ik)
		ret[instAddr.UniqueKey()] = resourceInstance{
			planner: r.planner,
			addr:    instAddr,
		}
	}
	return ret
}

func (r resource) EachValueForInstance(ctx context.Context, key addrs.StringKey) cty.Value {
	proxyVal := r.repetitionProxyValue(ctx)
	return eachValueForInstance(proxyVal, key)
}

func (r resource) PlannedNewValue(ctx context.Context) cty.Value {
	proxyVal := r.repetitionProxyValue(ctx)

	return aggregateValueForInstances(ctx, proxyVal, func(ctx context.Context, key addrs.InstanceKey) cty.Value {
		inst := r.Instance(key)
		return inst.PlannedNewValue(ctx)
	})
}

func (r resource) Schema(ctx context.Context) (*configschema.Block, int64) {
	return r.InConfig().Schema(ctx)
}

type resourceExpansionRequest struct {
	rsrc resource
}

type resourceExpansionRequestKey struct {
	k addrs.UniqueKey
}

func (req resourceExpansionRequest) requestKey() interface{} {
	return moduleCallExpansionRequestKey{req.rsrc.Addr().UniqueKey()}
}

func (req resourceExpansionRequest) handleDataRequest(ctx context.Context, p *planner) interface{} {
	rsrc := req.rsrc

	span, ctx := opentracing.StartSpanFromContext(ctx, "resource.Instances")
	span.LogFields(
		tracelog.String("resource", rsrc.Addr().String()),
	)
	defer span.Finish()

	callCfg := rsrc.InConfig().Config()
	scope := rsrc.ModuleInstance().ExprScope()
	return resolveInstanceRepetition(ctx, p, callCfg.ForEach, callCfg.Count, scope)
}
