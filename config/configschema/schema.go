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

	// Sensitive, if set to true, indicates that an attribute may contain
	// sensitive information.
	//
	// At present nothing is done with this information, but callers are
	// encouraged to set it where appropriate so that it may be used in the
	// future to help Terraform mask sensitive information. (Terraform
	// currently achieves this in a limited sense via other mechanisms.)
	Sensitive bool
}

// NestedBlock represents the embedding of one block within another.
type NestedBlock struct {
	// Block is the description of the block that's nested.
	Block

	// Nesting provides the nesting mode for the child block, which determines
	// how many instances of the block are allowed, how many labels it expects,
	// and how the resulting data will be converted into a data structure.
	Nesting NestingMode

	// MinItems and MaxItems set, for the NestingList and NestingSet nesting
	// modes, lower and upper limits on the number of child blocks allowed
	// of the given type. If both are left at zero, no limit is applied.
	//
	// As a special case, both values can be set to 1 for NestingSingle in
	// order to indicate that a particular single block is required.
	//
	// These fields are ignored for other nesting modes and must both be left
	// at zero.
	MinItems, MaxItems int
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
