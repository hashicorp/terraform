package planner

import (
	"context"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	opentracing "github.com/opentracing/opentracing-go"
	tracelog "github.com/opentracing/opentracing-go/log"
	"github.com/zclconf/go-cty/cty"
)

type localValue struct {
	planner *planner
	addr    addrs.AbsLocalValue
}

func (v localValue) Addr() addrs.AbsLocalValue {
	return v.addr
}

func (v localValue) IsDeclared() bool {
	return v.Config() != nil
}

func (v localValue) Config() *configs.Local {
	moduleConfig := v.planner.Config().DescendentForInstance(v.addr.Module)
	if moduleConfig == nil {
		return nil
	}

	return moduleConfig.Module.Locals[v.addr.LocalValue.Name]
}

func (v localValue) ModuleInstance() moduleInstance {
	return v.planner.ModuleInstance(v.addr.Module)
}

func (v localValue) Value(ctx context.Context) cty.Value {
	return v.planner.DataRequest(ctx, localValueRequest{v}).(cty.Value)
}

type localValueRequest struct {
	lv localValue
}

type localValueRequestKey string

func (req localValueRequest) requestKey() interface{} {
	return localValueRequestKey(req.lv.Addr().String())
}

func (req localValueRequest) handleDataRequest(ctx context.Context, p *planner) interface{} {
	lv := req.lv

	span, _ := opentracing.StartSpanFromContext(ctx, "localValue.Value")
	span.LogFields(
		tracelog.String("localValue", lv.Addr().String()),
	)
	defer span.Finish()

	config := lv.Config()
	if config == nil {
		// Reference to an undeclared value should be caught during
		// the validation step, but we'll tolerate it here to allow other
		// evaluation to complete.
		return cty.DynamicVal
	}

	scope := lv.ModuleInstance().ExprScope()
	val, diags := scope.EvalExpr(config.Expr, cty.DynamicPseudoType)
	p.AddDiagnostics(diags) // TODO: Weird that Value has this side-effect
	return val
}
