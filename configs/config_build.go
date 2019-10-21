package configs

import (
	"sort"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
)

// BuildConfig constructs a Config from a root module by loading all of its
// descendent modules via the given ModuleWalker.
//
// The result is a module tree that has so far only had basic module- and
// file-level invariants validated. If the returned diagnostics contains errors,
// the returned module tree may be incomplete but can still be used carefully
// for static analysis.
func BuildConfig(root *Module, walker ModuleWalker) (*Config, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	cfg := &Config{
		Module: root,
	}
	cfg.Root = cfg // Root module is self-referential.
	cfg.Children, diags = buildChildModules(cfg, walker)
	return cfg, diags
}

func buildChildModules(parent *Config, walker ModuleWalker) (map[string]*Config, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ret := map[string]*Config{}

	calls := parent.Module.ModuleCalls

	// We'll sort the calls by their local names so that they'll appear in a
	// predictable order in any logging that's produced during the walk.
	callNames := make([]string, 0, len(calls))
	for k := range calls {
		callNames = append(callNames, k)
	}
	sort.Strings(callNames)

	for _, callName := range callNames {
		call := calls[callName]
		path := make([]string, len(parent.Path)+1)
		copy(path, parent.Path)
		path[len(path)-1] = call.Name

		req := ModuleRequest{
			Name:              call.Name,
			Path:              path,
			SourceAddr:        call.SourceAddr,
			SourceAddrRange:   call.SourceAddrRange,
			VersionConstraint: call.Version,
			Parent:            parent,
			CallRange:         call.DeclRange,
		}

		mod, ver, modDiags := walker.LoadModule(&req)
		diags = append(diags, modDiags...)
		if mod == nil {
			// nil can be returned if the source address was invalid and so
			// nothing could be loaded whatsoever. LoadModule should've
			// returned at least one error diagnostic in that case.
			continue
		}

		child := &Config{
			Parent:          parent,
			Root:            parent.Root,
			Path:            path,
			Module:          mod,
			CallRange:       call.DeclRange,
			SourceAddr:      call.SourceAddr,
			SourceAddrRange: call.SourceAddrRange,
			Version:         ver,
		}

		child.Children, modDiags = buildChildModules(child, walker)
		diags = append(diags, modDiags...)

		ret[call.Name] = child
	}

	return ret, diags
}

// A ModuleWalker knows how to find and load a child module given details about
// the module to be loaded and a reference to its partially-loaded parent
// Config.
type ModuleWalker interface {
	// LoadModule finds and loads a requested child module.
	//
	// If errors are detected during loading, implementations should return them
	// in the diagnostics object. If the diagnostics object contains any errors
	// then the caller will tolerate the returned module being nil or incomplete.
	// If no errors are returned, it should be non-nil and complete.
	//
	// Full validation need not have been performed but an implementation should
	// ensure that the basic file- and module-validations performed by the
	// LoadConfigDir function (valid syntax, no namespace collisions, etc) have
	// been performed before returning a module.
	LoadModule(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics)
}

// ModuleWalkerFunc is an implementation of ModuleWalker that directly wraps
// a callback function, for more convenient use of that interface.
type ModuleWalkerFunc func(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics)

// LoadModule implements ModuleWalker.
func (f ModuleWalkerFunc) LoadModule(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics) {
	return f(req)
}

// ModuleRequest is used with the ModuleWalker interface to describe a child
// module that must be loaded.
type ModuleRequest struct {
	// Name is the "logical name" of the module call within configuration.
	// This is provided in case the name is used as part of a storage key
	// for the module, but implementations must otherwise treat it as an
	// opaque string. It is guaranteed to have already been validated as an
	// HCL identifier and UTF-8 encoded.
	Name string

	// Path is a list of logical names that traverse from the root module to
	// this module. This can be used, for example, to form a lookup key for
	// each distinct module call in a configuration, allowing for multiple
	// calls with the same name at different points in the tree.
	Path addrs.Module

	// SourceAddr is the source address string provided by the user in
	// configuration.
	SourceAddr string

	// SourceAddrRange is the source range for the SourceAddr value as it
	// was provided in configuration. This can and should be used to generate
	// diagnostics about the source address having invalid syntax, referring
	// to a non-existent object, etc.
	SourceAddrRange hcl.Range

	// VersionConstraint is the version constraint applied to the module in
	// configuration. This data structure includes the source range for
	// the constraint, which can and should be used to generate diagnostics
	// about constraint-related issues, such as constraints that eliminate all
	// available versions of a module whose source is otherwise valid.
	VersionConstraint VersionConstraint

	// Parent is the partially-constructed module tree node that the loaded
	// module will be added to. Callers may refer to any field of this
	// structure except Children, which is still under construction when
	// ModuleRequest objects are created and thus has undefined content.
	// The main reason this is provided is so that full module paths can
	// be constructed for uniqueness.
	Parent *Config

	// CallRange is the source range for the header of the "module" block
	// in configuration that prompted this request. This can be used as the
	// subject of an error diagnostic that relates to the module call itself,
	// rather than to either its source address or its version number.
	CallRange hcl.Range
}

// DisabledModuleWalker is a ModuleWalker that doesn't support
// child modules at all, and so will return an error if asked to load one.
//
// This is provided primarily for testing. There is no good reason to use this
// in the main application.
var DisabledModuleWalker ModuleWalker

func init() {
	DisabledModuleWalker = ModuleWalkerFunc(func(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics) {
		return nil, nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Child modules are not supported",
				Detail:   "Child module calls are not allowed in this context.",
				Subject:  &req.CallRange,
			},
		}
	})
}
