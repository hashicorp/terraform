package planner

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
	"github.com/zclconf/go-cty/cty"
)

type providerConfig struct {
	planner *planner
	addr    addrs.AbsProviderConfig
}

func (p providerConfig) Addr() addrs.AbsProviderConfig {
	return p.addr
}

func (p providerConfig) Provider() provider {
	return provider{
		planner: p.planner,
		addr:    p.addr.Provider,
	}
}

func (p providerConfig) Instance() (providers.Interface, error) {
	return p.planner.ConfiguredProviderInstance(p)
}

func (p providerConfig) configureProviderInstance(inst providers.Interface) tfdiags.Diagnostics {
	log.Printf("[TRACE] ConfigureProvider for %s", p.addr)
	resp := inst.ConfigureProvider(providers.ConfigureProviderRequest{
		TerraformVersion: version.SemVer.String(),

		// TODO: Evaluate and send the real configuration
		Config: cty.EmptyObjectVal,
	})
	return resp.Diagnostics
}
