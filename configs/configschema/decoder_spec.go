package configschema

import (
	"github.com/hashicorp/hcl/v2/hcldec"
)

var mapLabelNames = []string{"key"}

// DecoderSpec returns a hcldec.Spec that can be used to decode a HCL Body
// using the facilities in the hcldec package.
//
// The returned specification is guaranteed to return a value of the same type
// returned by method ImpliedType, but it may contain null values if any of the
// block attributes are defined as optional and/or computed respectively.
func (b *Block) DecoderSpec() hcldec.Spec {
	ret := hcldec.ObjectSpec{}
	if b == nil {
		return ret
	}

	objTy := b.ImpliedType()
	for name, aty := range objTy.AttributeTypes() {
		ret[name] = &hcldec.AttrSpec{
			Name:     name,
			Type:     aty,
			Required: !objTy.AttributeOptional(name),
		}

		// A custom transform function would be necessary to properly handle
		// nested blocks with dynamically-typed attributes inside, because
		// we need to choose a suitable concrete tuple or map type after
		// initial conversion, but we're not actually doing that in the
		// current incarnation of the experiment so such blocks will just
		// have their values passed through totally unvalidated and unconverted,
		// where invalid values will likely end up either caught by the
		// provider's validation logic or making the provider crash somehow.
		//
		// This is an experiment after all, so it would be strange if there
		// were not some risk of it crashing in strange ways.
		/*
			ret[name] = &hcldec.TransformFuncSpec{
				Wrapped: attrSpec,
				Func:    hcldecBlockTransformFunction(schema),
			}
		*/

	}

	return ret
}

func (a *Attribute) decoderSpec(name string) hcldec.Spec {
	return &hcldec.AttrSpec{
		Name:     name,
		Type:     a.Type,
		Required: a.Required,
	}
}
