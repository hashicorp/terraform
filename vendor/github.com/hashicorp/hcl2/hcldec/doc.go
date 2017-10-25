// Package hcldec provides a higher-level API for unpacking the content of
// HCL bodies, implemented in terms of the low-level "Content" API exposed
// by the bodies themselves.
//
// It allows decoding an entire nested configuration in a single operation
// by providing a description of the intended structure.
//
// For some applications it may be more convenient to use the "gohcl"
// package, which has a similar purpose but decodes directly into native
// Go data types. hcldec instead targets the cty type system, and thus allows
// a cty-driven application to remain within that type system.
package hcldec
