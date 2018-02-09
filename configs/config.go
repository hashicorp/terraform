package configs

import (
	"fmt"

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

// Path returns the path of logical names that lead to this Config from its
// root.
//
// This function should not be used to display a path to the end-user, since
// our UI conventions call for us to return a module address string in that
// case, and a module address string ought to be built from the dynamic
// module tree (resulting from evaluating "count" and "for_each" arguments
// on our calls to produce potentially multiple child instances per call)
// rather than from our static module tree.
//
// This function will panic if called on a config that is not part of a
// wholesome config tree, e.g. because it has incorrectly-built Children
// maps, missing node pointers, etc. However, it should work as expected
// for any tree constructed by BuildConfig and not subsequently modified.
func (c *Config) Path() []string {
	// The implementation here is not especially efficient, but we don't
	// care too much because module trees are shallow and narrow in all
	// reasonable configurations.

	// We'll build our path in reverse here, since we're starting at the
	// leafiest node, and then we'll flip it before we return.
	path := make([]string, 0, c.Depth())

	this := c
	for this.Parent != nil {
		parent := this.Parent
		var name string
		for candidate, ref := range parent.Children {
			if ref == this {
				name = candidate
			}
		}
		if name == "" {
			panic(fmt.Errorf(
				"Config %p does not appear in the child table for its parent %p: %#v",
				this, parent, parent.Children,
			))
		}
		path = append(path, name)
		this = parent
	}

	// reverse the items
	for i := 0; i < len(path)/2; i++ {
		j := len(path) - i - 1
		path[i], path[j] = path[j], path[i]
	}
	return path
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
