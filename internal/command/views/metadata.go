package views

import (
	"encoding/json"

	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

type Metadata interface {
	Functions(functions map[string]function.Function) tfdiags.Diagnostics
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewMetadata(view *View) Metadata {
	return &MetadataHuman{view: view}
}

type MetadataHuman struct {
	view *View
}

var _ Metadata = (*MetadataHuman)(nil)

func (v *MetadataHuman) Functions(functions map[string]function.Function) tfdiags.Diagnostics {
	// Parameter represents a parameter to a function.
	type Parameter struct {
		// Name is an optional name for the argument.
		Name string `json:"name,omitempty"`

		// A type that any argument for this parameter must conform to.
		Type json.RawMessage `json:"type"`
	}

	// Function represents a callable function.
	type Function struct {
		// Name is the identifier used to call the function
		Name string `json:"name"`

		// Type is the ctyjson representation of the function's return type,
		// based on supplying all required parameters using the expected types.
		// Functions can have dynamic return types.
		Type json.RawMessage `json:"type"`

		// Parameters describes the function's fixed positional parameters.
		Parameters []*Parameter `json:"parameters"`

		// VariadicParameter describes the function's variadic parameters, if
		// any are supported.
		VariadicParameter *Parameter `json:"variadic_parameter,omitempty"`
	}

	result := make(map[string]*Function, len(functions))
	for name, fn := range functions {
		fnParams := fn.Params()
		argTypes := make([]cty.Type, 0, len(fnParams))
		params := make([]*Parameter, 0, len(fnParams))

		for _, param := range fnParams {
			argTypes = append(argTypes, param.Type)

			params = append(params, &Parameter{
				Name: param.Name,
				Type: mustJsonType(param.Type),
			})
		}

		fnType, err := fn.ReturnType(argTypes)
		if err != nil {
			// We intentionally ignore errors when determining return type, for
			// several reasons:
			//
			// - Some functions (e.g. `coalesce`) take only variadic arguments
			//   of dynamic types, so there is nothing useful we can say about
			//   the return type without concrete arguments;
			// - Deprecated functions return errors when determining their
			//   type, so again there is nothing useful to say.
			fnType = cty.DynamicPseudoType
		}

		var varParam *Parameter
		fnVarParam := fn.VarParam()
		if fnVarParam != nil {
			varParam = &Parameter{
				Name: fnVarParam.Name,
				Type: mustJsonType(fnVarParam.Type),
			}
		}

		function := &Function{
			Name:              name,
			Type:              mustJsonType(fnType),
			Parameters:        params,
			VariadicParameter: varParam,
		}
		result[name] = function
	}

	var diags tfdiags.Diagnostics

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		diags = diags.Append(err)
		return diags
	}

	v.view.streams.Println(string(jsonResult))

	return diags
}

func (v *MetadataHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func mustJsonType(ty cty.Type) json.RawMessage {
	jsonType, err := ctyjson.MarshalType(ty)
	// If we cannot express the type in JSON, the only useful thing we can say
	// is that the type is dynamic. Examples of this are capsule types for
	// functions like `try`.
	if err != nil {
		jsonType, _ = ctyjson.MarshalType(cty.DynamicPseudoType)
	}
	return json.RawMessage(jsonType)
}
