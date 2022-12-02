package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/experiments"
)

// Module is a container for a set of configuration constructs that are
// evaluated within a common namespace.
type Module struct {
	// SourceDir is the filesystem directory that the module was loaded from.
	//
	// This is populated automatically only for configurations loaded with
	// LoadConfigDir. If the parser is using a virtual filesystem then the
	// path here will be in terms of that virtual filesystem.

	// Any other caller that constructs a module directly with NewModule may
	// assign a suitable value to this attribute before using it for other
	// purposes. It should be treated as immutable by all consumers of Module
	// values.
	SourceDir string

	CoreVersionConstraints []VersionConstraint

	ActiveExperiments experiments.Set

	Backend              *Backend
	CloudConfig          *CloudConfig
	ProviderConfigs      map[string]*Provider
	ProviderRequirements *RequiredProviders
	ProviderLocalNames   map[addrs.Provider]string
	ProviderMetas        map[addrs.Provider]*ProviderMeta

	Variables map[string]*Variable
	Locals    map[string]*Local
	Outputs   map[string]*Output

	ModuleCalls map[string]*ModuleCall

	ManagedResources map[string]*Resource
	DataResources    map[string]*Resource

	SmokeTests map[string]*SmokeTest

	Moved []*Moved
}

// File describes the contents of a single configuration file.
//
// Individual files are not usually used alone, but rather combined together
// with other files (conventionally, those in the same directory) to produce
// a *Module, using NewModule.
//
// At the level of an individual file we represent directly the structural
// elements present in the file, without any attempt to detect conflicting
// declarations. A File object can therefore be used for some basic static
// analysis of individual elements, but must be built into a Module to detect
// duplicate declarations.
type File struct {
	CoreVersionConstraints []VersionConstraint

	ActiveExperiments experiments.Set

	Backends          []*Backend
	CloudConfigs      []*CloudConfig
	ProviderConfigs   []*Provider
	ProviderMetas     []*ProviderMeta
	RequiredProviders []*RequiredProviders

	Variables []*Variable
	Locals    []*Local
	Outputs   []*Output

	ModuleCalls []*ModuleCall

	ManagedResources []*Resource
	DataResources    []*Resource

	SmokeTests []*SmokeTest

	Moved []*Moved
}

// NewModule takes a list of primary files and a list of override files and
// produces a *Module by combining the files together.
//
// If there are any conflicting declarations in the given files -- for example,
// if the same variable name is defined twice -- then the resulting module
// will be incomplete and error diagnostics will be returned. Careful static
// analysis of the returned Module is still possible in this case, but the
// module will probably not be semantically valid.
func NewModule(primaryFiles, overrideFiles []*File) (*Module, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	mod := &Module{
		ProviderConfigs:    map[string]*Provider{},
		ProviderLocalNames: map[addrs.Provider]string{},
		Variables:          map[string]*Variable{},
		Locals:             map[string]*Local{},
		Outputs:            map[string]*Output{},
		ModuleCalls:        map[string]*ModuleCall{},
		ManagedResources:   map[string]*Resource{},
		DataResources:      map[string]*Resource{},
		SmokeTests:         map[string]*SmokeTest{},
		ProviderMetas:      map[addrs.Provider]*ProviderMeta{},
	}

	// Process the required_providers blocks first, to ensure that all
	// resources have access to the correct provider FQNs
	for _, file := range primaryFiles {
		for _, r := range file.RequiredProviders {
			if mod.ProviderRequirements != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate required providers configuration",
					Detail:   fmt.Sprintf("A module may have only one required providers configuration. The required providers were previously configured at %s.", mod.ProviderRequirements.DeclRange),
					Subject:  &r.DeclRange,
				})
				continue
			}
			mod.ProviderRequirements = r
		}
	}

	// If no required_providers block is configured, create a useful empty
	// state to reduce nil checks elsewhere
	if mod.ProviderRequirements == nil {
		mod.ProviderRequirements = &RequiredProviders{
			RequiredProviders: make(map[string]*RequiredProvider),
		}
	}

	// Any required_providers blocks in override files replace the entire
	// block for each provider
	for _, file := range overrideFiles {
		for _, override := range file.RequiredProviders {
			for name, rp := range override.RequiredProviders {
				mod.ProviderRequirements.RequiredProviders[name] = rp
			}
		}
	}

	for _, file := range primaryFiles {
		fileDiags := mod.appendFile(file)
		diags = append(diags, fileDiags...)
	}

	for _, file := range overrideFiles {
		fileDiags := mod.mergeFile(file)
		diags = append(diags, fileDiags...)
	}

	diags = append(diags, checkModuleExperiments(mod)...)

	// Generate the FQN -> LocalProviderName map
	mod.gatherProviderLocalNames()

	return mod, diags
}

// ResourceByAddr returns the configuration for the resource with the given
// address, or nil if there is no such resource.
func (m *Module) ResourceByAddr(addr addrs.Resource) *Resource {
	key := addr.String()
	switch addr.Mode {
	case addrs.ManagedResourceMode:
		return m.ManagedResources[key]
	case addrs.DataResourceMode:
		return m.DataResources[key]
	default:
		return nil
	}
}

func (m *Module) appendFile(file *File) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// If there are any conflicting requirements then we'll catch them
	// when we actually check these constraints.
	m.CoreVersionConstraints = append(m.CoreVersionConstraints, file.CoreVersionConstraints...)

	m.ActiveExperiments = experiments.SetUnion(m.ActiveExperiments, file.ActiveExperiments)

	for _, b := range file.Backends {
		if m.Backend != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate backend configuration",
				Detail:   fmt.Sprintf("A module may have only one backend configuration. The backend was previously configured at %s.", m.Backend.DeclRange),
				Subject:  &b.DeclRange,
			})
			continue
		}
		m.Backend = b
	}

	for _, c := range file.CloudConfigs {
		if m.CloudConfig != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate Terraform Cloud configurations",
				Detail:   fmt.Sprintf("A module may have only one 'cloud' block configuring Terraform Cloud. Terraform Cloud was previously configured at %s.", m.CloudConfig.DeclRange),
				Subject:  &c.DeclRange,
			})
			continue
		}

		m.CloudConfig = c
	}

	if m.Backend != nil && m.CloudConfig != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Both a backend and Terraform Cloud configuration are present",
			Detail:   fmt.Sprintf("A module may declare either one 'cloud' block configuring Terraform Cloud OR one 'backend' block configuring a state backend. Terraform Cloud is configured at %s; a backend is configured at %s. Remove the backend block to configure Terraform Cloud.", m.CloudConfig.DeclRange, m.Backend.DeclRange),
			Subject:  &m.Backend.DeclRange,
		})
	}

	for _, pc := range file.ProviderConfigs {
		key := pc.moduleUniqueKey()
		if existing, exists := m.ProviderConfigs[key]; exists {
			if existing.Alias == "" {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate provider configuration",
					Detail:   fmt.Sprintf("A default (non-aliased) provider configuration for %q was already given at %s. If multiple configurations are required, set the \"alias\" argument for alternative configurations.", existing.Name, existing.DeclRange),
					Subject:  &pc.DeclRange,
				})
			} else {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate provider configuration",
					Detail:   fmt.Sprintf("A provider configuration for %q with alias %q was already given at %s. Each configuration for the same provider must have a distinct alias.", existing.Name, existing.Alias, existing.DeclRange),
					Subject:  &pc.DeclRange,
				})
			}
			continue
		}
		m.ProviderConfigs[key] = pc
	}

	for _, pm := range file.ProviderMetas {
		provider := m.ProviderForLocalConfig(addrs.LocalProviderConfig{LocalName: pm.Provider})
		if existing, exists := m.ProviderMetas[provider]; exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate provider_meta block",
				Detail:   fmt.Sprintf("A provider_meta block for provider %q was already declared at %s. Providers may only have one provider_meta block per module.", existing.Provider, existing.DeclRange),
				Subject:  &pm.DeclRange,
			})
		}
		m.ProviderMetas[provider] = pm
	}

	for _, v := range file.Variables {
		if existing, exists := m.Variables[v.Name]; exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate variable declaration",
				Detail:   fmt.Sprintf("A variable named %q was already declared at %s. Variable names must be unique within a module.", existing.Name, existing.DeclRange),
				Subject:  &v.DeclRange,
			})
		}
		m.Variables[v.Name] = v
	}

	for _, l := range file.Locals {
		if existing, exists := m.Locals[l.Name]; exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate local value definition",
				Detail:   fmt.Sprintf("A local value named %q was already defined at %s. Local value names must be unique within a module.", existing.Name, existing.DeclRange),
				Subject:  &l.DeclRange,
			})
		}
		m.Locals[l.Name] = l
	}

	for _, o := range file.Outputs {
		if existing, exists := m.Outputs[o.Name]; exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate output definition",
				Detail:   fmt.Sprintf("An output named %q was already defined at %s. Output names must be unique within a module.", existing.Name, existing.DeclRange),
				Subject:  &o.DeclRange,
			})
		}
		m.Outputs[o.Name] = o
	}

	for _, mc := range file.ModuleCalls {
		if existing, exists := m.ModuleCalls[mc.Name]; exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate module call",
				Detail:   fmt.Sprintf("A module call named %q was already defined at %s. Module calls must have unique names within a module.", existing.Name, existing.DeclRange),
				Subject:  &mc.DeclRange,
			})
		}
		m.ModuleCalls[mc.Name] = mc
	}

	for _, r := range file.ManagedResources {
		key := r.moduleUniqueKey()
		if existing, exists := m.ManagedResources[key]; exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Duplicate resource %q configuration", existing.Type),
				Detail:   fmt.Sprintf("A %s resource named %q was already declared at %s. Resource names must be unique per type in each module.", existing.Type, existing.Name, existing.DeclRange),
				Subject:  &r.DeclRange,
			})
			continue
		}
		m.ManagedResources[key] = r

		// set the provider FQN for the resource
		if r.ProviderConfigRef != nil {
			r.Provider = m.ProviderForLocalConfig(r.ProviderConfigAddr())
		} else {
			// an invalid resource name (for e.g. "null resource" instead of
			// "null_resource") can cause a panic down the line in addrs:
			// https://github.com/hashicorp/terraform/issues/25560
			implied, err := addrs.ParseProviderPart(r.Addr().ImpliedProvider())
			if err == nil {
				r.Provider = m.ImpliedProviderForUnqualifiedType(implied)
			}
			// We don't return a diagnostic because the invalid resource name
			// will already have been caught.
		}
	}

	for _, r := range file.DataResources {
		key := r.moduleUniqueKey()
		if existing, exists := m.DataResources[key]; exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Duplicate data %q configuration", existing.Type),
				Detail:   fmt.Sprintf("A %s data resource named %q was already declared at %s. Resource names must be unique per type in each module.", existing.Type, existing.Name, existing.DeclRange),
				Subject:  &r.DeclRange,
			})
			continue
		}
		m.DataResources[key] = r
	}

	for _, st := range file.SmokeTests {
		// Any data resources declared inside a smoke test block live in the
		// same namespace as top-level data resources, so we'll merge those
		// here and generate specialized errors at this point if there are
		// any colliding names between top-level and smoke test, between
		// two different smoke tests, or within the same smoke test.
		for _, r := range st.DataResources {
			key := r.moduleUniqueKey()
			if existing, exists := m.DataResources[key]; exists {
				switch {
				case existing.SmokeTest == nil:
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  fmt.Sprintf("Duplicate data %q configuration", existing.Type),
						Detail:   fmt.Sprintf("A %s data resource named %q was already declared at %s. Resource names must be unique per type in each module, including within smoke tests.", existing.Type, existing.Name, existing.DeclRange),
						Subject:  r.DeclRange.Ptr(),
					})
				case existing.SmokeTest.Name == st.Name:
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  fmt.Sprintf("Duplicate data %q configuration", existing.Type),
						Detail:   fmt.Sprintf("A %s data resource named %q was already declared at %s. Resource names must be unique per type in each module.", existing.Type, existing.Name, existing.DeclRange),
						Subject:  r.DeclRange.Ptr(),
					})
				default:
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  fmt.Sprintf("Duplicate data %q configuration", existing.Type),
						Detail:   fmt.Sprintf("A %s data resource named %q was already declared at %s for smoke_test %q. Resource names must be unique per type in each module, including across all smoke tests.", existing.Type, existing.Name, existing.DeclRange, existing.SmokeTest.Name),
						Subject:  r.DeclRange.Ptr(),
					})
				}
				continue
			}
			m.DataResources[key] = r
		}

		if existing, exists := m.SmokeTests[st.Name]; exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate smoke test name",
				Detail:   fmt.Sprintf("A smoke test named %q was already declared at %s. Smoke test names must be unique within each module.", existing.Name, existing.DeclRange),
				Subject:  st.DeclRange.Ptr(),
			})
			continue
		}
		m.SmokeTests[st.Name] = st
	}

	// We must now deal with the provider associations for the merged set of
	// data resources, across both top-level ones and smoke test ones.
	for _, r := range m.DataResources {
		// set the provider FQN for the resource
		if r.ProviderConfigRef != nil {
			r.Provider = m.ProviderForLocalConfig(r.ProviderConfigAddr())
		} else {
			// an invalid data source name (for e.g. "null resource" instead of
			// "null_resource") can cause a panic down the line in addrs:
			// https://github.com/hashicorp/terraform/issues/25560
			implied, err := addrs.ParseProviderPart(r.Addr().ImpliedProvider())
			if err == nil {
				r.Provider = m.ImpliedProviderForUnqualifiedType(implied)
			}
			// We don't return a diagnostic because the invalid resource name
			// will already have been caught.
		}
	}

	// "Moved" blocks just append, because they are all independent
	// of one another at this level. (We handle any references between
	// them at runtime.)
	m.Moved = append(m.Moved, file.Moved...)

	return diags
}

func (m *Module) mergeFile(file *File) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if len(file.CoreVersionConstraints) != 0 {
		// This is a bit of a strange case for overriding since we normally
		// would union together across multiple files anyway, but we'll
		// allow it and have each override file clobber any existing list.
		m.CoreVersionConstraints = nil
		m.CoreVersionConstraints = append(m.CoreVersionConstraints, file.CoreVersionConstraints...)
	}

	if len(file.Backends) != 0 {
		switch len(file.Backends) {
		case 1:
			m.CloudConfig = nil // A backend block is mutually exclusive with a cloud one, and overwrites any cloud config
			m.Backend = file.Backends[0]
		default:
			// An override file with multiple backends is still invalid, even
			// though it can override backends from _other_ files.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate backend configuration",
				Detail:   fmt.Sprintf("Each override file may have only one backend configuration. A backend was previously configured at %s.", file.Backends[0].DeclRange),
				Subject:  &file.Backends[1].DeclRange,
			})
		}
	}

	if len(file.CloudConfigs) != 0 {
		switch len(file.CloudConfigs) {
		case 1:
			m.Backend = nil // A cloud block is mutually exclusive with a backend one, and overwrites any backend
			m.CloudConfig = file.CloudConfigs[0]
		default:
			// An override file with multiple cloud blocks is still invalid, even
			// though it can override cloud/backend blocks from _other_ files.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate Terraform Cloud configurations",
				Detail:   fmt.Sprintf("A module may have only one 'cloud' block configuring Terraform Cloud. Terraform Cloud was previously configured at %s.", file.CloudConfigs[0].DeclRange),
				Subject:  &file.CloudConfigs[1].DeclRange,
			})
		}
	}

	for _, pc := range file.ProviderConfigs {
		key := pc.moduleUniqueKey()
		existing, exists := m.ProviderConfigs[key]
		if pc.Alias == "" {
			// We allow overriding a non-existing _default_ provider configuration
			// because the user model is that an absent provider configuration
			// implies an empty provider configuration, which is what the user
			// is therefore overriding here.
			if exists {
				mergeDiags := existing.merge(pc)
				diags = append(diags, mergeDiags...)
			} else {
				m.ProviderConfigs[key] = pc
			}
		} else {
			// For aliased providers, there must be a base configuration to
			// override. This allows us to detect and report alias typos
			// that might otherwise cause the override to not apply.
			if !exists {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing base provider configuration for override",
					Detail:   fmt.Sprintf("There is no %s provider configuration with the alias %q. An override file can only override an aliased provider configuration that was already defined in a primary configuration file.", pc.Name, pc.Alias),
					Subject:  &pc.DeclRange,
				})
				continue
			}
			mergeDiags := existing.merge(pc)
			diags = append(diags, mergeDiags...)
		}
	}

	for _, v := range file.Variables {
		existing, exists := m.Variables[v.Name]
		if !exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing base variable declaration to override",
				Detail:   fmt.Sprintf("There is no variable named %q. An override file can only override a variable that was already declared in a primary configuration file.", v.Name),
				Subject:  &v.DeclRange,
			})
			continue
		}
		mergeDiags := existing.merge(v)
		diags = append(diags, mergeDiags...)
	}

	for _, l := range file.Locals {
		existing, exists := m.Locals[l.Name]
		if !exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing base local value definition to override",
				Detail:   fmt.Sprintf("There is no local value named %q. An override file can only override a local value that was already defined in a primary configuration file.", l.Name),
				Subject:  &l.DeclRange,
			})
			continue
		}
		mergeDiags := existing.merge(l)
		diags = append(diags, mergeDiags...)
	}

	for _, o := range file.Outputs {
		existing, exists := m.Outputs[o.Name]
		if !exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing base output definition to override",
				Detail:   fmt.Sprintf("There is no output named %q. An override file can only override an output that was already defined in a primary configuration file.", o.Name),
				Subject:  &o.DeclRange,
			})
			continue
		}
		mergeDiags := existing.merge(o)
		diags = append(diags, mergeDiags...)
	}

	for _, mc := range file.ModuleCalls {
		existing, exists := m.ModuleCalls[mc.Name]
		if !exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing module call to override",
				Detail:   fmt.Sprintf("There is no module call named %q. An override file can only override a module call that was defined in a primary configuration file.", mc.Name),
				Subject:  &mc.DeclRange,
			})
			continue
		}
		mergeDiags := existing.merge(mc)
		diags = append(diags, mergeDiags...)
	}

	for _, r := range file.ManagedResources {
		key := r.moduleUniqueKey()
		existing, exists := m.ManagedResources[key]
		if !exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing resource to override",
				Detail:   fmt.Sprintf("There is no %s resource named %q. An override file can only override a resource block defined in a primary configuration file.", r.Type, r.Name),
				Subject:  &r.DeclRange,
			})
			continue
		}
		mergeDiags := existing.merge(r, m.ProviderRequirements.RequiredProviders)
		diags = append(diags, mergeDiags...)
	}

	for _, r := range file.DataResources {
		key := r.moduleUniqueKey()
		existing, exists := m.DataResources[key]
		if !exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing data resource to override",
				Detail:   fmt.Sprintf("There is no %s data resource named %q. An override file can only override a data block defined in a primary configuration file.", r.Type, r.Name),
				Subject:  &r.DeclRange,
			})
			continue
		}
		mergeDiags := existing.merge(r, m.ProviderRequirements.RequiredProviders)
		diags = append(diags, mergeDiags...)
	}

	for _, m := range file.Moved {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot override 'moved' blocks",
			Detail:   "Records of moved objects can appear only in normal files, not in override files.",
			Subject:  m.DeclRange.Ptr(),
		})
	}

	return diags
}

// gatherProviderLocalNames is a helper function that populatesA a map of
// provider FQNs -> provider local names. This information is useful for
// user-facing output, which should include both the FQN and LocalName. It must
// only be populated after the module has been parsed.
func (m *Module) gatherProviderLocalNames() {
	providers := make(map[addrs.Provider]string)
	for k, v := range m.ProviderRequirements.RequiredProviders {
		providers[v.Type] = k
	}
	m.ProviderLocalNames = providers
}

// LocalNameForProvider returns the module-specific user-supplied local name for
// a given provider FQN, or the default local name if none was supplied.
func (m *Module) LocalNameForProvider(p addrs.Provider) string {
	if existing, exists := m.ProviderLocalNames[p]; exists {
		return existing
	} else {
		// If there isn't a map entry, fall back to the default:
		// Type = LocalName
		return p.Type
	}
}

// ProviderForLocalConfig returns the provider FQN for a given
// LocalProviderConfig, based on its local name.
func (m *Module) ProviderForLocalConfig(pc addrs.LocalProviderConfig) addrs.Provider {
	return m.ImpliedProviderForUnqualifiedType(pc.LocalName)
}

// ImpliedProviderForUnqualifiedType returns the provider FQN for a given type,
// first by looking up the type in the provider requirements map, and falling
// back to an implied default provider.
//
// The intended behaviour is that configuring a provider with local name "foo"
// in a required_providers block will result in resources with type "foo" using
// that provider.
func (m *Module) ImpliedProviderForUnqualifiedType(pType string) addrs.Provider {
	if provider, exists := m.ProviderRequirements.RequiredProviders[pType]; exists {
		return provider.Type
	}
	return addrs.ImpliedProviderForUnqualifiedType(pType)
}
