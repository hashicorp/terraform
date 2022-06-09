// Package checks has the logic to deal with the various kinds of self-checks
// that Terraform configurations can define, including:
//   - Preconditions and postconditions for resources.
//   - Preconditions for output values.
//
// Terraform Core (the "terraform" package) is responsible for actually
// evaluating and handling the results of checks at the appropriate time in
// various operations, but it relies on parts of this package in various ways
// when doing so.
package checks
