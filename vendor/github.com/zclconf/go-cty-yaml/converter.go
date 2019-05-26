package yaml

import (
	"github.com/zclconf/go-cty/cty"
)

// ConverterConfig is used to configure a new converter, using NewConverter.
type ConverterConfig struct {
	// EncodeAsFlow, when set to true, causes Marshal to produce flow-style
	// mapping and sequence serializations.
	EncodeAsFlow bool
}

// A Converter can marshal and unmarshal between cty values and YAML bytes.
//
// Because there are many different ways to map cty to YAML and vice-versa,
// a converter is configurable using the settings in ConverterConfig, which
// allow for a few different permutations of mapping to YAML.
//
// If you are just trying to work with generic, standard YAML, the predefined
// converter in Standard should be good enough.
type Converter struct {
	encodeAsFlow bool
}

// NewConverter creates a new Converter with the given configuration.
func NewConverter(config *ConverterConfig) *Converter {
	return &Converter{
		encodeAsFlow: config.EncodeAsFlow,
	}
}

// Standard is a predefined Converter that produces and consumes generic YAML
// using only built-in constructs that any other YAML implementation ought to
// understand.
var Standard *Converter = NewConverter(&ConverterConfig{})

// ImpliedType analyzes the given source code and returns a suitable type that
// it could be decoded into.
//
// For a converter that is using standard YAML rather than cty-specific custom
// tags, only a subset of cty types can be produced: strings, numbers, bools,
// tuple types, and object types.
func (c *Converter) ImpliedType(src []byte) (cty.Type, error) {
	return c.impliedType(src)
}

// Marshal serializes the given value into a YAML document, using a fixed
// mapping from cty types to YAML constructs.
//
// Note that unlike the function of the same name in the cty JSON package,
// this does not take a type constraint and therefore the YAML serialization
// cannot preserve late-bound type information in the serialization to be
// recovered from Unmarshal. Instead, any cty.DynamicPseudoType in the type
// constraint given to Unmarshal will be decoded as if the corresponding portion
// of the input were processed with ImpliedType to find a target type.
func (c *Converter) Marshal(v cty.Value) ([]byte, error) {
	return c.marshal(v)
}

// Unmarshal reads the document found within the given source buffer
// and attempts to convert it into a value conforming to the given type
// constraint.
//
// An error is returned if the given source contains any YAML document
// delimiters.
func (c *Converter) Unmarshal(src []byte, ty cty.Type) (cty.Value, error) {
	return c.unmarshal(src, ty)
}
