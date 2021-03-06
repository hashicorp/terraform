// Package exprstress and its subdirectories are utilities for stress-testing
// Terraform's expression evaluation using randomly-generated expressions which
// ought to be valid and then raising an error if evaluation fails.
//
// It also does some limited verification of the expression result value
// by propagating some cross-cutting characteristics like expected result type,
// and whether the result is unknown, null, and/or sensitive. However, we
// don't try to fully model the final result value because it would be
// burdensome and of questionable value to have to, in effect, maintain a
// parallel implementation of each language operator or function to check
// against.
package exprstress
