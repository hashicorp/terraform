package configschema

import (
	"github.com/zclconf/go-cty/cty"
)

// ImpliedType returns the cty.Type that would result from decoding a
// configuration block using the receiving block schema.
//
// ImpliedType always returns a result, even if the given schema is
// inconsistent. Code that creates configschema.Block objects should be
// tested using the InternalValidate method to detect any inconsistencies
// that would cause this method to fall back on defaults and assumptions.
func (b *Block) ImpliedType() cty.Type {
	if b == nil {
		return cty.EmptyObject
	}

	// NOTE WELL: This is a hacky experimental implementation of
	// ImpliedType to support an experiment with using attribute
	// syntax exclusively, even for things that the provider protocol
	// currently models as blocks. See the commentary in
	// decoder_spec_block_transform.go for details. This is not intended
	// for inclusion in any real Terraform release.
	//
	// The experiment needs a direct implementation of ImpliedType,
	// rather than just relying on b.DecoderSpec().ImpliedType() as
	// before, because this is relying on an experimental feature
	// of cty that allows object types with some attributes marked
	// as being optional under conversion, and because it's experimental
	// hcldec doesn't currently know about it.

	atys := make(map[string]cty.Type)
	var optionalAttrs []string

	for name, attrS := range b.Attributes {
		atys[name] = attrS.Type
		if !attrS.Required {
			optionalAttrs = append(optionalAttrs, name)
		}
	}

	for name, blockS := range b.BlockTypes {
		objectType := blockS.Block.ImpliedType()

		if blockS.MinItems == 0 {
			optionalAttrs = append(optionalAttrs, name)
		}

		switch blockS.Nesting {
		case NestingSingle, NestingGroup:
			atys[name] = objectType
		case NestingList:
			if objectType.HasDynamicTypes() {
				// We can't properly support this situation within the
				// limitations of the blocks-as-attributes experiment, because
				// it would require some custom transformation logic during
				// decoding to construct a real tuple type. For experiment
				// purposes we'll just pass through the user's value verbatim,
				// without any further processing, and let it fail downstream
				// (in the provider's own validation code) if it's not
				// of a suitable type.
				atys[name] = cty.DynamicPseudoType
			} else {
				atys[name] = cty.List(objectType)
			}
		case NestingSet:
			atys[name] = cty.Set(objectType)
		case NestingMap:
			if blockS.Block.ImpliedType().HasDynamicTypes() {
				// We can't properly support this situation within the
				// limitations of the blocks-as-attributes experiment, because
				// it would require some custom transformation logic during
				// decoding to construct a real tuple type. For experiment
				// purposes we'll just pass through the user's value verbatim,
				// without any further processing, and let it fail downstream
				// (in the provider's own validation code) if it's not
				// of a suitable type.
				atys[name] = cty.DynamicPseudoType
			} else {
				atys[name] = cty.Map(objectType)
			}
		default:
			// Invalid nesting type is just ignored. It's checked by
			// InternalValidate.
			continue
		}
	}

	return cty.ObjectWithOptionalAttrs(atys, optionalAttrs)
}

// ContainsSensitive returns true if any of the attributes of the receiving
// block or any of its descendent blocks are marked as sensitive.
//
// Blocks themselves cannot be sensitive as a whole -- sensitivity is a
// per-attribute idea -- but sometimes we want to include a whole object
// decoded from a block in some UI output, and that is safe to do only if
// none of the contained attributes are sensitive.
func (b *Block) ContainsSensitive() bool {
	for _, attrS := range b.Attributes {
		if attrS.Sensitive {
			return true
		}
	}
	for _, blockS := range b.BlockTypes {
		if blockS.ContainsSensitive() {
			return true
		}
	}
	return false
}
