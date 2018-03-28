package terraform

import (
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/moduledeps"
	"github.com/hashicorp/terraform/plugin/discovery"
)

// ConfigTreeDependencies returns the dependencies of the tree of modules
// described by the given configuration and state.
//
// Both configuration and state are required because there can be resources
// implied by instances in the state that no longer exist in config.
func ConfigTreeDependencies(root *configs.Config, state *State) *moduledeps.Module {
	// First we walk the configuration tree to build the overall structure
	// and capture the explicit/implicit/inherited provider dependencies.
	deps := configTreeConfigDependencies(root, nil)

	// Next we walk over the resources in the state to catch any additional
	// dependencies created by existing resources that are no longer in config.
	// Most things we find in state will already be present in 'deps', but
	// we're interested in the rare thing that isn't.
	configTreeMergeStateDependencies(deps, state)

	return deps
}

func configTreeConfigDependencies(root *configs.Config, inheritProviders map[string]*configs.Provider) *moduledeps.Module {
	if root == nil {
		// If no config is provided, we'll make a synthetic root.
		// This isn't necessarily correct if we're called with a nil that
		// *isn't* at the root, but in practice that can never happen.
		return &moduledeps.Module{
			Name: "root",
		}
	}

	name := "root"
	if len(root.Path) != 0 {
		name = root.Path[len(root.Path)-1]
	}

	ret := &moduledeps.Module{
		Name: name,
	}

	module := root.Module

	// Provider dependencies
	{
		providers := make(moduledeps.Providers)

		// The main way to declare a provider dependency is explicitly inside
		// the "terraform" block, which allows declaring a requirement without
		// also creating a configuration.
		for fullName, constraints := range module.ProviderRequirements {
			inst := moduledeps.ProviderInstance(fullName)

			// The handling here is a bit fiddly because the moduledeps package
			// was designed around the legacy (pre-0.12) configuration model
			// and hasn't yet been revised to handle the new model. As a result,
			// we need to do some translation here.
			// FIXME: Eventually we should adjust the underlying model so we
			// can also retain the source location of each constraint, for
			// more informative output from the "terraform providers" command.
			var rawConstraints version.Constraints
			for _, constraint := range constraints {
				rawConstraints = append(rawConstraints, constraint.Required...)
			}
			discoConstraints := discovery.NewConstraints(rawConstraints)

			providers[inst] = moduledeps.ProviderDependency{
				Constraints: discoConstraints,
				Reason:      moduledeps.ProviderDependencyExplicit,
			}
		}

		// Provider configurations can also include version constraints,
		// allowing for more terse declaration in situations where both a
		// configuration and a constraint are defined in the same module.
		for fullName, pCfg := range module.ProviderConfigs {
			inst := moduledeps.ProviderInstance(fullName)
			discoConstraints := discovery.AllVersions
			if pCfg.Version.Required != nil {
				discoConstraints = discovery.NewConstraints(pCfg.Version.Required)
			}
			if existing, exists := providers[inst]; exists {
				existing.Constraints = existing.Constraints.Append(discoConstraints)
			} else {
				providers[inst] = moduledeps.ProviderDependency{
					Constraints: discoConstraints,
					Reason:      moduledeps.ProviderDependencyExplicit,
				}
			}
		}

		// Each resource in the configuration creates an *implicit* provider
		// dependency, though we'll only record it if there isn't already
		// an explicit dependency on the same provider.
		for _, rc := range module.ManagedResources {
			fullName := rc.ProviderConfigKey()
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
			fullName := rc.ProviderConfigKey()
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

	childInherit := make(map[string]*configs.Provider)
	for k, v := range inheritProviders {
		childInherit[k] = v
	}
	for k, v := range module.ProviderConfigs {
		childInherit[k] = v
	}
	for _, c := range root.Children {
		ret.Children = append(ret.Children, configTreeConfigDependencies(c, childInherit))
	}

	return ret
}

func configTreeMergeStateDependencies(root *moduledeps.Module, state *State) {
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

// ModuleTreeDependencies returns the dependencies of the tree of modules
// described by the given configuration tree and state.
//
// Both configuration and state are required because there can be resources
// implied by instances in the state that no longer exist in config.
//
// This function will panic if any invalid version constraint strings are
// present in the configuration. This is guaranteed not to happen for any
// configuration that has passed a call to Config.Validate().
func ModuleTreeDependencies(root *module.Tree, state *State) *moduledeps.Module {
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

func moduleTreeConfigDependencies(root *module.Tree, inheritProviders map[string]*config.ProviderConfig) *moduledeps.Module {
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

	cfg := root.Config()
	providerConfigs := cfg.ProviderConfigsByFullName()

	// Provider dependencies
	{
		providers := make(moduledeps.Providers, len(providerConfigs))

		// Any providerConfigs elements are *explicit* provider dependencies,
		// which is the only situation where the user might provide an actual
		// version constraint. We'll take care of these first.
		for fullName, pCfg := range providerConfigs {
			inst := moduledeps.ProviderInstance(fullName)
			versionSet := discovery.AllVersions
			if pCfg.Version != "" {
				versionSet = discovery.ConstraintStr(pCfg.Version).MustParse()
			}
			providers[inst] = moduledeps.ProviderDependency{
				Constraints: versionSet,
				Reason:      moduledeps.ProviderDependencyExplicit,
			}
		}

		// Each resource in the configuration creates an *implicit* provider
		// dependency, though we'll only record it if there isn't already
		// an explicit dependency on the same provider.
		for _, rc := range cfg.Resources {
			fullName := rc.ProviderFullName()
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
	// This is really just the same logic as configTreeMergeStateDependencies
	// but we retain this old name just to keep the symmetry until we've
	// removed all of these "moduleTree..." versions that use the legacy
	// configuration structs.
	configTreeMergeStateDependencies(root, state)
}
