package hclsyntax

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
)

// Expression is the abstract type for nodes that behave as zcl expressions.
type Expression interface {
	Node

	// The hcl.Expression methods are duplicated here, rather than simply
	// embedded, because both Node and hcl.Expression have a Range method
	// and so they conflict.

	Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics)
	Variables() []hcl.Traversal
	StartRange() hcl.Range
}

// Assert that Expression implements hcl.Expression
var assertExprImplExpr hcl.Expression = Expression(nil)

// LiteralValueExpr is an expression that just always returns a given value.
type LiteralValueExpr struct {
	Val      cty.Value
	SrcRange hcl.Range
}

func (e *LiteralValueExpr) walkChildNodes(w internalWalkFunc) {
	// Literal values have no child nodes
}

func (e *LiteralValueExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	return e.Val, nil
}

func (e *LiteralValueExpr) Range() hcl.Range {
	return e.SrcRange
}

func (e *LiteralValueExpr) StartRange() hcl.Range {
	return e.SrcRange
}

// ScopeTraversalExpr is an Expression that retrieves a value from the scope
// using a traversal.
type ScopeTraversalExpr struct {
	Traversal hcl.Traversal
	SrcRange  hcl.Range
}

func (e *ScopeTraversalExpr) walkChildNodes(w internalWalkFunc) {
	// Scope traversals have no child nodes
}

func (e *ScopeTraversalExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	return e.Traversal.TraverseAbs(ctx)
}

func (e *ScopeTraversalExpr) Range() hcl.Range {
	return e.SrcRange
}

func (e *ScopeTraversalExpr) StartRange() hcl.Range {
	return e.SrcRange
}

// RelativeTraversalExpr is an Expression that retrieves a value from another
// value using a _relative_ traversal.
type RelativeTraversalExpr struct {
	Source    Expression
	Traversal hcl.Traversal
	SrcRange  hcl.Range
}

func (e *RelativeTraversalExpr) walkChildNodes(w internalWalkFunc) {
	// Scope traversals have no child nodes
}

func (e *RelativeTraversalExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	src, diags := e.Source.Value(ctx)
	ret, travDiags := e.Traversal.TraverseRel(src)
	diags = append(diags, travDiags...)
	return ret, diags
}

func (e *RelativeTraversalExpr) Range() hcl.Range {
	return e.SrcRange
}

func (e *RelativeTraversalExpr) StartRange() hcl.Range {
	return e.SrcRange
}

// FunctionCallExpr is an Expression that calls a function from the EvalContext
// and returns its result.
type FunctionCallExpr struct {
	Name string
	Args []Expression

	// If true, the final argument should be a tuple, list or set which will
	// expand to be one argument per element.
	ExpandFinal bool

	NameRange       hcl.Range
	OpenParenRange  hcl.Range
	CloseParenRange hcl.Range
}

func (e *FunctionCallExpr) walkChildNodes(w internalWalkFunc) {
	for i, arg := range e.Args {
		e.Args[i] = w(arg).(Expression)
	}
}

func (e *FunctionCallExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	var f function.Function
	exists := false
	hasNonNilMap := false
	thisCtx := ctx
	for thisCtx != nil {
		if thisCtx.Functions == nil {
			thisCtx = thisCtx.Parent()
			continue
		}
		hasNonNilMap = true
		f, exists = thisCtx.Functions[e.Name]
		if exists {
			break
		}
		thisCtx = thisCtx.Parent()
	}

	if !exists {
		if !hasNonNilMap {
			return cty.DynamicVal, hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Function calls not allowed",
					Detail:   "Functions may not be called here.",
					Subject:  e.Range().Ptr(),
				},
			}
		}

		avail := make([]string, 0, len(ctx.Functions))
		for name := range ctx.Functions {
			avail = append(avail, name)
		}
		suggestion := nameSuggestion(e.Name, avail)
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		}

		return cty.DynamicVal, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Call to unknown function",
				Detail:   fmt.Sprintf("There is no function named %q.%s", e.Name, suggestion),
				Subject:  &e.NameRange,
				Context:  e.Range().Ptr(),
			},
		}
	}

	params := f.Params()
	varParam := f.VarParam()

	args := e.Args
	if e.ExpandFinal {
		if len(args) < 1 {
			// should never happen if the parser is behaving
			panic("ExpandFinal set on function call with no arguments")
		}
		expandExpr := args[len(args)-1]
		expandVal, expandDiags := expandExpr.Value(ctx)
		diags = append(diags, expandDiags...)
		if expandDiags.HasErrors() {
			return cty.DynamicVal, diags
		}

		switch {
		case expandVal.Type().IsTupleType() || expandVal.Type().IsListType() || expandVal.Type().IsSetType():
			if expandVal.IsNull() {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid expanding argument value",
					Detail:   "The expanding argument (indicated by ...) must not be null.",
					Context:  expandExpr.Range().Ptr(),
					Subject:  e.Range().Ptr(),
				})
				return cty.DynamicVal, diags
			}
			if !expandVal.IsKnown() {
				return cty.DynamicVal, diags
			}

			newArgs := make([]Expression, 0, (len(args)-1)+expandVal.LengthInt())
			newArgs = append(newArgs, args[:len(args)-1]...)
			it := expandVal.ElementIterator()
			for it.Next() {
				_, val := it.Element()
				newArgs = append(newArgs, &LiteralValueExpr{
					Val:      val,
					SrcRange: expandExpr.Range(),
				})
			}
			args = newArgs
		default:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid expanding argument value",
				Detail:   "The expanding argument (indicated by ...) must be of a tuple, list, or set type.",
				Context:  expandExpr.Range().Ptr(),
				Subject:  e.Range().Ptr(),
			})
			return cty.DynamicVal, diags
		}
	}

	if len(args) < len(params) {
		missing := params[len(args)]
		qual := ""
		if varParam != nil {
			qual = " at least"
		}
		return cty.DynamicVal, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Not enough function arguments",
				Detail: fmt.Sprintf(
					"Function %q expects%s %d argument(s). Missing value for %q.",
					e.Name, qual, len(params), missing.Name,
				),
				Subject: &e.CloseParenRange,
				Context: e.Range().Ptr(),
			},
		}
	}

	if varParam == nil && len(args) > len(params) {
		return cty.DynamicVal, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Too many function arguments",
				Detail: fmt.Sprintf(
					"Function %q expects only %d argument(s).",
					e.Name, len(params),
				),
				Subject: args[len(params)].StartRange().Ptr(),
				Context: e.Range().Ptr(),
			},
		}
	}

	argVals := make([]cty.Value, len(args))

	for i, argExpr := range args {
		var param *function.Parameter
		if i < len(params) {
			param = &params[i]
		} else {
			param = varParam
		}

		val, argDiags := argExpr.Value(ctx)
		if len(argDiags) > 0 {
			diags = append(diags, argDiags...)
		}

		// Try to convert our value to the parameter type
		val, err := convert.Convert(val, param.Type)
		if err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid function argument",
				Detail: fmt.Sprintf(
					"Invalid value for %q parameter: %s.",
					param.Name, err,
				),
				Subject: argExpr.StartRange().Ptr(),
				Context: e.Range().Ptr(),
			})
		}

		argVals[i] = val
	}

	if diags.HasErrors() {
		// Don't try to execute the function if we already have errors with
		// the arguments, because the result will probably be a confusing
		// error message.
		return cty.DynamicVal, diags
	}

	resultVal, err := f.Call(argVals)
	if err != nil {
		switch terr := err.(type) {
		case function.ArgError:
			i := terr.Index
			var param *function.Parameter
			if i < len(params) {
				param = &params[i]
			} else {
				param = varParam
			}
			argExpr := e.Args[i]

			// TODO: we should also unpick a PathError here and show the
			// path to the deep value where the error was detected.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid function argument",
				Detail: fmt.Sprintf(
					"Invalid value for %q parameter: %s.",
					param.Name, err,
				),
				Subject: argExpr.StartRange().Ptr(),
				Context: e.Range().Ptr(),
			})

		default:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Error in function call",
				Detail: fmt.Sprintf(
					"Call to function %q failed: %s.",
					e.Name, err,
				),
				Subject: e.StartRange().Ptr(),
				Context: e.Range().Ptr(),
			})
		}

		return cty.DynamicVal, diags
	}

	return resultVal, diags
}

func (e *FunctionCallExpr) Range() hcl.Range {
	return hcl.RangeBetween(e.NameRange, e.CloseParenRange)
}

func (e *FunctionCallExpr) StartRange() hcl.Range {
	return hcl.RangeBetween(e.NameRange, e.OpenParenRange)
}

type ConditionalExpr struct {
	Condition   Expression
	TrueResult  Expression
	FalseResult Expression

	SrcRange hcl.Range
}

func (e *ConditionalExpr) walkChildNodes(w internalWalkFunc) {
	e.Condition = w(e.Condition).(Expression)
	e.TrueResult = w(e.TrueResult).(Expression)
	e.FalseResult = w(e.FalseResult).(Expression)
}

func (e *ConditionalExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	trueResult, trueDiags := e.TrueResult.Value(ctx)
	falseResult, falseDiags := e.FalseResult.Value(ctx)
	var diags hcl.Diagnostics

	// Try to find a type that both results can be converted to.
	resultType, convs := convert.UnifyUnsafe([]cty.Type{trueResult.Type(), falseResult.Type()})
	if resultType == cty.NilType {
		return cty.DynamicVal, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Inconsistent conditional result types",
				Detail: fmt.Sprintf(
					// FIXME: Need a helper function for showing natural-language type diffs,
					// since this will generate some useless messages in some cases, like
					// "These expressions are object and object respectively" if the
					// object types don't exactly match.
					"The true and false result expressions must have consistent types. The given expressions are %s and %s, respectively.",
					trueResult.Type(), falseResult.Type(),
				),
				Subject: hcl.RangeBetween(e.TrueResult.Range(), e.FalseResult.Range()).Ptr(),
				Context: &e.SrcRange,
			},
		}
	}

	condResult, condDiags := e.Condition.Value(ctx)
	diags = append(diags, condDiags...)
	if condResult.IsNull() {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Null condition",
			Detail:   "The condition value is null. Conditions must either be true or false.",
			Subject:  e.Condition.Range().Ptr(),
			Context:  &e.SrcRange,
		})
		return cty.UnknownVal(resultType), diags
	}
	if !condResult.IsKnown() {
		return cty.UnknownVal(resultType), diags
	}
	condResult, err := convert.Convert(condResult, cty.Bool)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incorrect condition type",
			Detail:   fmt.Sprintf("The condition expression must be of type bool."),
			Subject:  e.Condition.Range().Ptr(),
			Context:  &e.SrcRange,
		})
		return cty.UnknownVal(resultType), diags
	}

	if condResult.True() {
		diags = append(diags, trueDiags...)
		if convs[0] != nil {
			var err error
			trueResult, err = convs[0](trueResult)
			if err != nil {
				// Unsafe conversion failed with the concrete result value
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Inconsistent conditional result types",
					Detail: fmt.Sprintf(
						"The true result value has the wrong type: %s.",
						err.Error(),
					),
					Subject: e.TrueResult.Range().Ptr(),
					Context: &e.SrcRange,
				})
				trueResult = cty.UnknownVal(resultType)
			}
		}
		return trueResult, diags
	} else {
		diags = append(diags, falseDiags...)
		if convs[1] != nil {
			var err error
			falseResult, err = convs[1](falseResult)
			if err != nil {
				// Unsafe conversion failed with the concrete result value
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Inconsistent conditional result types",
					Detail: fmt.Sprintf(
						"The false result value has the wrong type: %s.",
						err.Error(),
					),
					Subject: e.TrueResult.Range().Ptr(),
					Context: &e.SrcRange,
				})
				falseResult = cty.UnknownVal(resultType)
			}
		}
		return falseResult, diags
	}
}

func (e *ConditionalExpr) Range() hcl.Range {
	return e.SrcRange
}

func (e *ConditionalExpr) StartRange() hcl.Range {
	return e.Condition.StartRange()
}

type IndexExpr struct {
	Collection Expression
	Key        Expression

	SrcRange  hcl.Range
	OpenRange hcl.Range
}

func (e *IndexExpr) walkChildNodes(w internalWalkFunc) {
	e.Collection = w(e.Collection).(Expression)
	e.Key = w(e.Key).(Expression)
}

func (e *IndexExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	coll, collDiags := e.Collection.Value(ctx)
	key, keyDiags := e.Key.Value(ctx)
	diags = append(diags, collDiags...)
	diags = append(diags, keyDiags...)

	return hcl.Index(coll, key, &e.SrcRange)
}

func (e *IndexExpr) Range() hcl.Range {
	return e.SrcRange
}

func (e *IndexExpr) StartRange() hcl.Range {
	return e.OpenRange
}

type TupleConsExpr struct {
	Exprs []Expression

	SrcRange  hcl.Range
	OpenRange hcl.Range
}

func (e *TupleConsExpr) walkChildNodes(w internalWalkFunc) {
	for i, expr := range e.Exprs {
		e.Exprs[i] = w(expr).(Expression)
	}
}

func (e *TupleConsExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var vals []cty.Value
	var diags hcl.Diagnostics

	vals = make([]cty.Value, len(e.Exprs))
	for i, expr := range e.Exprs {
		val, valDiags := expr.Value(ctx)
		vals[i] = val
		diags = append(diags, valDiags...)
	}

	return cty.TupleVal(vals), diags
}

func (e *TupleConsExpr) Range() hcl.Range {
	return e.SrcRange
}

func (e *TupleConsExpr) StartRange() hcl.Range {
	return e.OpenRange
}

type ObjectConsExpr struct {
	Items []ObjectConsItem

	SrcRange  hcl.Range
	OpenRange hcl.Range
}

type ObjectConsItem struct {
	KeyExpr   Expression
	ValueExpr Expression
}

func (e *ObjectConsExpr) walkChildNodes(w internalWalkFunc) {
	for i, item := range e.Items {
		e.Items[i].KeyExpr = w(item.KeyExpr).(Expression)
		e.Items[i].ValueExpr = w(item.ValueExpr).(Expression)
	}
}

func (e *ObjectConsExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var vals map[string]cty.Value
	var diags hcl.Diagnostics

	// This will get set to true if we fail to produce any of our keys,
	// either because they are actually unknown or if the evaluation produces
	// errors. In all of these case we must return DynamicPseudoType because
	// we're unable to know the full set of keys our object has, and thus
	// we can't produce a complete value of the intended type.
	//
	// We still evaluate all of the item keys and values to make sure that we
	// get as complete as possible a set of diagnostics.
	known := true

	vals = make(map[string]cty.Value, len(e.Items))
	for _, item := range e.Items {
		key, keyDiags := item.KeyExpr.Value(ctx)
		diags = append(diags, keyDiags...)

		val, valDiags := item.ValueExpr.Value(ctx)
		diags = append(diags, valDiags...)

		if keyDiags.HasErrors() {
			known = false
			continue
		}

		if key.IsNull() {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Null value as key",
				Detail:   "Can't use a null value as a key.",
				Subject:  item.ValueExpr.Range().Ptr(),
			})
			known = false
			continue
		}

		var err error
		key, err = convert.Convert(key, cty.String)
		if err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Incorrect key type",
				Detail:   fmt.Sprintf("Can't use this value as a key: %s.", err.Error()),
				Subject:  item.ValueExpr.Range().Ptr(),
			})
			known = false
			continue
		}

		if !key.IsKnown() {
			known = false
			continue
		}

		keyStr := key.AsString()

		vals[keyStr] = val
	}

	if !known {
		return cty.DynamicVal, diags
	}

	return cty.ObjectVal(vals), diags
}

func (e *ObjectConsExpr) Range() hcl.Range {
	return e.SrcRange
}

func (e *ObjectConsExpr) StartRange() hcl.Range {
	return e.OpenRange
}

// ForExpr represents iteration constructs:
//
//     tuple = [for i, v in list: upper(v) if i > 2]
//     object = {for k, v in map: k => upper(v)}
//     object_of_tuples = {for v in list: v.key: v...}
type ForExpr struct {
	KeyVar string // empty if ignoring the key
	ValVar string

	CollExpr Expression

	KeyExpr  Expression // nil when producing a tuple
	ValExpr  Expression
	CondExpr Expression // null if no "if" clause is present

	Group bool // set if the ellipsis is used on the value in an object for

	SrcRange   hcl.Range
	OpenRange  hcl.Range
	CloseRange hcl.Range
}

func (e *ForExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	collVal, collDiags := e.CollExpr.Value(ctx)
	diags = append(diags, collDiags...)

	if collVal.IsNull() {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Iteration over null value",
			Detail:   "A null value cannot be used as the collection in a 'for' expression.",
			Subject:  e.CollExpr.Range().Ptr(),
			Context:  &e.SrcRange,
		})
		return cty.DynamicVal, diags
	}
	if collVal.Type() == cty.DynamicPseudoType {
		return cty.DynamicVal, diags
	}
	if !collVal.CanIterateElements() {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Iteration over non-iterable value",
			Detail: fmt.Sprintf(
				"A value of type %s cannot be used as the collection in a 'for' expression.",
				collVal.Type().FriendlyName(),
			),
			Subject: e.CollExpr.Range().Ptr(),
			Context: &e.SrcRange,
		})
		return cty.DynamicVal, diags
	}
	if !collVal.IsKnown() {
		return cty.DynamicVal, diags
	}

	childCtx := ctx.NewChild()
	childCtx.Variables = map[string]cty.Value{}

	// Before we start we'll do an early check to see if any CondExpr we've
	// been given is of the wrong type. This isn't 100% reliable (it may
	// be DynamicVal until real values are given) but it should catch some
	// straightforward cases and prevent a barrage of repeated errors.
	if e.CondExpr != nil {
		if e.KeyVar != "" {
			childCtx.Variables[e.KeyVar] = cty.DynamicVal
		}
		childCtx.Variables[e.ValVar] = cty.DynamicVal

		result, condDiags := e.CondExpr.Value(childCtx)
		diags = append(diags, condDiags...)
		if result.IsNull() {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Condition is null",
				Detail:   "The value of the 'if' clause must not be null.",
				Subject:  e.CondExpr.Range().Ptr(),
				Context:  &e.SrcRange,
			})
			return cty.DynamicVal, diags
		}
		_, err := convert.Convert(result, cty.Bool)
		if err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid 'for' condition",
				Detail:   fmt.Sprintf("The 'if' clause value is invalid: %s.", err.Error()),
				Subject:  e.CondExpr.Range().Ptr(),
				Context:  &e.SrcRange,
			})
			return cty.DynamicVal, diags
		}
		if condDiags.HasErrors() {
			return cty.DynamicVal, diags
		}
	}

	if e.KeyExpr != nil {
		// Producing an object
		var vals map[string]cty.Value
		var groupVals map[string][]cty.Value
		if e.Group {
			groupVals = map[string][]cty.Value{}
		} else {
			vals = map[string]cty.Value{}
		}

		it := collVal.ElementIterator()

		known := true
		for it.Next() {
			k, v := it.Element()
			if e.KeyVar != "" {
				childCtx.Variables[e.KeyVar] = k
			}
			childCtx.Variables[e.ValVar] = v

			if e.CondExpr != nil {
				includeRaw, condDiags := e.CondExpr.Value(childCtx)
				diags = append(diags, condDiags...)
				if includeRaw.IsNull() {
					if known {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Condition is null",
							Detail:   "The value of the 'if' clause must not be null.",
							Subject:  e.CondExpr.Range().Ptr(),
							Context:  &e.SrcRange,
						})
					}
					known = false
					continue
				}
				include, err := convert.Convert(includeRaw, cty.Bool)
				if err != nil {
					if known {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid 'for' condition",
							Detail:   fmt.Sprintf("The 'if' clause value is invalid: %s.", err.Error()),
							Subject:  e.CondExpr.Range().Ptr(),
							Context:  &e.SrcRange,
						})
					}
					known = false
					continue
				}
				if !include.IsKnown() {
					known = false
					continue
				}

				if include.False() {
					// Skip this element
					continue
				}
			}

			keyRaw, keyDiags := e.KeyExpr.Value(childCtx)
			diags = append(diags, keyDiags...)
			if keyRaw.IsNull() {
				if known {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid object key",
						Detail:   "Key expression in 'for' expression must not produce a null value.",
						Subject:  e.KeyExpr.Range().Ptr(),
						Context:  &e.SrcRange,
					})
				}
				known = false
				continue
			}
			if !keyRaw.IsKnown() {
				known = false
				continue
			}

			key, err := convert.Convert(keyRaw, cty.String)
			if err != nil {
				if known {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid object key",
						Detail:   fmt.Sprintf("The key expression produced an invalid result: %s.", err.Error()),
						Subject:  e.KeyExpr.Range().Ptr(),
						Context:  &e.SrcRange,
					})
				}
				known = false
				continue
			}

			val, valDiags := e.ValExpr.Value(childCtx)
			diags = append(diags, valDiags...)

			if e.Group {
				k := key.AsString()
				groupVals[k] = append(groupVals[k], val)
			} else {
				k := key.AsString()
				if _, exists := vals[k]; exists {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate object key",
						Detail: fmt.Sprintf(
							"Two different items produced the key %q in this for expression. If duplicates are expected, use the ellipsis (...) after the value expression to enable grouping by key.",
							k,
						),
						Subject: e.KeyExpr.Range().Ptr(),
						Context: &e.SrcRange,
					})
				} else {
					vals[key.AsString()] = val
				}
			}
		}

		if !known {
			return cty.DynamicVal, diags
		}

		if e.Group {
			vals = map[string]cty.Value{}
			for k, gvs := range groupVals {
				vals[k] = cty.TupleVal(gvs)
			}
		}

		return cty.ObjectVal(vals), diags

	} else {
		// Producing a tuple
		vals := []cty.Value{}

		it := collVal.ElementIterator()

		known := true
		for it.Next() {
			k, v := it.Element()
			if e.KeyVar != "" {
				childCtx.Variables[e.KeyVar] = k
			}
			childCtx.Variables[e.ValVar] = v

			if e.CondExpr != nil {
				includeRaw, condDiags := e.CondExpr.Value(childCtx)
				diags = append(diags, condDiags...)
				if includeRaw.IsNull() {
					if known {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Condition is null",
							Detail:   "The value of the 'if' clause must not be null.",
							Subject:  e.CondExpr.Range().Ptr(),
							Context:  &e.SrcRange,
						})
					}
					known = false
					continue
				}
				if !includeRaw.IsKnown() {
					// We will eventually return DynamicVal, but we'll continue
					// iterating in case there are other diagnostics to gather
					// for later elements.
					known = false
					continue
				}

				include, err := convert.Convert(includeRaw, cty.Bool)
				if err != nil {
					if known {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid 'for' condition",
							Detail:   fmt.Sprintf("The 'if' clause value is invalid: %s.", err.Error()),
							Subject:  e.CondExpr.Range().Ptr(),
							Context:  &e.SrcRange,
						})
					}
					known = false
					continue
				}

				if include.False() {
					// Skip this element
					continue
				}
			}

			val, valDiags := e.ValExpr.Value(childCtx)
			diags = append(diags, valDiags...)
			vals = append(vals, val)
		}

		if !known {
			return cty.DynamicVal, diags
		}

		return cty.TupleVal(vals), diags
	}
}

func (e *ForExpr) walkChildNodes(w internalWalkFunc) {
	e.CollExpr = w(e.CollExpr).(Expression)

	scopeNames := map[string]struct{}{}
	if e.KeyVar != "" {
		scopeNames[e.KeyVar] = struct{}{}
	}
	if e.ValVar != "" {
		scopeNames[e.ValVar] = struct{}{}
	}

	if e.KeyExpr != nil {
		w(ChildScope{
			LocalNames: scopeNames,
			Expr:       &e.KeyExpr,
		})
	}
	w(ChildScope{
		LocalNames: scopeNames,
		Expr:       &e.ValExpr,
	})
	if e.CondExpr != nil {
		w(ChildScope{
			LocalNames: scopeNames,
			Expr:       &e.CondExpr,
		})
	}
}

func (e *ForExpr) Range() hcl.Range {
	return e.SrcRange
}

func (e *ForExpr) StartRange() hcl.Range {
	return e.OpenRange
}

type SplatExpr struct {
	Source Expression
	Each   Expression
	Item   *AnonSymbolExpr

	SrcRange    hcl.Range
	MarkerRange hcl.Range
}

func (e *SplatExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	sourceVal, diags := e.Source.Value(ctx)
	if diags.HasErrors() {
		// We'll evaluate our "Each" expression here just to see if it
		// produces any more diagnostics we can report. Since we're not
		// assigning a value to our AnonSymbolExpr here it will return
		// DynamicVal, which should short-circuit any use of it.
		_, itemDiags := e.Item.Value(ctx)
		diags = append(diags, itemDiags...)
		return cty.DynamicVal, diags
	}

	if sourceVal.IsNull() {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Splat of null value",
			Detail:   "Splat expressions (with the * symbol) cannot be applied to null values.",
			Subject:  e.Source.Range().Ptr(),
			Context:  hcl.RangeBetween(e.Source.Range(), e.MarkerRange).Ptr(),
		})
		return cty.DynamicVal, diags
	}
	if !sourceVal.IsKnown() {
		return cty.DynamicVal, diags
	}

	// A "special power" of splat expressions is that they can be applied
	// both to tuples/lists and to other values, and in the latter case
	// the value will be treated as an implicit single-value list. We'll
	// deal with that here first.
	if !(sourceVal.Type().IsTupleType() || sourceVal.Type().IsListType()) {
		sourceVal = cty.ListVal([]cty.Value{sourceVal})
	}

	vals := make([]cty.Value, 0, sourceVal.LengthInt())
	it := sourceVal.ElementIterator()
	if ctx == nil {
		// we need a context to use our AnonSymbolExpr, so we'll just
		// make an empty one here to use as a placeholder.
		ctx = ctx.NewChild()
	}
	isKnown := true
	for it.Next() {
		_, sourceItem := it.Element()
		e.Item.setValue(ctx, sourceItem)
		newItem, itemDiags := e.Each.Value(ctx)
		diags = append(diags, itemDiags...)
		if itemDiags.HasErrors() {
			isKnown = false
		}
		vals = append(vals, newItem)
	}
	e.Item.clearValue(ctx) // clean up our temporary value

	if !isKnown {
		return cty.DynamicVal, diags
	}

	return cty.TupleVal(vals), diags
}

func (e *SplatExpr) walkChildNodes(w internalWalkFunc) {
	e.Source = w(e.Source).(Expression)
	e.Each = w(e.Each).(Expression)
}

func (e *SplatExpr) Range() hcl.Range {
	return e.SrcRange
}

func (e *SplatExpr) StartRange() hcl.Range {
	return e.MarkerRange
}

// AnonSymbolExpr is used as a placeholder for a value in an expression that
// can be applied dynamically to any value at runtime.
//
// This is a rather odd, synthetic expression. It is used as part of the
// representation of splat expressions as a placeholder for the current item
// being visited in the splat evaluation.
//
// AnonSymbolExpr cannot be evaluated in isolation. If its Value is called
// directly then cty.DynamicVal will be returned. Instead, it is evaluated
// in terms of another node (i.e. a splat expression) which temporarily
// assigns it a value.
type AnonSymbolExpr struct {
	SrcRange hcl.Range
	values   map[*hcl.EvalContext]cty.Value
}

func (e *AnonSymbolExpr) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	if ctx == nil {
		return cty.DynamicVal, nil
	}
	val, exists := e.values[ctx]
	if !exists {
		return cty.DynamicVal, nil
	}
	return val, nil
}

// setValue sets a temporary local value for the expression when evaluated
// in the given context, which must be non-nil.
func (e *AnonSymbolExpr) setValue(ctx *hcl.EvalContext, val cty.Value) {
	if e.values == nil {
		e.values = make(map[*hcl.EvalContext]cty.Value)
	}
	if ctx == nil {
		panic("can't setValue for a nil EvalContext")
	}
	e.values[ctx] = val
}

func (e *AnonSymbolExpr) clearValue(ctx *hcl.EvalContext) {
	if e.values == nil {
		return
	}
	if ctx == nil {
		panic("can't clearValue for a nil EvalContext")
	}
	delete(e.values, ctx)
}

func (e *AnonSymbolExpr) walkChildNodes(w internalWalkFunc) {
	// AnonSymbolExpr is a leaf node in the tree
}

func (e *AnonSymbolExpr) Range() hcl.Range {
	return e.SrcRange
}

func (e *AnonSymbolExpr) StartRange() hcl.Range {
	return e.SrcRange
}
