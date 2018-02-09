package configs

import (
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl2/hcl"
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
	// This should not be used to display a path to the end-user, since
	// our UI conventions call for us to return a module address string in that
	// case, and a module address string ought to be built from the dynamic
	// module tree (resulting from evaluating "count" and "for_each" arguments
	// on our calls to produce potentially multiple child instances per call)
	// rather than from our static module tree.
	Path []string

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
