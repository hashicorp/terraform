package globalref

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/providers"
)

// Analyzer is the main component of this package, serving as a container for
// various state that the analysis algorithms depend on either for their core
// functionality or for producing results more quickly.
//
// Global reference analysis is currently intended only for "best effort"
// use-cases related to giving hints to the user or tailoring UI output.
// Avoid using it for anything that would cause changes to the analyzer being
// considered a breaking change under the v1 compatibility promises, because
// we expect to continue to refine and evolve these rules over time in ways
// that may cause us to detect either more or fewer references than today.
// Typically we will conservatively return more references than would be
// necessary dynamically, but that isn't guaranteed for all situations.
//
// In particular, we currently typically don't distinguish between multiple
// instances of the same module, and so we overgeneralize references from
// one instance of a module as references from the same location in all
// instances of that module. We may make this more precise in future, which
// would then remove various detected references from the analysis results.
//
// Each Analyzer works with a particular configs.Config object which it assumes
// represents the root module of a configuration. Config objects are typically
// immutable by convention anyway, but it's particularly important not to
// modify a configuration while it's attached to a live Analyzer, because
// the Analyzer contains caches derived from data in the configuration tree.
type Analyzer struct {
	cfg             *configs.Config
	providerSchemas map[addrs.Provider]*providers.Schemas
}

// NewAnalyzer constructs a new analyzer bound to the given configuration and
// provider schemas.
//
// The given object must represent a root module, or this function will panic.
//
// The given provider schemas must cover at least all of the providers used
// in the given configuration. If not then analysis results will be silently
// incomplete for any decision that requires checking schema.
func NewAnalyzer(cfg *configs.Config, providerSchemas map[addrs.Provider]*providers.Schemas) *Analyzer {
	if !cfg.Path.IsRoot() {
		panic(fmt.Sprintf("constructing an Analyzer with non-root module %s", cfg.Path))
	}

	ret := &Analyzer{
		cfg:             cfg,
		providerSchemas: providerSchemas,
	}
	return ret
}

// ModuleConfig retrieves a module configuration from the configuration the
// analyzer belongs to, or nil if there is no module with the given address.
func (a *Analyzer) ModuleConfig(addr addrs.ModuleInstance) *configs.Module {
	modCfg := a.cfg.DescendentForInstance(addr)
	if modCfg == nil {
		return nil
	}
	return modCfg.Module
}
