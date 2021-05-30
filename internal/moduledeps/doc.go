// Package moduledeps contains types that can be used to describe the
// providers required for all of the modules in a module tree.
//
// It does not itself contain the functionality for populating such
// data structures; that's in Terraform core, since this package intentionally
// does not depend on terraform core to avoid package dependency cycles.
package moduledeps
