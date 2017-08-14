// Package configschema contains types for describing the expected structure
// of a configuration block whose shape is not known until runtime.
//
// For example, this is used to describe the expected contents of a resource
// configuration block, which is defined by the corresponding provider plugin
// and thus not compiled into Terraform core.
//
// This package should not be confused with the package helper/schema, which
// is the higher-level helper library used to implement providers themselves.
package configschema
