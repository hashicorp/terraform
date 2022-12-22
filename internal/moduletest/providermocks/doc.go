// Package providermocks implements our HCL-based language for configuring
// mock provider behaviors.
//
// A mock provider shares the same schema as the real provider it represents,
// and executes the same validation logic, but it never actually configures
// that provider or asks it to perform any operations that normally occur only
// after a provider is configured.
//
// Instead, the plan, apply, and read steps are faked by looking up a matching
// response from a table configured using an HCL-based DSL.
package providermocks
