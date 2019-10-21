package earlyconfig

import (
	"fmt"
	"sort"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
)

// BuildConfig constructs a Config from a root module by loading all of its
// descendent modules via the given ModuleWalker.
func BuildConfig(root *tfconfig.Module, walker ModuleWalker) (*Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	cfg := &Config{
		Module: root,
	}
	cfg.Root = cfg // Root module is self-referential.
	cfg.Children, diags = buildChildModules(cfg, walker)
	return cfg, diags
}

func buildChildModules(parent *Config, walker ModuleWalker) (map[string]*Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
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

		var vc version.Constraints
		if strings.TrimSpace(call.Version) != "" {
			var err error
			vc, err = version.NewConstraint(call.Version)
			if err != nil {
				diags = diags.Append(wrapDiagnostic(tfconfig.Diagnostic{
					Severity: tfconfig.DiagError,
					Summary:  "Invalid version constraint",
					Detail:   fmt.Sprintf("Module %q (declared at %s line %d) has invalid version constraint %q: %s.", callName, call.Pos.Filename, call.Pos.Line, call.Version, err),
				}))
				continue
			}
		}

		req := ModuleRequest{
			Name:               call.Name,
			Path:               path,
			SourceAddr:         call.Source,
			VersionConstraints: vc,
			Parent:             parent,
			CallPos:            call.Pos,
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
			Parent:     parent,
			Root:       parent.Root,
			Path:       path,
			Module:     mod,
			CallPos:    call.Pos,
			SourceAddr: call.Source,
			Version:    ver,
		}

		child.Children, modDiags = buildChildModules(child, walker)
		diags = diags.Append(modDiags)

		ret[call.Name] = child
	}

	return ret, diags
}

// ModuleRequest is used as part of the ModuleWalker interface used with
// function BuildConfig.
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

	// VersionConstraint is the version constraint applied to the module in
	// configuration.
	VersionConstraints version.Constraints

	// Parent is the partially-constructed module tree node that the loaded
	// module will be added to. Callers may refer to any field of this
	// structure except Children, which is still under construction when
	// ModuleRequest objects are created and thus has undefined content.
	// The main reason this is provided is so that full module paths can
	// be constructed for uniqueness.
	Parent *Config

	// CallRange is the source position for the header of the "module" block
	// in configuration that prompted this request.
	CallPos tfconfig.SourcePos
}

// ModuleWalker is an interface used with BuildConfig.
type ModuleWalker interface {
	LoadModule(req *ModuleRequest) (*tfconfig.Module, *version.Version, tfdiags.Diagnostics)
}

// ModuleWalkerFunc is an implementation of ModuleWalker that directly wraps
// a callback function, for more convenient use of that interface.
type ModuleWalkerFunc func(req *ModuleRequest) (*tfconfig.Module, *version.Version, tfdiags.Diagnostics)

func (f ModuleWalkerFunc) LoadModule(req *ModuleRequest) (*tfconfig.Module, *version.Version, tfdiags.Diagnostics) {
	return f(req)
}
