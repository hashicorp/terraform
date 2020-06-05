// Package json is the JSON parser for HCL. It parses JSON files and returns
// implementations of the core HCL structural interfaces in terms of the
// JSON data inside.
//
// This is not a generic JSON parser. Instead, it deals with the mapping from
// the JSON information model to the HCL information model, using a number
// of hard-coded structural conventions.
//
// In most cases applications will not import this package directly, but will
// instead access its functionality indirectly through functions in the main
// "hcl" package and in the "hclparse" package.
package json
