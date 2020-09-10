package configschema

import (
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
)

//------------------------------------------------------
// THIS IS JUST A TEMPORARY EXPERIMENT AND IS NOT FOR
//     INCLUSION IN A REAL RELEASE OF TERRAFORM
//------------------------------------------------------
//
// The items in this file are in support of a temporary
// hack to make Terraform expect all blocks declared in
// a provider schema to use the HCL attribute syntax
// instead, while keeping all of the other non-syntax
// behaviors.
//
// The purpose of this experiment is twofold:
// - To confirm whether the choice of the HCL syntax type is,
//   as we suspect, independent from all of the other
//   processing differences between attributes and blocks.
// - To allow playing with the ergonomics of existing
//   provider features if they were hypothetically switched
//   over to the attribute syntax, and thus to see if there
//   are any unexpected losses of convenience or functionality
//   that result from using the attribute syntax, assuming
//   that all of the other behavior differences were to
//   remain intact.
//
// If we were to choose to move forward with using the HCL
// attribute syntax exclusively then what's implemented here
// is probably not the best way to implement that change,
// but this is an approach that's relatively easy to build
// within the constraints of the HCL API and provider
// protocol as they currently stand, and thus is hopefully
// a useful proof-of-concept vehicle.

func nestedBlockAsAttrDecoderSpec(typeName string, schema *NestedBlock) hcldec.Spec {
	attrSpec := &hcldec.AttrSpec{
		Name:     typeName,
		Type:     schema.ImpliedType(),
		Required: schema.MinItems > 0,
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
		return &hcldec.TransformFuncSpec{
			Wrapped: attrSpec,
			Func:    hcldecBlockTransformFunction(schema),
		}
	*/
	return attrSpec
}

// hcldecBlockTransformFunction is a cty function that is compatible with
// hcldec.TransformFuncSpec, which takes a value of any type and forces
// it to conform to the given nested block schema, or produces an error
// if that's not possible.
//
// Because this is just a proof-of-concept, the error reporting is not
// of a high standard.
//
// (This turned out to not be necessary for the initial incarnation of the
// experiment, but is retained here for reference in case we want to
// expand it to properly deal with nested blocks containing dynamically-typed
// attributes.)
func hcldecBlockTransformFunction(schema *NestedBlock) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "val",
				Type: cty.DynamicPseudoType,
			},
		},
		Type: func(args []cty.Value) (cty.Type, error) {
			return schema.ImpliedType(), nil
		},
		Impl: func(args []cty.Value, retTy cty.Type) (cty.Value, error) {
			return convert.Convert(args[0], retTy)
		},
	})
}
