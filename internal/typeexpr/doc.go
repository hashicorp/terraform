// Package typeexpr is a fork of github.com/hashicorp/hcl/v2/ext/typeexpr
// which has additional experimental support for optional attributes.
//
// This is here as part of the module_variable_optional_attrs experiment.
// If that experiment is successful, the changes here may be upstreamed into
// HCL itself or, if we deem it to be Terraform-specific, we should at least
// update this documentation to reflect that this is now the primary
// Terraform-specific type expression implementation, separate from the
// upstream HCL one.
package typeexpr
