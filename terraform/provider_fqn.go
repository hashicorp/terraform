package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/states"
)

// gatherProviderFQNs creates a map of AbsProviderConfig string addresses to
// addrs.Provider from providers in module.ProviderRequirements.
func gatherProviderFQNs(c *configs.Config, s *states.State) map[string]addrs.Provider {
	ret := make(map[string]addrs.Provider)
	// FIXME: we need to be able to get this information out of state,
	// but state doesn't have that information yet.

	// if the provider is in m.RequireProviders, it's possible that
	// the localName does not match the type
	gatherProviderFQNsFromConfig(c, addrs.RootModuleInstance, ret)

	return ret
}

func gatherProviderFQNsFromConfig(c *configs.Config, path addrs.ModuleInstance, ret map[string]addrs.Provider) {
	for localName, provider := range c.Module.ProviderRequirements {
		pc := addrs.LocalProviderConfig{Type: localName}
		pAddr := pc.Absolute(path)
		ret[pAddr.String()] = provider.Type
	}

	for _, child := range c.Children {
		gatherProviderFQNsFromConfig(child, child.Path.UnkeyedInstanceShim(), ret)
	}
}
