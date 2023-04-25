package sourcebundle

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// DependencyFinder is responsible for analyzing some artifact from a module
// package and reporting any other artifacts it depends on.
//
// The package builder uses this to chase down recursive dependencies which
// might require other module packages that were not included yet. For example,
// when building a source package for Terraform modules the dependency finder
// would presumably treat the given directory as the path to a Terraform
// module and produce one dependency report for each child "module" block.
type DependencyFinder interface {
	// FindDependencies analyzes the directory or file at localPath in whatever
	// way is appropriate for the [DependencyFinder] implementation, and calls
	// the announce function for each dependency it detects.
	//
	// The second argument to announce is the [DependencyFinder] to use when
	// analyzing the _new_ source location, which does not necessarily need to
	// be the same as the receiver of this call, for situtions where one kind
	// of artifact calls into another kind of artifact.
	FindDependencies(src addrs.ModuleSourceRemote, localPath string, deps Dependencies) tfdiags.Diagnostics
}

// Dependencies is the interface used by a [DependencyFinder] to report the
// dependencies it has discovered back to the requesting [Builder].
type Dependencies interface {
	// AddRemoteSource announces a dependency on a module from a remote
	// module package, that should be analyzed using the given dependency
	// finder.
	//
	// srcRng is optional and if set should be a source range that a user
	// would recognize as them configuring the source address given in addr.
	AddRemoteSource(addr addrs.ModuleSourceRemote, depFinder DependencyFinder, srcRng *tfdiags.SourceRange)

	// AddRegistrySource announces a dependency on a module from a module
	// package delivered indirectly through a module registry, that should be
	// analyzed using the given dependency finder.
	//
	// srcRng is optional and if set should be a source range that a user
	// would recognize as them configuring the source address given in addr.
	AddRegistrySource(addr addrs.ModuleSourceRegistry, depFinder DependencyFinder, srcRng *tfdiags.SourceRange)

	// AddLocalSource announces a dependency on a module at a local path
	// relative to the module being analyzed, which must therefore belong to
	// the same module package as that calling module. It will be analyzed
	// using the given dependency finder.
	//
	// If the given relative path traverses out of the module package root
	// then the calling [Builder] will reject it by returning an error to
	// the user which blames the module package as being invalid.
	//
	// srcRng is optional and if set should be a source range that a user
	// would recognize as them configuring the source address given in addr.
	AddLocalSource(addr addrs.ModuleSourceLocal, depFinder DependencyFinder, srcRng *tfdiags.SourceRange)
}
