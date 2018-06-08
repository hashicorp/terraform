// Package gohcl allows decoding HCL configurations into Go data structures.
//
// It provides a convenient and concise way of describing the schema for
// configuration and then accessing the resulting data via native Go
// types.
//
// A struct field tag scheme is used, similar to other decoding and
// unmarshalling libraries. The tags are formatted as in the following example:
//
//    ThingType string `hcl:"thing_type,attr"`
//
// Within each tag there are two comma-separated tokens. The first is the
// name of the corresponding construct in configuration, while the second
// is a keyword giving the kind of construct expected. The following
// kind keywords are supported:
//
//    attr (the default) indicates that the value is to be populated from an attribute
//    block indicates that the value is to populated from a block
//    label indicates that the value is to populated from a block label
//    remain indicates that the value is to be populated from the remaining body after populating other fields
//
// "attr" fields may either be of type *hcl.Expression, in which case the raw
// expression is assigned, or of any type accepted by gocty, in which case
// gocty will be used to assign the value to a native Go type.
//
// "block" fields may be of type *hcl.Block or hcl.Body, in which case the
// corresponding raw value is assigned, or may be a struct that recursively
// uses the same tags. Block fields may also be slices of any of these types,
// in which case multiple blocks of the corresponding type are decoded into
// the slice.
//
// "label" fields are considered only in a struct used as the type of a field
// marked as "block", and are used sequentially to capture the labels of
// the blocks being decoded. In this case, the name token is used only as
// an identifier for the label in diagnostic messages.
//
// "remain" can be placed on a single field that may be either of type
// hcl.Body or hcl.Attributes, in which case any remaining body content is
// placed into this field for delayed processing. If no "remain" field is
// present then any attributes or blocks not matched by another valid tag
// will cause an error diagnostic.
//
// Broadly-speaking this package deals with two types of error. The first is
// errors in the configuration itself, which are returned as diagnostics
// written with the configuration author as the target audience. The second
// is bugs in the calling program, such as invalid struct tags, which are
// surfaced via panics since there can be no useful runtime handling of such
// errors and they should certainly not be returned to the user as diagnostics.
package gohcl
