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
	FindDependencies(src addrs.ModuleSourceRemote, localPath string, announce func(newSrc addrs.ModuleSource, srcRng *tfdiags.SourceRange, depFinder DependencyFinder)) tfdiags.Diagnostics
}
