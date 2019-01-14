package earlyconfig

import (
	"fmt"
	"sort"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/moduledeps"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/tfdiags"
)

// A Config is a node in the tree of modules within a configuration.
//
// The module tree is constructed by following ModuleCall instances recursively
// through the root module transitively into descendent modules.
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
	// have been resolved. In most cases, a addrs.ModuleInstance describing
	// a node in the dynamic module tree is better, since it will then include
	// any keys resulting from evaluating "count" and "for_each" arguments.
	Path addrs.Module

	// ChildModules points to the Config for each of the direct child modules
	// called from this module. The keys in this map match the keys in
	// Module.ModuleCalls.
	Children map[string]*Config

	// Module points to the object describing the configuration for the
	// various elements (variables, resources, etc) defined by this module.
	Module *tfconfig.Module

	// CallPos is the source position for the header of the module block that
	// requested this module.
	//
	// This field is meaningless for the root module, where its contents are undefined.
	CallPos tfconfig.SourcePos

	// SourceAddr is the source address that the referenced module was requested
	// from, as specified in configuration.
	//
	// This field is meaningless for the root module, where its contents are undefined.
	SourceAddr string

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

// ProviderDependencies returns the provider dependencies for the recieving
// config, including all of its descendent modules.
func (c *Config) ProviderDependencies() (*moduledeps.Module, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var name string
	if len(c.Path) > 0 {
		name = c.Path[len(c.Path)-1]
	}

	ret := &moduledeps.Module{
		Name: name,
	}

	providers := make(moduledeps.Providers)
	for name, reqs := range c.Module.RequiredProviders {
		inst := moduledeps.ProviderInstance(name)
		var constraints version.Constraints
		for _, reqStr := range reqs {
			if reqStr != "" {
				constraint, err := version.NewConstraint(reqStr)
				if err != nil {
					diags = diags.Append(wrapDiagnostic(tfconfig.Diagnostic{
						Severity: tfconfig.DiagError,
						Summary:  "Invalid provider version constraint",
						Detail:   fmt.Sprintf("Invalid version constraint %q for provider %s.", reqStr, name),
					}))
					continue
				}
				constraints = append(constraints, constraint...)
			}
		}
		providers[inst] = moduledeps.ProviderDependency{
			Constraints: discovery.NewConstraints(constraints),
			Reason:      moduledeps.ProviderDependencyExplicit,
		}
	}
	ret.Providers = providers

	childNames := make([]string, 0, len(c.Children))
	for name := range c.Children {
		childNames = append(childNames, name)
	}
	sort.Strings(childNames)

	for _, name := range childNames {
		child, childDiags := c.Children[name].ProviderDependencies()
		ret.Children = append(ret.Children, child)
		diags = diags.Append(childDiags)
	}

	return ret, diags
}
