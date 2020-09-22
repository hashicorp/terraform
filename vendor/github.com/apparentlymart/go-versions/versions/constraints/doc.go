// Package constraints contains a high-level representation of version
// constraints that retains enough information for direct analysis and
// serialization as a string.
//
// The package also contains parsers to produce that representation from
// various compact constraint specification formats.
//
// The main "versions" package, available in the parent directory, can consume
// the high-level constraint representation from this package to construct
// a version set that contains all versions meeting the given constraints.
// Package "constraints" does not contain any functionalty for checking versions
// against constraints since that is provided by package "versions".
package constraints
