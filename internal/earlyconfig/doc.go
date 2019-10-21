// Package earlyconfig is a specialized alternative to the top-level "configs"
// package that does only shallow processing of configuration and is therefore
// able to be much more liberal than the full config loader in what it accepts.
//
// In particular, it can accept both current and legacy HCL syntax, and it
// ignores top-level blocks that it doesn't recognize. These two characteristics
// make this package ideal for dependency-checking use-cases so that we are
// more likely to be able to return an error message about an explicit
// incompatibility than to return a less-actionable message about a construct
// not being supported.
//
// However, its liberal approach also means it should be used sparingly. It
// exists primarily for "terraform init", so that it is able to detect
// incompatibilities more robustly when installing dependencies. For most
// other use-cases, use the "configs" and "configs/configload" packages.
//
// Package earlyconfig is a wrapper around the terraform-config-inspect
// codebase, adding to it just some helper functionality for Terraform's own
// use-cases.
package earlyconfig
