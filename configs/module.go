package configs

import (
	"github.com/hashicorp/hcl2/hcl"
)

// Module is a container for a set of configuration constructs that are
// evaluated within a common namespace.
type Module struct {
	CoreVersionConstraints []VersionConstraint

	Backend              *Backend
	ProviderConfigs      map[string]*Provider
	ProviderRequirements map[string][]VersionConstraint

	Variables map[string]*Variable
	Locals    map[string]*Local
	Outputs   map[string]*Output

	ModuleCalls map[string]*ModuleCall

	ManagedResources map[string]*ManagedResource
	DataResources    map[string]*DataResource
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
	// TODO: process each file in turn, combining and merging as necessary
	// to produce a single flat *Module.
	panic("NewModule not yet implemented")
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

	ManagedResources []*ManagedResource
	DataResources    []*DataResource
}
