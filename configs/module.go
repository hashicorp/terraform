package configs

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
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

	Backend              *Backend
	ProviderConfigs      map[string]*Provider
	ProviderRequirements map[string][]VersionConstraint

	Variables map[string]*Variable
	Locals    map[string]*Local
	Outputs   map[string]*Output

	ModuleCalls map[string]*ModuleCall

	ManagedResources map[string]*Resource
	DataResources    map[string]*Resource
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

	Backends             []*Backend
	ProviderConfigs      []*Provider
	ProviderRequirements []*ProviderRequirement

	Variables []*Variable
	Locals    []*Local
	Outputs   []*Output

	ModuleCalls []*ModuleCall

	ManagedResources []*Resource
	DataResources    []*Resource
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
		ProviderConfigs:      map[string]*Provider{},
		ProviderRequirements: map[string][]VersionConstraint{},
		Variables:            map[string]*Variable{},
		Locals:               map[string]*Local{},
		Outputs:              map[string]*Output{},
		ModuleCalls:          map[string]*ModuleCall{},
		ManagedResources:     map[string]*Resource{},
		DataResources:        map[string]*Resource{},
	}

	for _, file := range primaryFiles {
		fileDiags := mod.appendFile(file)
		diags = append(diags, fileDiags...)
	}

	for _, file := range overrideFiles {
		fileDiags := mod.mergeFile(file)
		diags = append(diags, fileDiags...)
	}

	return mod, diags
}

func (m *Module) appendFile(file *File) hcl.Diagnostics {
	var diags hcl.Diagnostics

	for _, constraint := range file.CoreVersionConstraints {
		// If there are any conflicting requirements then we'll catch them
		// when we actually check these constraints.
		m.CoreVersionConstraints = append(m.CoreVersionConstraints, constraint)
	}

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

	for _, reqd := range file.ProviderRequirements {
		m.ProviderRequirements[reqd.Name] = append(m.ProviderRequirements[reqd.Name], reqd.Requirement)
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
				Detail:   fmt.Sprintf("An module call named %q was already defined at %s. Module calls must have unique names within a module.", existing.Name, existing.DeclRange),
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

	return diags
}

func (m *Module) mergeFile(file *File) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if len(file.CoreVersionConstraints) != 0 {
		// This is a bit of a strange case for overriding since we normally
		// would union together across multiple files anyway, but we'll
		// allow it and have each override file clobber any existing list.
		m.CoreVersionConstraints = nil
		for _, constraint := range file.CoreVersionConstraints {
			m.CoreVersionConstraints = append(m.CoreVersionConstraints, constraint)
		}
	}

	if len(file.Backends) != 0 {
		switch len(file.Backends) {
		case 1:
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

	if len(file.ProviderRequirements) != 0 {
		mergeProviderVersionConstraints(m.ProviderRequirements, file.ProviderRequirements)
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
		mergeDiags := existing.merge(r)
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
		mergeDiags := existing.merge(r)
		diags = append(diags, mergeDiags...)
	}

	return diags
}
