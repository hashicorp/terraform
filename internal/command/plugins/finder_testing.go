package plugins

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
)

// FinderTestingOverrides is for use with NewFinderForTests to
// bypass all of the usual discovery behaviors inside unit and integration
// tests.
type FinderTestingOverrides struct {
	Providers    map[addrs.Provider]providers.Factory
	Provisioners map[string]provisioners.Factory
}

// NewFinderForTests creates and returns a provider that skips all of the
// usual discovery steps and instead just "finds" exactly the mock plugin
// components given in the argument.
//
// Because "built-in" providers are treated as special by various other
// Terraform components, NewFinderForTests will identify any providers in the
// overrides map which belong to the built-in namespace and retain them as
// built-in providers instead. They will therefore show up in the result of
// Finder.BuiltinProviderTypes, and can be overridden by a subsequent
// call to Finder.WithAdditionalBuiltinProviders.
//
// A test-oriented Finder can still handle all of the other "With..." methods,
// but they will have no observable effect. The goal is to allow the usual
// codepaths to try to customize the Finder in the ways they usually would,
// but for those calls to be effectively no-ops so that the testing overrides
// can still shine through regardless.
//
// After calling NewFinderForTests, all objects reachable through the given
// overrides object belong to the finder and must not be read or written by
// the caller.
func NewFinderForTests(overrides FinderTestingOverrides) Finder {
	// If any of the provider overrides are "builtin" providers then we need to
	// include them in the map of builtins instead, because some other
	// components (like the plugin installer) treat those as special.
	var builtins map[string]providers.Factory
	var others map[addrs.Provider]providers.Factory
	for addr, factory := range overrides.Providers {
		switch {
		case addr.IsBuiltIn():
			if builtins == nil {
				builtins = make(map[string]providers.Factory)
			}
			builtins[addr.Type] = factory
		default:
			if others == nil {
				others = make(map[addrs.Provider]providers.Factory)
			}
			others[addr] = factory
		}
	}

	return Finder{
		testingOverrides: &overrides,
		providerBuiltins: builtins,
	}
}
