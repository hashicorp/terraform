// Package configs contains types that represent Terraform configurations and
// the different elements thereof.
//
// The functionality in this package can be used for some static analyses of
// Terraform configurations, but this package generally exposes representations
// of the configuration source code rather than the result of evaluating these
// objects. The sibling package "lang" deals with evaluation of structures
// and expressions in the configuration.
//
// Due to its close relationship with HCL, this package makes frequent use
// of types from the HCL API, including raw HCL diagnostic messages. Such
// diagnostics can be converted into Terraform-flavored diagnostics, if needed,
// using functions in the sibling package tfdiags.
package configs
