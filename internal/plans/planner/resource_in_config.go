package planner

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
)

type resourceInConfig struct {
	planner *planner
	addr    addrs.ConfigResource
}

func (r resourceInConfig) Addr() addrs.ConfigResource {
	return r.addr
}

func (r resourceInConfig) Module() module {
	return module{
		planner: r.planner,
		addr:    r.addr.Module,
	}
}

func (r resourceInConfig) ProviderConfig() providerConfig {
	module := r.Module()

	config := r.Config()
	if config == nil {
		panic(fmt.Sprintf("no ProviderConfig for undeclared resource %s", r.Addr()))
	}

	providerConfigAddr := r.planner.Config().ResolveAbsProviderAddr(config.ProviderConfigAddr(), module.Addr())
	return providerConfig{
		planner: r.planner,
		addr:    providerConfigAddr,
	}
}

func (r resourceInConfig) IsDeclared() bool {
	return r.Config() != nil
}

func (r resourceInConfig) Config() *configs.Resource {
	cfg := r.planner.Config()
	mc := cfg.Descendent(r.addr.Module)
	if mc == nil {
		return nil
	}
	return mc.Module.ResourceByAddr(r.addr.Resource)
}

func (r resourceInConfig) IsTargeted() bool {
	targetAddrs := r.planner.TargetAddrs()
	if len(targetAddrs) == 0 {
		return true // everything is included by default
	}
	for _, targetAddr := range targetAddrs {
		if targetAddr.TargetContains(r.addr) {
			return true
		}
	}
	return false
}

func (r resourceInConfig) PerModuleInstance(ctx context.Context) map[addrs.UniqueKey]resource {
	ret := make(map[addrs.UniqueKey]resource)
	for _, mi := range r.Module().Instances(ctx) {
		rAddr := r.addr.Resource.Absolute(mi.Addr())
		ret[rAddr.Module.UniqueKey()] = resource{
			planner: r.planner,
			addr:    rAddr,
		}
	}
	return ret
}

func (r resourceInConfig) Instances(ctx context.Context) map[addrs.UniqueKey]resourceInstance {
	ret := make(map[addrs.UniqueKey]resourceInstance)
	for _, pm := range r.PerModuleInstance(ctx) {
		for k, inst := range pm.Instances(ctx) {
			ret[k] = inst
		}
	}
	return ret
}

func (r resourceInConfig) Schema(ctx context.Context) (*configschema.Block, int64) {
	cfg := r.Config()
	if cfg == nil {
		return nil, 0
	}
	pdr := provider{
		planner: r.planner,
		addr:    cfg.Provider,
	}
	providerSchema, err := pdr.Schema(ctx)
	if err != nil {
		return nil, 0
	}
	rMode := r.Addr().Resource.Mode
	rType := r.Addr().Resource.Type
	var schema providers.Schema
	switch rMode {
	case addrs.ManagedResourceMode:
		schema = providerSchema.ResourceTypes[rType]
	case addrs.DataResourceMode:
		schema = providerSchema.DataSources[rType]
	}
	return schema.Block, schema.Version
}
