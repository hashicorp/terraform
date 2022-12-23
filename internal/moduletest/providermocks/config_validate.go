package providermocks

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/dynblock"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// Validate compares the mock configuration with the given provider schema,
// which should be the true and complete schema for the provider this
// configuration belongs to, and returns diagnostics describing any
// inconsistencies in how the mock provider has been configured.
//
// [Config.Instantiate] automatically validates compatibility between the
// given provider instance and its reciever, so it's redundant to call
// both functions with equivalent schemas.
func (c *Config) Validate(schema *providers.Schemas) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for rt, rtc := range c.ResourceTypes {
		rts, _ := schema.SchemaForResourceType(rt.Mode, rt.Type)
		if rts == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Mock configuration for unsupported resource type",
				fmt.Sprintf(
					"Mock filename %s declares resource type %s which is not supported by provider %s.",
					filepath.Join(c.BaseDir, rt.MockConfigFilename()), rt, c.ForProvider,
				),
			))
			continue
		}

		for reqType, resps := range rtc.Responses {
			for _, respConfig := range resps {
				diags = diags.Append(
					c.validateResourceTypeResponse(rt, reqType, respConfig, rts),
				)
			}
		}
	}

	return diags
}

func (c *Config) validateResourceTypeResponse(resType ResourceType, reqType requestType, cfg *ResponseConfig, schema *configschema.Block) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// We don't have real values to test with yet, but we can still stub out
	// the top-level symbols and thereby catch any references that can't
	// possibly be correct at runtime.
	probeVars := make(map[string]cty.Value)

	switch reqType {
	case readRequest:
		probeVars["previous_plan_response"] = cty.UnknownVal(responseObjTy)
		probeVars["previous_apply_response"] = cty.UnknownVal(responseObjTy)
		probeVars["previous_state"] = cty.DynamicVal
	case planRequest:
		probeVars["read_response"] = cty.UnknownVal(responseObjTy)
		probeVars["current_state"] = cty.DynamicVal
		probeVars["config"] = cty.DynamicVal
		probeVars["proposed_state"] = cty.DynamicVal
	case applyRequest:
		probeVars["plan_response"] = cty.UnknownVal(responseObjTy)
		probeVars["current_state"] = cty.DynamicVal
		probeVars["planned_state"] = cty.DynamicVal
		probeVars["config"] = cty.DynamicVal
	}

	probeCtx := c.exprEvalContext(probeVars, reqType == planRequest)

	_, hclDiags := cfg.Condition.Value(probeCtx)
	diags = diags.Append(hclDiags)

	contentBody := dynblock.Expand(cfg.Content, probeCtx)
	decSpec := schema.NoneRequired().DecoderSpec()
	contentObj, hclDiags := hcldec.Decode(contentBody, decSpec, probeCtx)
	diags = diags.Append(hclDiags)

	// For any request type other than readRequest a provider is forbidden
	// from setting a configured attribute to anything other than what the
	// author configured. Required arguments must always be configured by
	// definition, and so there's no situation where it makes sense to
	// set them in the content block.
	if reqType != readRequest {
		// FIXME: This is currently testing only the top-level attributes.
		// A real implementation should recursively check nested blocks and
		// structural attributes too.
		for name, attr := range schema.Attributes {
			if attr.Required {
				if v := contentObj.GetAttr(name); !v.IsNull() {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Mock value for required argument",
						Detail:   fmt.Sprintf("Argument %q must always be set in a configuration for %s, so a mock value for it would never be used.", name, resType),
						Subject:  &cfg.DeclRange, // FIXME: This is not a very accurate range for this error
					})
				}
			}
		}
	}

	return diags
}

func (c *Config) exprEvalContext(vars map[string]cty.Value, allowUnknown bool) *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: vars,
		Functions: c.exprFunctions(allowUnknown),
	}
}

func (c *Config) exprFunctions(allowUnknown bool) map[string]function.Function {
	// HACK: Our lang package isn't designed to provide just a disconnected
	// set of functions for use in places other than module source code, so
	// we're going to lie to it here and tell it about a fake module that
	// is just enough to get working functions.
	fakeScope := &lang.Scope{
		BaseDir:     c.BaseDir,
		ConsoleMode: true,
		PureOnly:    false,
	}
	funcs := fakeScope.Functions()
	if allowUnknown {
		funcs["unknown"] = unknownValueFunc
	} else {
		funcs["unknown"] = unknownValueFuncStub
	}
	return funcs
}

var responseObjTy = cty.Object(map[string]cty.Type{
	"name": cty.String,
})

// unknownValueFunc is a mock-provider-specific function that constructs an
// unknown value with a specified type constraint. This is available only
// for when constructing mock plan responses, since all of the other response
// types require all values to be known.
var unknownValueFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Type: typeexpr.TypeConstraintType,
			Name: "type",
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		typeVal := args[0]
		return typeexpr.TypeConstraintFromVal(typeVal), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.UnknownVal(retType), nil
	},
})

// unknownValueFuncStub is a function with the same signature as
// unknownValueFunc but which immediately returns an error when called,
// explaining that the unknown function is available only for plan responses.
var unknownValueFuncStub = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Type: typeexpr.TypeConstraintType,
			Name: "type",
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		return cty.DynamicPseudoType, fmt.Errorf("can only return unknown values in mocked plan responses")
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.UnknownVal(retType), nil
	},
})
