// Package zcldec provides a higher-level API for unpacking the content of
// zcl bodies, implemented in terms of the low-level "Content" API exposed
// by the bodies themselves.
//
// It allows decoding an entire nested configuration in a single operation
// by providing a description of the intended structure.
//
// For some applications it may be more convenient to use the "gozcl"
// package, which has a similar purpose but decodes directly into native
// Go data types. zcldec instead targets the cty type system, and thus allows
// a cty-driven application to remain within that type system.
package zcldec
