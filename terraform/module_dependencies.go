package terraform

import (
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/moduledeps"
	"github.com/hashicorp/terraform/plugin/discovery"
)

// ModuleTreeDependencies returns the dependencies of the tree of modules
// described by the given configuration tree and state.
//
// Both configuration and state are required because there can be resources
// implied by instances in the state that no longer exist in config.
func ModuleTreeDependencies(root *configs.Config, state *State) *moduledeps.Module {
	// First we walk the configuration tree to build the overall structure
	// and capture the explicit/implicit/inherited provider dependencies.
	deps := moduleTreeConfigDependencies(root, nil)

	// Next we walk over the resources in the state to catch any additional
	// dependencies created by existing resources that are no longer in config.
	// Most things we find in state will already be present in 'deps', but
	// we're interested in the rare thing that isn't.
	moduleTreeMergeStateDependencies(deps, state)

	return deps
}

func moduleTreeConfigDependencies(root *configs.Config, inheritProviders map[string]*config.ProviderConfig) *moduledeps.Module {
	if root == nil {
		// If no config is provided, we'll make a synthetic root.
		// This isn't necessarily correct if we're called with a nil that
		// *isn't* at the root, but in practice that can never happen.
		return &moduledeps.Module{
			Name: "root",
		}
	}

	ret := &moduledeps.Module{
		Name: root.Name(),
	}

	module := root.Module

	// Provider dependencies
	// FIXME: The structs used here were designed before we were able to retain
	// source location information for provider dependencies, and so we lose
	// our source location information here. At some point we should try to
	// rework this so that we can retain the location of each constraint and
	// then produce a contextual diagnostic if any of the requirements can't
	// be met.
	{
		providers := make(moduledeps.Providers, len(providerConfigs))

		// A module can declare an explicit provider dependency without
		// actually configuring the requested provider, under the assumption
		// that a configuration will be passed by the caller.
		for name, required := range module.ProviderRequirements {
			inst := moduledeps.ProviderInstance(name)
			var rawConstraints version.Constraints
			for _, constraint := range required {
				rawConstraints = append(rawConstraints, constraint.Required...)
			}
			providers[inst] = moduledeps.ProviderDependency{
				Constraints: discovery.NewConstraints(rawConstraints),
				Reason:      moduledeps.ProviderDependencyExplicit,
			}
		}

		// Any providerConfigs elements are *explicit* provider dependencies,
		// which is the only situation where the user might provide an actual
		// version constraint. We'll take care of these first.
		for fullName, pCfg := range module.ProviderConfigs {
			inst := moduledeps.ProviderInstance(fullName)
			versionSet := discovery.AllVersions
			if pCfg.Version != "" {
				versionSet = discovery.ConstraintStr(pCfg.Version).MustParse()
			}
			if existing, exists := providers[inst]; exists {
				// Explicit dependency already present, so we'll append our
				// new constraints into it.
				existing.Constraints = append(existing.Constraints, versionSet...)
				continue
			}

			providers[inst] = moduledeps.ProviderDependency{
				Constraints: versionSet,
				Reason:      moduledeps.ProviderDependencyExplicit,
			}
		}

		// Each resource in the configuration creates an *implicit* provider
		// dependency, though we'll only record it if there isn't already
		// an explicit dependency on the same provider.
		for _, rc := range module.ManagedResources {
			fullName := rc.ProviderConfigFullName()
			inst := moduledeps.ProviderInstance(fullName)
			if _, exists := providers[inst]; exists {
				// Explicit dependency already present
				continue
			}

			reason := moduledeps.ProviderDependencyImplicit
			if _, inherited := inheritProviders[fullName]; inherited {
				reason = moduledeps.ProviderDependencyInherited
			}

			providers[inst] = moduledeps.ProviderDependency{
				Constraints: discovery.AllVersions,
				Reason:      reason,
			}
		}
		for _, rc := range module.DataResources {
			fullName := rc.ProviderConfigFullName()
			inst := moduledeps.ProviderInstance(fullName)
			if _, exists := providers[inst]; exists {
				// Explicit dependency already present
				continue
			}

			reason := moduledeps.ProviderDependencyImplicit
			if _, inherited := inheritProviders[fullName]; inherited {
				reason = moduledeps.ProviderDependencyInherited
			}

			providers[inst] = moduledeps.ProviderDependency{
				Constraints: discovery.AllVersions,
				Reason:      reason,
			}
		}

		ret.Providers = providers
	}

	childInherit := make(map[string]*config.ProviderConfig)
	for k, v := range inheritProviders {
		childInherit[k] = v
	}
	for k, v := range providerConfigs {
		childInherit[k] = v
	}
	for _, c := range root.Children() {
		ret.Children = append(ret.Children, moduleTreeConfigDependencies(c, childInherit))
	}

	return ret
}

func moduleTreeMergeStateDependencies(root *moduledeps.Module, state *State) {
	if state == nil {
		return
	}

	findModule := func(path []string) *moduledeps.Module {
		module := root
		for _, name := range path[1:] { // skip initial "root"
			var next *moduledeps.Module
			for _, cm := range module.Children {
				if cm.Name == name {
					next = cm
					break
				}
			}

			if next == nil {
				// If we didn't find a next node, we'll need to make one
				next = &moduledeps.Module{
					Name: name,
				}
				module.Children = append(module.Children, next)
			}

			module = next
		}
		return module
	}

	for _, ms := range state.Modules {
		module := findModule(ms.Path)

		for _, is := range ms.Resources {
			fullName := config.ResourceProviderFullName(is.Type, is.Provider)
			inst := moduledeps.ProviderInstance(fullName)
			if _, exists := module.Providers[inst]; !exists {
				if module.Providers == nil {
					module.Providers = make(moduledeps.Providers)
				}
				module.Providers[inst] = moduledeps.ProviderDependency{
					Constraints: discovery.AllVersions,
					Reason:      moduledeps.ProviderDependencyFromState,
				}
			}
		}
	}

}
