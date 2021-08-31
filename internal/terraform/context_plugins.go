package terraform

import (
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
)

// contextPlugins represents a library of available plugins (providers and
// provisioners) which we assume will all be used with the same
// terraform.Context, and thus it'll be safe to cache certain information
// about the providers for performance reasons.
type contextPlugins struct {
	providerFactories    map[addrs.Provider]providers.Factory
	provisionerFactories map[string]provisioners.Factory

	// We memoize the schemas we've previously loaded in here, to avoid
	// repeatedly paying the cost of activating the same plugins to access
	// their schemas in various different spots. We use schemas for many
	// purposes in Terraform, so there isn't a single choke point where
	// it makes sense to preload all of them.
	providerSchemas    map[addrs.Provider]*ProviderSchema
	provisionerSchemas map[string]*configschema.Block
	schemasLock        *sync.Mutex
}

func newContextPlugins(providerFactories map[addrs.Provider]providers.Factory, provisionerFactories map[string]provisioners.Factory) *contextPlugins {
	ret := &contextPlugins{
		providerFactories:    providerFactories,
		provisionerFactories: provisionerFactories,
	}
	ret.init()
	return ret
}

func (cp *contextPlugins) init() {
	cp.providerSchemas = make(map[addrs.Provider]*ProviderSchema, len(cp.providerFactories))
	cp.provisionerSchemas = make(map[string]*configschema.Block, len(cp.provisionerFactories))
}

func (cp *contextPlugins) NewProviderInstance(addr addrs.Provider) (providers.Interface, error) {
	f, ok := cp.providerFactories[addr]
	if !ok {
		return nil, fmt.Errorf("unavailable provider %q", addr.String())
	}

	return f()

}

func (cp *contextPlugins) NewProvisionerInstance(typ string) (provisioners.Interface, error) {
	f, ok := cp.provisionerFactories[typ]
	if !ok {
		return nil, fmt.Errorf("unavailable provisioner %q", typ)
	}

	return f()
}
