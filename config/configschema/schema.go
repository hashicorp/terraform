package configschema

import (
	"github.com/zclconf/go-cty/cty"
)

// Block represents a configuration block.
//
// "Block" here is a logical grouping construct, though it happens to map
// directly onto the physical block syntax of Terraform's native configuration
// syntax. It may be a more a matter of convention in other syntaxes, such as
// JSON.
//
// When converted to a value, a Block always becomes an instance of an object
// type derived from its defined attributes and nested blocks
type Block struct {
	// Attributes describes any attributes that may appear directly inside
	// the block.
	Attributes map[string]*Attribute

	// BlockTypes describes any nested block types that may appear directly
	// inside the block.
	BlockTypes map[string]*NestedBlock
}

// Attribute represents a configuration attribute, within a block.
type Attribute struct {
	// Type is a type specification that the attribute's value must conform to.
	Type cty.Type

	// Required, if set to true, specifies that an omitted or null value is
	// not permitted.
	Required bool

	// Optional, if set to true, specifies that an omitted or null value is
	// permitted. This field conflicts with Required.
	Optional bool

	// Computed, if set to true, specifies that the value comes from the
	// provider rather than from configuration. If combined with Optional,
	// then the config may optionally provide an overridden value.
	Computed bool
}

// NestedBlock represents the embedding of one block within another.
type NestedBlock struct {
	// Block is the description of the block that's nested.
	Block

	// Nesting provides the nesting mode for the child block, which determines
	// how many instances of the block are allowed, how many labels it expects,
	// and how the resulting data will be converted into a data structure.
	Nesting NestingMode
}

// NestingMode is an enumeration of modes for nesting blocks inside other
// blocks.
type NestingMode int

//go:generate stringer -type=NestingMode

const (
	nestingModeInvalid NestingMode = iota

	// NestingSingle indicates that only a single instance of a given
	// block type is permitted, with no labels, and its content should be
	// provided directly as an object value.
	NestingSingle

	// NestingList indicates that multiple blocks of the given type are
	// permitted, with no labels, and that their corresponding objects should
	// be provided in a list.
	NestingList

	// NestingSet indicates that multiple blocks of the given type are
	// permitted, with no labels, and that their corresponding objects should
	// be provided in a set.
	NestingSet

	// NestingMap indicates that multiple blocks of the given type are
	// permitted, each with a single label, and that their corresponding
	// objects should be provided in a map whose keys are the labels.
	//
	// It's an error, therefore, to use the same label value on multiple
	// blocks.
	NestingMap
)
