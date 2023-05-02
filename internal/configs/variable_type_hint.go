// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

// VariableTypeHint is an enumeration used for the Variable.TypeHint field,
// which is an incompletely-specified type for the variable which is used
// as a hint for whether a value provided in an ambiguous context (on the
// command line or in an environment variable) should be taken literally as a
// string or parsed as an HCL expression to produce a data structure.
//
// The type hint is applied to runtime values as well, but since it does not
// accurately describe a precise type it is not fully-sufficient to infer
// the dynamic type of a value passed through a variable.
//
// These hints use inaccurate terminology for historical reasons. Full details
// are in the documentation for each constant in this enumeration, but in
// summary:
//
//   - TypeHintString requires a primitive type
//   - TypeHintList requires a type that could be converted to a tuple
//   - TypeHintMap requires a type that could be converted to an object
type VariableTypeHint rune

//go:generate go run golang.org/x/tools/cmd/stringer -type VariableTypeHint

// TypeHintNone indicates the absence of a type hint. Values specified in
// ambiguous contexts will be treated as literal strings, as if TypeHintString
// were selected, but no runtime value checks will be applied. This is reasonable
// type hint for a module that is never intended to be used at the top-level
// of a configuration, since descendent modules never receive values from
// ambiguous contexts.
const TypeHintNone VariableTypeHint = 0

// TypeHintString spec indicates that a value provided in an ambiguous context
// should be treated as a literal string, and additionally requires that the
// runtime value for the variable is of a primitive type (string, number, bool).
const TypeHintString VariableTypeHint = 'S'

// TypeHintList indicates that a value provided in an ambiguous context should
// be treated as an HCL expression, and additionally requires that the
// runtime value for the variable is of an tuple, list, or set type.
const TypeHintList VariableTypeHint = 'L'

// TypeHintMap indicates that a value provided in an ambiguous context should
// be treated as an HCL expression, and additionally requires that the
// runtime value for the variable is of an object or map type.
const TypeHintMap VariableTypeHint = 'M'
