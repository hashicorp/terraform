package configupgrade

import (
	"log"

	hcl2 "github.com/hashicorp/hcl2/hcl"
	hcl2syntax "github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/tfdiags"
)

// InferExpressionType attempts to determine a result type for the given
// expression source code, which should already have been upgraded to new
// expression syntax.
//
// If self is non-nil, it will determine the meaning of the special "self"
// reference.
//
// If such an inference isn't possible, either because of limitations of
// static analysis or because of errors in the expression, the result is
// cty.DynamicPseudoType indicating "unknown".
func (an *analysis) InferExpressionType(src []byte, self addrs.Referenceable) cty.Type {
	expr, diags := hcl2syntax.ParseExpression(src, "", hcl2.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		// If there's a syntax error then analysis is impossible.
		return cty.DynamicPseudoType
	}

	data := analysisData{an}
	scope := &lang.Scope{
		Data:     data,
		SelfAddr: self,
		PureOnly: false,
		BaseDir:  ".",
	}
	val, _ := scope.EvalExpr(expr, cty.DynamicPseudoType)

	// Value will be cty.DynamicVal if either inference was impossible or
	// if there was an error, leading to cty.DynamicPseudoType here.
	return val.Type()
}

// analysisData is an implementation of lang.Data that returns unknown values
// of suitable types in order to achieve approximate dynamic analysis of
// expression result types, which we need for some upgrade rules.
//
// Unlike a usual implementation of this interface, this one never returns
// errors and will instead just return cty.DynamicVal if it can't produce
// an exact type for any reason. This can then allow partial upgrading to
// proceed and the caller can emit warning comments for ambiguous situations.
//
// N.B.: Source ranges in the data methods are meaningless, since they are
// just relative to the byte array passed to InferExpressionType, not to
// any real input file.
type analysisData struct {
	an *analysis
}

var _ lang.Data = (*analysisData)(nil)

func (d analysisData) StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable) tfdiags.Diagnostics {
	// This implementation doesn't do any static validation.
	return nil
}

func (d analysisData) GetCountAttr(addr addrs.CountAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// All valid count attributes are numbers
	return cty.UnknownVal(cty.Number), nil
}

func (d analysisData) GetResourceInstance(instAddr addrs.ResourceInstance, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	log.Printf("[TRACE] configupgrade: Determining type for %s", instAddr)
	addr := instAddr.Resource

	// Our analysis pass should've found a suitable schema for every resource
	// type in the module.
	providerType, ok := d.an.ResourceProviderType[addr]
	if !ok {
		// Should not be possible, since analysis visits every resource block.
		log.Printf("[TRACE] configupgrade: analysis.GetResourceInstance doesn't have a provider type for %s", addr)
		return cty.DynamicVal, nil
	}
	providerSchema, ok := d.an.ProviderSchemas[providerType]
	if !ok {
		// Should not be possible, since analysis loads schema for every provider.
		log.Printf("[TRACE] configupgrade: analysis.GetResourceInstance doesn't have a provider schema for for %q", providerType)
		return cty.DynamicVal, nil
	}
	schema, _ := providerSchema.SchemaForResourceAddr(addr)
	if schema == nil {
		// Should not be possible, since analysis loads schema for every provider.
		log.Printf("[TRACE] configupgrade: analysis.GetResourceInstance doesn't have a schema for for %s", addr)
		return cty.DynamicVal, nil
	}

	objTy := schema.ImpliedType()

	// We'll emulate the normal evaluator's behavor of deciding whether to
	// return a list or a single object type depending on whether count is
	// set and whether an instance key is given in the address.
	if d.an.ResourceHasCount[addr] {
		if instAddr.Key == addrs.NoKey {
			log.Printf("[TRACE] configupgrade: %s refers to counted instance without a key, so result is a list of %#v", instAddr, objTy)
			return cty.UnknownVal(cty.List(objTy)), nil
		}
		log.Printf("[TRACE] configupgrade: %s refers to counted instance with a key, so result is single object", instAddr)
		return cty.UnknownVal(objTy), nil
	}

	if instAddr.Key != addrs.NoKey {
		log.Printf("[TRACE] configupgrade: %s refers to non-counted instance with a key, which is invalid", instAddr)
		return cty.DynamicVal, nil
	}
	log.Printf("[TRACE] configupgrade: %s refers to non-counted instance without a key, so result is single object", instAddr)
	return cty.UnknownVal(objTy), nil
}

func (d analysisData) GetLocalValue(addrs.LocalValue, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// We can only predict these in general by recursively evaluating their
	// expressions, which creates some undesirable complexity here so for
	// now we'll just skip analyses with locals and see if this complexity
	// is warranted with real-world testing.
	return cty.DynamicVal, nil
}

func (d analysisData) GetModuleInstance(addrs.ModuleCallInstance, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// We only work on one module at a time during upgrading, so we have no
	// information about the outputs of a child module.
	return cty.DynamicVal, nil
}

func (d analysisData) GetModuleInstanceOutput(addrs.ModuleCallOutput, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// We only work on one module at a time during upgrading, so we have no
	// information about the outputs of a child module.
	return cty.DynamicVal, nil
}

func (d analysisData) GetPathAttr(addrs.PathAttr, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// All valid path attributes are strings
	return cty.UnknownVal(cty.String), nil
}

func (d analysisData) GetTerraformAttr(addrs.TerraformAttr, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// All valid "terraform" attributes are strings
	return cty.UnknownVal(cty.String), nil
}

func (d analysisData) GetInputVariable(addr addrs.InputVariable, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// TODO: Collect shallow type information (list vs. map vs. string vs. unknown)
	// in analysis and then return a similarly-approximate type here.
	log.Printf("[TRACE] configupgrade: Determining type for %s", addr)
	name := addr.Name
	typeName := d.an.VariableTypes[name]
	switch typeName {
	case "list":
		return cty.UnknownVal(cty.List(cty.DynamicPseudoType)), nil
	case "map":
		return cty.UnknownVal(cty.Map(cty.DynamicPseudoType)), nil
	case "string":
		return cty.UnknownVal(cty.String), nil
	default:
		return cty.DynamicVal, nil
	}
}
