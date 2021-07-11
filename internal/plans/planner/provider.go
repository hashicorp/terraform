package planner

import (
	"context"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	opentracing "github.com/opentracing/opentracing-go"
	tracelog "github.com/opentracing/opentracing-go/log"
)

type provider struct {
	planner *planner
	addr    addrs.Provider
}

func (p provider) Addr() addrs.Provider {
	return p.addr
}

func (p provider) Config(moduleAddr addrs.Module, alias string) providerConfig {
	configAddr := addrs.AbsProviderConfig{
		Module:   moduleAddr,
		Provider: p.addr,
		Alias:    alias,
	}
	return providerConfig{
		planner: p.planner,
		addr:    configAddr,
	}
}

func (p provider) Instance(ctx context.Context) (providers.Interface, error) {
	return p.planner.UnconfiguredProviderInstance(p)
}

func (p provider) Schema(ctx context.Context) (providers.GetProviderSchemaResponse, error) {
	resp := p.planner.DataRequest(ctx, providerSchemaRequest{p.Addr()}).(providerSchemaResponse)
	return resp.resp, resp.err
}

// providerSchemaRequest is a dataRequest for the schema of a particular
// provider.
type providerSchemaRequest struct {
	addr addrs.Provider
}

func (req providerSchemaRequest) requestKey() interface{} {
	// providerSchemaRequest is comparable, so it can be its own key
	return req
}

func (req providerSchemaRequest) handleDataRequest(ctx context.Context, p *planner) interface{} {
	pdr := p.Provider(req.addr)
	span, _ := opentracing.StartSpanFromContext(ctx, "provider.Schema")
	span.LogFields(
		tracelog.String("provider", req.addr.String()),
	)
	defer span.Finish()

	inst, err := p.UnconfiguredProviderInstance(pdr)
	if err != nil {
		return providerSchemaResponse{err: err}
	}
	defer inst.Close()

	return providerSchemaResponse{resp: inst.GetProviderSchema()}
}

type providerSchemaResponse struct {
	resp providers.GetProviderSchemaResponse
	err  error
}
