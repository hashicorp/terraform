package earlyconfig

import (
	"fmt"
	"sort"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/moduledeps"
	"github.com/hashicorp/terraform/internal/plugin/discovery"
	"github.com/hashicorp/terraform/internal/tfdiags"
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
	SourceAddr addrs.ModuleSource

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

// ProviderRequirements searches the full tree of modules under the receiver
// for both explicit and implicit dependencies on providers.
//
// The result is a full manifest of all of the providers that must be available
// in order to work with the receiving configuration.
//
// If the returned diagnostics includes errors then the resulting Requirements
// may be incomplete.
func (c *Config) ProviderRequirements() (getproviders.Requirements, tfdiags.Diagnostics) {
	reqs := make(getproviders.Requirements)
	diags := c.addProviderRequirements(reqs)
	return reqs, diags
}

// addProviderRequirements is the main part of the ProviderRequirements
// implementation, gradually mutating a shared requirements object to
// eventually return.
func (c *Config) addProviderRequirements(reqs getproviders.Requirements) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// First we'll deal with the requirements directly in _our_ module...
	for localName, providerReqs := range c.Module.RequiredProviders {
		var fqn addrs.Provider
		if source := providerReqs.Source; source != "" {
			addr, moreDiags := addrs.ParseProviderSourceString(source)
			if moreDiags.HasErrors() {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid provider source address",
					fmt.Sprintf("Invalid source %q for provider %q in %s", source, localName, c.Path),
				))
				continue
			}
			fqn = addr
		}
		if fqn.IsZero() {
			fqn = addrs.ImpliedProviderForUnqualifiedType(localName)
		}
		if _, ok := reqs[fqn]; !ok {
			// We'll at least have an unconstrained dependency then, but might
			// add to this in the loop below.
			reqs[fqn] = nil
		}
		for _, constraintsStr := range providerReqs.VersionConstraints {
			if constraintsStr != "" {
				constraints, err := getproviders.ParseVersionConstraints(constraintsStr)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider version constraint",
						fmt.Sprintf("Provider %q in %s has invalid version constraint %q: %s.", localName, c.Path, constraintsStr, err),
					))
					continue
				}
				reqs[fqn] = append(reqs[fqn], constraints...)
			}
		}
	}

	// ...and now we'll recursively visit all of the child modules to merge
	// in their requirements too.
	for _, childConfig := range c.Children {
		moreDiags := childConfig.addProviderRequirements(reqs)
		diags = diags.Append(moreDiags)
	}

	return diags
}

// ProviderDependencies is a deprecated variant of ProviderRequirements which
// uses the moduledeps models for representation. This is preserved to allow
// a gradual transition over to ProviderRequirements, but note that its
// support for fully-qualified provider addresses has some idiosyncracies.
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
		var fqn addrs.Provider
		if source := reqs.Source; source != "" {
			addr, parseDiags := addrs.ParseProviderSourceString(source)
			if parseDiags.HasErrors() {
				diags = diags.Append(wrapDiagnostic(tfconfig.Diagnostic{
					Severity: tfconfig.DiagError,
					Summary:  "Invalid provider source",
					Detail:   fmt.Sprintf("Invalid source %q for provider", name),
				}))
				continue
			}
			fqn = addr
		}
		if fqn.IsZero() {
			fqn = addrs.NewDefaultProvider(name)
		}
		var constraints version.Constraints
		for _, reqStr := range reqs.VersionConstraints {
			if reqStr != "" {
				constraint, err := version.NewConstraint(reqStr)
				if err != nil {
					diags = diags.Append(wrapDiagnostic(tfconfig.Diagnostic{
						Severity: tfconfig.DiagError,
						Summary:  "Invalid provider version constraint",
						Detail:   fmt.Sprintf("Invalid version constraint %q for provider %s.", reqStr, fqn.String()),
					}))
					continue
				}
				constraints = append(constraints, constraint...)
			}
		}
		providers[fqn] = moduledeps.ProviderDependency{
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
