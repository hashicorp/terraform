// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package states

// Generation is used to represent multiple objects in a succession of objects
// represented by a single resource instance address. A resource instance can
// have multiple generations over its lifetime due to object replacement
// (when a change can't be applied without destroying and re-creating), and
// multiple generations can exist at the same time when create_before_destroy
// is used.
//
// A Generation value can either be the value of the variable "CurrentGen" or
// a value of type DeposedKey. Generation values can be compared for equality
// using "==" and used as map keys. The zero value of Generation (nil) is not
// a valid generation and must not be used.
type Generation interface {
	generation()
}

// CurrentGen is the Generation representing the currently-active object for
// a resource instance.
var CurrentGen Generation
