package pluggable_state

import (
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	grpc_statemgr "github.com/hashicorp/terraform/internal/states/grpc"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func NewPluggable(p providers.Interface, typeName string) backend.Backend {
	return &Pluggable{
		provider: p,
		typeName: typeName,
	}
}

var _ backend.Backend = &Pluggable{}

type Pluggable struct {
	provider providers.Interface
	typeName string
}

func (p *Pluggable) ConfigSchema() *configschema.Block {
	schemaResp := p.provider.GetProviderSchema()
	if schemaResp.StateStores == nil {
		// No state stores
		return nil
	}
	val, ok := schemaResp.StateStores[p.typeName]
	if !ok {
		// Cannot find state store with that type
		return nil
	}

	// State store type exists
	return val.Body
}

// PrepareConfig validates the provided config value and returns any necessary diagnostics.
// The method also returns the input config value, unchanged.
//
// Context: Why do we return the config? The answer is this method's signature is affected
// by some legacy stuff:
//
//	> In the past, config validation was performed by methods starting with "Prepare". These
//	  methods were allowed to return a mutated version of the configuration. This can be seen
//	  in plugin protocol v5 where RPCs like `PrepareProviderConfig` exist and return a 'prepared'
//	  config.
//	> Since then, plugin protocol v6 included changes to rename those methods to start with
//	  "Validate" and no longer return 'prepared' config.
//	> How backend.Backend is implemented reflects that past idea of how config validation should
//	  be approached. Note the method here also starts with "Prepare".
//	> In the context of pluggable state storage, we disallow providers from returning prepared
//	  config for a state store during state store-related "Validate" RPCs. Therefore in the
//	  code below we return the original config in order to fulfil the method signature.
//
// TODO (SarahFrench) - update the backend.Backend interface to have a `ValidateConfig` method,
// instead of `PrepareConfig`, which only returns diagnostics.
func (p *Pluggable) PrepareConfig(config cty.Value) (cty.Value, tfdiags.Diagnostics) {
	req := providers.ValidateStateStoreConfigRequest{
		TypeName: p.typeName,
		Config:   config,
	}
	resp := p.provider.ValidateStateStoreConfig(req)
	return config, resp.Diagnostics
}

func (p *Pluggable) Configure(config cty.Value) tfdiags.Diagnostics {
	req := providers.ConfigureStateStoreRequest{
		TypeName: p.typeName,
		Config:   config,
	}
	resp := p.provider.ConfigureStateStore(req)
	return resp.Diagnostics
}

func (p *Pluggable) Workspaces() ([]string, error) {
	return nil, nil
}

func (p *Pluggable) DeleteWorkspace(workspace string, force bool) error {
	return nil
}

func (p *Pluggable) StateMgr(workspace string) (statemgr.Full, error) {
	// repackages the provider's methods inside a state manager,
	// to be passed to the calling code that expects a statemgr.Full
	return grpc_statemgr.NewGrpcStateManager(p.provider, p.typeName, workspace), nil
}
