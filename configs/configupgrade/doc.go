// Package configupgrade upgrades configurations targeting our legacy
// configuration loader (in package "config") to be compatible with and
// idiomatic for the newer configuration loader (in package "configs").
//
// It works on one module directory at a time, producing new content for
// each existing .tf file and possibly creating new files as needed. The
// legacy HCL and HIL parsers are used to read the existing configuration
// for maximum compatibility with any non-idiomatic constructs that were
// accepted by those implementations but not accepted by the new HCL parsers.
//
// Unlike the loaders and validators elsewhere in Terraform, this package
// always generates diagnostics with paths relative to the module directory
// currently being upgraded, with no intermediate paths. This means that the
// filenames in these ranges can be used directly as keys into the ModuleSources
// map that the file was parsed from.
package configupgrade
