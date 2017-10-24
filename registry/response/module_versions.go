package response

import (
	"github.com/hashicorp/terraform-registry/api/regsrc"

	"github.com/hashicorp/terraform-registry/api/models"
)

// ModuleVersions is the response format that contains all metadata about module
// versions needed for terraform CLI to resolve version constraints. See RFC
// TF-042 for details on this format.
type ModuleVersions struct {
	Modules []*ModuleProviderVersions `json:"modules"`
}

// ModuleProviderVersions is the response format for a single module instance,
// containing metadata about all versions and their dependencies.
type ModuleProviderVersions struct {
	Source   string           `json:"source"`
	Versions []*ModuleVersion `json:"versions"`
}

// ModuleVersion is the output metadata for a given version needed by CLI to
// resolve candidate versions to satisfy requirements.
type ModuleVersion struct {
	Version    string              `json:"version"`
	Root       VersionSubmodule    `json:"root"`
	Submodules []*VersionSubmodule `json:"submodules"`
}

// VersionSubmodule is the output metadata for a submodule within a given
// version needed by CLI to resolve candidate versions to satisfy requirements.
// When representing the Root in JSON the path is omitted.
type VersionSubmodule struct {
	Path         string               `json:"path,omitempty"`
	Providers    []*ModuleProviderDep `json:"providers"`
	Dependencies []*ModuleDep         `json:"dependencies"`
}

// NewModuleVersions populates a ModuleVersions response based on a slice of
// ModuleProviders. It is assumed these are fully populated with all versions
// submodules and dependencies etc, required in the response, and in the desired
// order (i.e. the first mp is the specific one requested and any others are
// optionally pre-fetched dependencies.) The host is needed to generate correct
// Source strings for all modules and must be the canonical hostname for the
// registry instance.
func NewModuleVersions(mps []*models.ModuleProvider,
	host regsrc.FriendlyHost) *ModuleVersions {

	mods := make([]*ModuleProviderVersions, 0, len(mps))
	for _, mp := range mps {
		mods = append(mods, NewModuleProviderVersions(mp, host))
	}

	return &ModuleVersions{
		Modules: mods,
	}
}

// NewModuleProviderVersions constructs the metadata about a specific module
// for the ModuleVersions response.
func NewModuleProviderVersions(mp *models.ModuleProvider,
	host regsrc.FriendlyHost) *ModuleProviderVersions {

	src := regsrc.NewModule(
		host.String(),
		mp.Module.Namespace,
		mp.Module.Name,
		mp.Provider,
		"",
	)

	versions := make([]*ModuleVersion, 0, len(mp.Versions))
	for _, mv := range mp.Versions {
		versions = append(versions, NewModuleVersion(&mv))
	}

	return &ModuleProviderVersions{
		Source:   src.Display(),
		Versions: versions,
	}
}

// NewModuleVersion constructs the metadata about a specific module version
// for the ModuleVersions response.
func NewModuleVersion(mv *models.ModuleVersion) *ModuleVersion {
	// Build the submodule response objects
	var submodules []*VersionSubmodule
	var submoduleRoot VersionSubmodule
	for _, sub := range mv.Submodules {
		resp := NewVersionSubmodule(&sub)

		if sub.Root() {
			submoduleRoot = *resp
		} else {
			submodules = append(submodules, resp)
		}
	}

	return &ModuleVersion{
		Version:    mv.Version,
		Root:       submoduleRoot,
		Submodules: submodules,
	}
}

// NewVersionSubmodule constructs a representation of a submodule within a
// specific module version for the ModuleVersions response.
func NewVersionSubmodule(m *models.ModuleSubmodule) *VersionSubmodule {
	providerDeps := make([]*ModuleProviderDep, 0, len(m.ProviderDependencies))
	for _, v := range m.ProviderDependencies {
		providerDeps = append(providerDeps, &ModuleProviderDep{
			Name:    v.Provider,
			Version: v.VersionConstraints,
		})
	}

	deps := make([]*ModuleDep, 0, len(m.Dependencies))
	for _, v := range m.Dependencies {
		deps = append(deps, &ModuleDep{
			Name:   v.Name,
			Source: v.Source,
		})
	}

	return &VersionSubmodule{
		Path:         m.Path,
		Providers:    providerDeps,
		Dependencies: deps,
	}
}
