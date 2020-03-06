package configs

import (
	"fmt"
	"sort"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
)

// A Config is a node in the tree of modules within a configuration.
//
// The module tree is constructed by following ModuleCall instances recursively
// through the root module transitively into descendent modules.
//
// A module tree described in *this* package represents the static tree
// represented by configuration. During evaluation a static ModuleNode may
// expand into zero or more module instances depending on the use of count and
// for_each configuration attributes within each call.
type Config struct {
	// RootModule points to the Config for the root module within the same
	// module tree as this module. If this module _is_ the root module then
	// this is self-referential.
	Root *Config

	// ParentModule points to the Config for the module that directly calls
	// this module. If this is the root module then this field is nil.
	Parent *Config

	// Path is a sequence of module logical names that traverse from the root
	// module to this config. Path is empty for the root module.
	//
	// This should only be used to display paths to the end-user in rare cases
	// where we are talking about the static module tree, before module calls
	// have been resolved. In most cases, an addrs.ModuleInstance describing
	// a node in the dynamic module tree is better, since it will then include
	// any keys resulting from evaluating "count" and "for_each" arguments.
	Path addrs.Module

	// ChildModules points to the Config for each of the direct child modules
	// called from this module. The keys in this map match the keys in
	// Module.ModuleCalls.
	Children map[string]*Config

	// Module points to the object describing the configuration for the
	// various elements (variables, resources, etc) defined by this module.
	Module *Module

	// CallRange is the source range for the header of the module block that
	// requested this module.
	//
	// This field is meaningless for the root module, where its contents are undefined.
	CallRange hcl.Range

	// SourceAddr is the source address that the referenced module was requested
	// from, as specified in configuration.
	//
	// This field is meaningless for the root module, where its contents are undefined.
	SourceAddr string

	// SourceAddrRange is the location in the configuration source where the
	// SourceAddr value was set, for use in diagnostic messages.
	//
	// This field is meaningless for the root module, where its contents are undefined.
	SourceAddrRange hcl.Range

	// Version is the specific version that was selected for this module,
	// based on version constraints given in configuration.
	//
	// This field is nil if the module was loaded from a non-registry source,
	// since versions are not supported for other sources.
	//
	// This field is meaningless for the root module, where it will always
	// be nil.
	Version *version.Version
}

// NewEmptyConfig constructs a single-node configuration tree with an empty
// root module. This is generally a pretty useless thing to do, so most callers
// should instead use BuildConfig.
func NewEmptyConfig() *Config {
	ret := &Config{}
	ret.Root = ret
	ret.Children = make(map[string]*Config)
	ret.Module = &Module{}
	return ret
}

// Depth returns the number of "hops" the receiver is from the root of its
// module tree, with the root module having a depth of zero.
func (c *Config) Depth() int {
	ret := 0
	this := c
	for this.Parent != nil {
		ret++
		this = this.Parent
	}
	return ret
}

// DeepEach calls the given function once for each module in the tree, starting
// with the receiver.
//
// A parent is always called before its children and children of a particular
// node are visited in lexicographic order by their names.
func (c *Config) DeepEach(cb func(c *Config)) {
	cb(c)

	names := make([]string, 0, len(c.Children))
	for name := range c.Children {
		names = append(names, name)
	}

	for _, name := range names {
		c.Children[name].DeepEach(cb)
	}
}

// AllModules returns a slice of all the receiver and all of its descendent
// nodes in the module tree, in the same order they would be visited by
// DeepEach.
func (c *Config) AllModules() []*Config {
	var ret []*Config
	c.DeepEach(func(c *Config) {
		ret = append(ret, c)
	})
	return ret
}

// Descendent returns the descendent config that has the given path beneath
// the receiver, or nil if there is no such module.
//
// The path traverses the static module tree, prior to any expansion to handle
// count and for_each arguments.
//
// An empty path will just return the receiver, and is therefore pointless.
func (c *Config) Descendent(path addrs.Module) *Config {
	current := c
	for _, name := range path {
		current = current.Children[name]
		if current == nil {
			return nil
		}
	}
	return current
}

// DescendentForInstance is like Descendent except that it accepts a path
// to a particular module instance in the dynamic module graph, returning
// the node from the static module graph that corresponds to it.
//
// All instances created by a particular module call share the same
// configuration, so the keys within the given path are disregarded.
func (c *Config) DescendentForInstance(path addrs.ModuleInstance) *Config {
	current := c
	for _, step := range path {
		current = current.Children[step.Name]
		if current == nil {
			return nil
		}
	}
	return current
}

// ProviderTypes returns the FQNs of each distinct provider type referenced
// in the receiving configuration.
//
// This is a helper for easily determining which provider types are required
// to fully interpret the configuration, though it does not include version
// information and so callers are expected to have already dealt with
// provider version selection in an earlier step and have identified suitable
// versions for each provider.
func (c *Config) ProviderTypes() []addrs.Provider {
	m := make(map[addrs.Provider]struct{})
	c.gatherProviderTypes(m)

	ret := make([]addrs.Provider, 0, len(m))
	for k := range m {
		ret = append(ret, k)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].String() < ret[j].String()
	})
	return ret
}

func (c *Config) gatherProviderTypes(m map[addrs.Provider]struct{}) {
	if c == nil {
		return
	}

	// FIXME: These are currently all assuming legacy provider addresses.
	// As part of phasing those out we'll need to change this to look up
	// the true provider addresses via the local-to-FQN mapping table
	// stored inside c.Module.
	for _, pc := range c.Module.ProviderConfigs {
		fqn := c.Module.ProviderForLocalConfig(addrs.LocalProviderConfig{LocalName: pc.Name})
		m[fqn] = struct{}{}
	}
	for _, rc := range c.Module.ManagedResources {
		providerAddr := rc.ProviderConfigAddr()
		fqn := c.Module.ProviderForLocalConfig(providerAddr)
		m[fqn] = struct{}{}
	}
	for _, rc := range c.Module.DataResources {
		providerAddr := rc.ProviderConfigAddr()
		fqn := c.Module.ProviderForLocalConfig(providerAddr)
		m[fqn] = struct{}{}
	}

	// Must also visit our child modules, recursively.
	for _, cc := range c.Children {
		cc.gatherProviderTypes(m)
	}
}

// ResolveAbsProviderAddr returns the AbsProviderConfig represented by the given
// ProviderConfig address, which must not be nil or this method will panic.
//
// If the given address is already an AbsProviderConfig then this method returns
// it verbatim, and will always succeed. If it's a LocalProviderConfig then
// it will consult the local-to-FQN mapping table for the given module
// to find the absolute address corresponding to the given local one.
//
// The module address to resolve local addresses in must be given in the second
// argument, and must refer to a module that exists under the receiver or
// else this method will panic.
func (c *Config) ResolveAbsProviderAddr(addr addrs.ProviderConfig, inModule addrs.ModuleInstance) addrs.AbsProviderConfig {
	switch addr := addr.(type) {

	case addrs.AbsProviderConfig:
		return addr

	case addrs.LocalProviderConfig:
		// Find the descendent Config that contains the module that this
		// local config belongs to.
		mc := c.DescendentForInstance(inModule)
		if mc == nil {
			panic(fmt.Sprintf("ResolveAbsProviderAddr with non-existent module %s", inModule.String()))
		}

		var provider addrs.Provider
		if providerReq, exists := c.Module.ProviderRequirements[addr.LocalName]; exists {
			provider = providerReq.Type
		} else {
			// FIXME: For now we're returning a _legacy_ address as fallback here,
			// but once we remove legacy addresses this should actually be a
			// _default_ provider address.
			provider = addrs.NewLegacyProvider(addr.LocalName)
		}

		return addrs.AbsProviderConfig{
			Module:   inModule,
			Provider: provider,
			Alias:    addr.Alias,
		}

	default:
		panic(fmt.Sprintf("cannot ResolveAbsProviderAddr(%v, ...)", addr))
	}

}

// ProviderForConfigAddr returns the FQN for a given addrs.ProviderConfig, first
// by checking for the provider in module.ProviderRequirements and falling
// back to addrs.NewLegacyProvider if it is not found.
func (c *Config) ProviderForConfigAddr(addr addrs.LocalProviderConfig) addrs.Provider {
	if provider, exists := c.Module.ProviderRequirements[addr.LocalName]; exists {
		return provider.Type
	}
	return c.ResolveAbsProviderAddr(addr, addrs.RootModuleInstance).Provider
}
