// Package testharness implements a test harness for Terraform modules and
// configurations.
//
// The test harness is intended to provide a convenient way to make test
// assertions against a Terraform state, both directly (writing assertions
// directly inside the test specification files) and indirectly (delegating
// to external testing tools such as serverspec, passing connection information
// from Terraform state).
//
// The test framework uses test specifications written in a DSL based on the
// Lua language. Lua is used because it is a small language with a clean
// syntax that is easy to integrate closely with Terraform. In practice, the
// author or reader of tests is not required to have extensive Lua knowledge
// since test specifications primarily interact with a high-level specification
// API provided by this package, rather than with the Lua standard library and
// advanced Lua language features.
package testharness
