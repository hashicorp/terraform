package hclhil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-zcl/zcl"
	"github.com/hashicorp/hil"
	hilast "github.com/hashicorp/hil/ast"
)

func parseTemplate(src []byte, filename string, startPos zcl.Pos) (zcl.Expression, zcl.Diagnostics) {
	hilStartPos := hilast.Pos{
		Filename: filename,
		Line:     startPos.Line,
		Column:   startPos.Column,
		// HIL positions don't have byte offsets, so we ignore startPos.Byte here
	}
	rootNode, err := hil.ParseWithPosition(string(src), hilStartPos)

	if err != nil {
		return nil, zcl.Diagnostics{
			{
				Severity: zcl.DiagError,
				Summary:  "Syntax error in template",
				Detail:   fmt.Sprintf("The template could not be parsed: %s", err),
				Subject:  errorRange(err),
			},
		}
	}

	return &templateExpression{
		node: rootNode,
	}, nil
}

type templateExpression struct {
	node hilast.Node
}

func (e *templateExpression) Value(ctx *zcl.EvalContext) (cty.Value, zcl.Diagnostics) {
	cfg := hilEvalConfig(ctx)
	return ctyValueFromHILNode(e.node, cfg)
}

func (e *templateExpression) Variables() []zcl.Traversal {
	var vars []zcl.Traversal
	e.node.Accept(func(n hilast.Node) hilast.Node {
		vn, ok := n.(*hilast.VariableAccess)
		if !ok {
			return n
		}

		rawName := vn.Name
		parts := strings.Split(rawName, ".")
		if len(parts) == 0 {
			return n
		}

		tr := make(zcl.Traversal, 0, len(parts))
		tr = append(tr, zcl.TraverseRoot{
			Name:     parts[0],
			SrcRange: rangeFromHILPos(n.Pos()),
		})

		for _, name := range parts {
			if nv, err := strconv.Atoi(name); err == nil {
				// Turn this into a sequence index in zcl land, to save
				// callers from having to understand both HIL-style numeric
				// attributes and zcl-style indices.
				tr = append(tr, zcl.TraverseIndex{
					Key:      cty.NumberIntVal(int64(nv)),
					SrcRange: rangeFromHILPos(n.Pos()),
				})
				continue
			}

			if name == "*" {
				// TODO: support splat traversals, but that requires some
				// more work here because we need to then accumulate the
				// rest of the parts into the splat's own "Each" traversal.
				continue
			}

			tr = append(tr, zcl.TraverseAttr{
				Name:     name,
				SrcRange: rangeFromHILPos(n.Pos()),
			})
		}

		vars = append(vars, tr)

		return n
	})
	return vars
}

func (e *templateExpression) Range() zcl.Range {
	return rangeFromHILPos(e.node.Pos())
}
func (e *templateExpression) StartRange() zcl.Range {
	return rangeFromHILPos(e.node.Pos())
}

func hilEvalConfig(ctx *zcl.EvalContext) *hil.EvalConfig {
	cfg := &hil.EvalConfig{
		GlobalScope: &hilast.BasicScope{
			VarMap:  map[string]hilast.Variable{},
			FuncMap: map[string]hilast.Function{},
		},
	}

	if ctx == nil {
		return cfg
	}

	if ctx.Variables != nil {
		for name, val := range ctx.Variables {
			cfg.GlobalScope.VarMap[name] = hilVariableForInput(hilVariableFromCtyValue(val))
		}
	}

	if ctx.Functions != nil {
		for name, hf := range ctx.Functions {
			cfg.GlobalScope.FuncMap[name] = hilFunctionFromCtyFunction(hf)
		}
	}

	return cfg
}

func ctyValueFromHILNode(node hilast.Node, cfg *hil.EvalConfig) (cty.Value, zcl.Diagnostics) {
	result, err := hil.Eval(node, cfg)
	if err != nil {
		return cty.DynamicVal, zcl.Diagnostics{
			{
				Severity: zcl.DiagError,
				Summary:  "Template evaluation failed",
				Detail:   fmt.Sprintf("Error while evaluating template: %s", err),
				Subject:  rangeFromHILPos(node.Pos()).Ptr(),
			},
		}
	}

	return ctyValueFromHILResult(result), nil
}

func ctyValueFromHILResult(result hil.EvaluationResult) cty.Value {
	switch result.Type {
	case hil.TypeString:
		return cty.StringVal(result.Value.(string))
	case hil.TypeBool:
		return cty.BoolVal(result.Value.(bool))
	case hil.TypeList:
		varsI := result.Value.([]interface{})
		if len(varsI) == 0 {
			return cty.ListValEmpty(cty.String)
		}
		vals := make([]cty.Value, len(varsI))
		for i, varI := range varsI {
			hv, err := hil.InterfaceToVariable(varI)
			if err != nil {
				panic("HIL returned type that can't be converted back to variable")
			}
			vals[i] = ctyValueFromHILVariable(hv)
		}
		return cty.TupleVal(vals)
	case hil.TypeMap:
		varsI := result.Value.(map[string]interface{})
		if len(varsI) == 0 {
			return cty.MapValEmpty(cty.String)
		}
		vals := make(map[string]cty.Value)
		for key, varI := range varsI {
			hv, err := hil.InterfaceToVariable(varI)
			if err != nil {
				panic("HIL returned type that can't be converted back to variable")
			}
			vals[key] = ctyValueFromHILVariable(hv)
		}
		return cty.ObjectVal(vals)
	case hil.TypeUnknown:
		// HIL doesn't have typed unknowns, so we have to return dynamic
		return cty.DynamicVal
	default:
		// should never happen
		panic(fmt.Sprintf("unsupported EvaluationResult type %s", result.Type))
	}

}

func ctyValueFromHILVariable(vr hilast.Variable) cty.Value {
	switch vr.Type {
	case hilast.TypeBool:
		return cty.BoolVal(vr.Value.(bool))
	case hilast.TypeString:
		return cty.StringVal(vr.Value.(string))
	case hilast.TypeInt:
		return cty.NumberIntVal(vr.Value.(int64))
	case hilast.TypeFloat:
		return cty.NumberFloatVal(vr.Value.(float64))
	case hilast.TypeList:
		vars := vr.Value.([]hilast.Variable)
		if len(vars) == 0 {
			return cty.ListValEmpty(cty.String)
		}
		vals := make([]cty.Value, len(vars))
		for i, v := range vars {
			vals[i] = ctyValueFromHILVariable(v)
		}
		return cty.TupleVal(vals)
	case hilast.TypeMap:
		vars := vr.Value.(map[string]hilast.Variable)
		if len(vars) == 0 {
			return cty.MapValEmpty(cty.String)
		}
		vals := make(map[string]cty.Value)
		for key, v := range vars {
			vals[key] = ctyValueFromHILVariable(v)
		}
		return cty.ObjectVal(vals)
	case hilast.TypeAny, hilast.TypeUnknown:
		return cty.DynamicVal
	default:
		// should never happen
		panic(fmt.Sprintf("unsupported HIL Variable type %s", vr.Type))
	}
}

func hilVariableFromCtyValue(val cty.Value) hilast.Variable {
	if !val.IsKnown() {
		return hilast.Variable{
			Type:  hilast.TypeUnknown,
			Value: hil.UnknownValue,
		}
	}
	if val.IsNull() {
		// HIL doesn't actually support nulls, so we'll cheat a bit and
		// use an unknown. This is not quite right since nulls are supposed
		// to fail when evaluated, but it should suffice as a compatibility
		// shim since HIL-using applications probably won't be generating
		// nulls anyway.
		return hilast.Variable{
			Type:  hilast.TypeUnknown,
			Value: hil.UnknownValue,
		}
	}

	ty := val.Type()
	switch ty {
	case cty.String:
		return hilast.Variable{
			Type:  hilast.TypeString,
			Value: val.AsString(),
		}
	case cty.Number:
		// cty doesn't distinguish between floats and ints, so we'll
		// just always use floats here and depend on automatic conversions
		// to produce ints where needed.
		bf := val.AsBigFloat()
		f, _ := bf.Float64()
		return hilast.Variable{
			Type:  hilast.TypeFloat,
			Value: f,
		}
	case cty.Bool:
		return hilast.Variable{
			Type:  hilast.TypeBool,
			Value: val.True(),
		}
	}

	switch {
	case ty.IsListType() || ty.IsSetType() || ty.IsTupleType():
		// HIL doesn't have sets, so we'll just turn them into lists
		// HIL doesn't support tuples either, so any tuples without consistent
		// element types will fail HIL's check for consistent types, but that's
		// okay since we don't intend to change HIL semantics here.
		vars := []hilast.Variable{}
		it := val.ElementIterator()
		for it.Next() {
			_, ev := it.Element()
			vars = append(vars, hilVariableFromCtyValue(ev))
		}
		return hilast.Variable{
			Type:  hilast.TypeList,
			Value: vars,
		}
	case ty.IsMapType():
		vars := map[string]hilast.Variable{}
		it := val.ElementIterator()
		for it.Next() {
			kv, ev := it.Element()
			k := kv.AsString()
			vars[k] = hilVariableFromCtyValue(ev)
		}
		return hilast.Variable{
			Type:  hilast.TypeMap,
			Value: vars,
		}
	case ty.IsObjectType():
		// HIL doesn't support objects, so objects that don't have consistent
		// attribute types will fail HIL's check for consistent types. That's
		// okay since we don't intend to change HIL semantics here.
		vars := map[string]interface{}{}
		atys := ty.AttributeTypes()
		for k := range atys {
			vars[k] = hilVariableFromCtyValue(val.GetAttr(k))
		}
		return hilast.Variable{
			Type:  hilast.TypeMap,
			Value: vars,
		}
	case ty.IsCapsuleType():
		// Can't do anything reasonable with capsule types, so we'll just
		// treat them as unknown and let the caller deal with it as an error.
		return hilast.Variable{
			Type:  hilast.TypeUnknown,
			Value: hil.UnknownValue,
		}
	default:
		// Should never happen if we've done our job right here
		panic(fmt.Sprintf("don't know how to convert %#v into a HIL variable", ty))
	}

}

// hilVariableForInput constrains the given variable to be of the types HIL
// accepts as input, which entails converting all primitive types to string.
func hilVariableForInput(v hilast.Variable) hilast.Variable {
	switch v.Type {
	case hilast.TypeFloat:
		return hilast.Variable{
			Type:  hilast.TypeString,
			Value: strconv.FormatFloat(v.Value.(float64), 'f', -1, 64),
		}
	case hilast.TypeBool:
		if v.Value.(bool) {
			return hilast.Variable{
				Type:  hilast.TypeString,
				Value: "true",
			}
		} else {
			return hilast.Variable{
				Type:  hilast.TypeString,
				Value: "false",
			}
		}
	case hilast.TypeList:
		inVars := v.Value.([]hilast.Variable)
		outVars := make([]hilast.Variable, len(inVars))
		for i, inVar := range inVars {
			outVars[i] = hilVariableForInput(inVar)
		}
		return hilast.Variable{
			Type:  hilast.TypeList,
			Value: outVars,
		}
	case hilast.TypeMap:
		inVars := v.Value.(map[string]hilast.Variable)
		outVars := make(map[string]hilast.Variable)
		for k, inVar := range inVars {
			outVars[k] = hilVariableForInput(inVar)
		}
		return hilast.Variable{
			Type:  hilast.TypeMap,
			Value: outVars,
		}
	default:
		return v
	}
}

func hilTypeFromCtyType(ty cty.Type) hilast.Type {
	switch ty {
	case cty.String:
		return hilast.TypeString
	case cty.Number:
		return hilast.TypeFloat
	case cty.Bool:
		return hilast.TypeBool
	case cty.DynamicPseudoType:
		// Assume we're using this as a type specification, so we'd rather
		// have TypeAny than TypeUnknown.
		return hilast.TypeAny
	}

	switch {
	case ty.IsListType() || ty.IsSetType() || ty.IsTupleType():
		return hilast.TypeList
	case ty.IsMapType(), ty.IsObjectType():
		return hilast.TypeMap
	default:
		return hilast.TypeUnknown
	}

}

func hilFunctionFromCtyFunction(f function.Function) hilast.Function {
	hf := hilast.Function{}
	params := f.Params()
	varParam := f.VarParam()

	hf.ArgTypes = make([]hilast.Type, len(params))
	staticTypes := make([]cty.Type, len(params))
	for i, param := range params {
		hf.ArgTypes[i] = hilTypeFromCtyType(param.Type)
		staticTypes[i] = param.Type
	}
	if varParam != nil {
		hf.Variadic = true
		hf.VariadicType = hilTypeFromCtyType(varParam.Type)
	}

	retType, err := f.ReturnType(staticTypes)
	if err == nil {
		hf.ReturnType = hilTypeFromCtyType(retType)
	} else {
		hf.ReturnType = hilTypeFromCtyType(cty.DynamicPseudoType)
	}

	hf.Callback = func(hilArgs []interface{}) (interface{}, error) {
		args := make([]cty.Value, len(hilArgs))
		for i, hilArg := range hilArgs {
			var hilType hilast.Type
			if i < len(hf.ArgTypes) {
				hilType = hf.ArgTypes[i]
			} else {
				hilType = hf.VariadicType
			}
			args[i] = ctyValueFromHILVariable(hilast.Variable{
				Type:  hilType,
				Value: hilArg,
			})
		}

		result, err := f.Call(args)
		if err != nil {
			return nil, err
		}

		hilResult := hilVariableFromCtyValue(result)
		return hilResult.Value, nil
	}

	return hf
}
