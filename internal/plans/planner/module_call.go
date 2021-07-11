package planner

import (
	"context"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	opentracing "github.com/opentracing/opentracing-go"
	tracelog "github.com/opentracing/opentracing-go/log"
	"github.com/zclconf/go-cty/cty"
)

type moduleCall struct {
	planner *planner
	addr    addrs.AbsModuleCall
}

func (mc moduleCall) Addr() addrs.AbsModuleCall {
	return mc.addr
}

func (mc moduleCall) Caller() moduleInstance {
	return mc.planner.ModuleInstance(mc.Addr().Module)
}

func (mc moduleCall) InConfig() module {
	modAddr := make(addrs.Module, len(mc.Addr().Module), len(mc.Addr().Module)+1)
	copy(modAddr, mc.Addr().Module.Module())
	modAddr = append(modAddr, mc.Addr().Call.Name)
	return mc.planner.Module(modAddr)
}

func (mc moduleCall) IsDeclared() bool {
	return mc.Config() != nil
}

func (mc moduleCall) Config() *configs.ModuleCall {
	rootCfg := mc.planner.Config()
	modCfg := rootCfg.DescendentForInstance(mc.Addr().Module)
	if modCfg == nil {
		return nil
	}
	return modCfg.Module.ModuleCalls[mc.Addr().Call.Name]
}

func (mc moduleCall) repetitionProxyValue(ctx context.Context) cty.Value {
	return mc.planner.DataRequest(ctx, moduleCallExpansionRequest{mc}).(cty.Value)
}

func (mc moduleCall) InstanceKeys(ctx context.Context) map[addrs.InstanceKey]struct{} {
	proxyVal := mc.repetitionProxyValue(ctx)
	return instanceKeysForRepetition(proxyVal)
}

func (mc moduleCall) Instances(ctx context.Context) map[addrs.UniqueKey]moduleInstance {
	instKeys := mc.InstanceKeys(ctx)
	ret := make(map[addrs.UniqueKey]moduleInstance, len(instKeys))
	callAddr := mc.Addr()
	for instKey := range instKeys {
		instAddr := callAddr.Instance(instKey)
		ret[instAddr.UniqueKey()] = mc.planner.ModuleInstance(instAddr)
	}
	return ret
}

func (mc moduleCall) EachValueForInstance(ctx context.Context, key addrs.StringKey) cty.Value {
	proxyVal := mc.repetitionProxyValue(ctx)
	return eachValueForInstance(proxyVal, key)
}

type moduleCallExpansionRequest struct {
	call moduleCall
}

type moduleCallExpansionRequestKey struct {
	k addrs.UniqueKey
}

func (req moduleCallExpansionRequest) requestKey() interface{} {
	return moduleCallExpansionRequestKey{req.call.Addr().UniqueKey()}
}

func (req moduleCallExpansionRequest) handleDataRequest(ctx context.Context, p *planner) interface{} {
	call := req.call

	span, ctx := opentracing.StartSpanFromContext(ctx, "moduleCall.Instances")
	span.LogFields(
		tracelog.String("moduleCall", call.Addr().String()),
	)
	defer span.Finish()

	callCfg := call.Config()
	scope := call.Caller().ExprScope()
	return resolveInstanceRepetition(ctx, p, callCfg.ForEach, callCfg.Count, scope)
}
