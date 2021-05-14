package templatevals

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/customdecode"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
)

var MakeTemplateFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "source",
			Type: customdecode.ExpressionClosureType,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		// Because our parameter is constrained with the special
		// customdecode.ExpressionClosureType constraint, it's
		// guaranteed to be a known value of a capsule type
		// wrapping customdecode.ExpressionClosure.
		closure := args[0].EncapsulatedValue().(*customdecode.ExpressionClosure)
		expr := closure.Expression
		parentCtx := closure.EvalContext

		// Although in principle this lazy evaluation mechanism could
		// apply to any sort of expression, we intentionally constrain
		// it only to template expressions here to reinforce that this
		// is not intended as a general lambda function mechanism.
		switch expr.(type) {
		case *hclsyntax.TemplateExpr, *hclsyntax.TemplateWrapExpr:
			// ok
		default:
			return cty.DynamicPseudoType, function.NewArgErrorf(0, "must be a string template expression")
		}

		// Our initial template type has arguments derived from the references
		// that have traversals starting with "template".
		atys := make(map[string]cty.Type)
		for _, traversal := range expr.Variables() {
			if traversal.RootName() != "template" {
				// We don't care about any other traversals
				continue
			}
			var step1 hcl.TraverseAttr
			if len(traversal) >= 2 {
				if ta, ok := traversal[1].(hcl.TraverseAttr); ok {
					step1 = ta
				}
			}
			name := step1.Name
			if name == "" { // The conditions above didn't match, then
				return cty.DynamicPseudoType, function.NewArgErrorf(0, "template argument reference at %s must include an attribute lookup representing the argument name", traversal.SourceRange())
			}
			// All of our arguments start off with unconstrained types because
			// we can't walk backwards from an expression to all of the types
			// that could succeed with it. However, the type conversion
			// behavior for template values includes a more specific type check
			// if the destination type has more constrained arguments.
			atys[name] = cty.DynamicPseudoType
		}

		// Before we return we'll check to make sure the expression is
		// evaluable _at all_ (even before we know the argument values)
		// because that'll help users catch totally-invalid templates early,
		// even before they try to pass them to another module to be evaluated.
		ctx := parentCtx.NewChild()
		ctx.Variables = map[string]cty.Value{
			"template": cty.UnknownVal(cty.Object(atys)),
		}
		v, diags := expr.Value(ctx)
		if diags.HasErrors() {
			// It would be nice to have a way to report these diags
			// directly out to HCL, but unfortunately we're sending them
			// out through cty and it doesn't understand HCL diagnostics.
			return cty.DynamicPseudoType, function.NewArgErrorf(0, "invalid template: %s", diags.Error())
		}
		if _, err := convert.Convert(v, cty.String); err != nil {
			// We'll catch this early where possible. It won't always be
			// possible, because the return type might vary depending on
			// the input, so we must re-check this in evaltemplate too.
			return cty.DynamicPseudoType, function.NewArgErrorf(0, "invalid template: must produce a string result")
		}

		// If all of the above was successful then this template seems valid
		// and we can determine which type we're returning. (The actual
		// _value_ of that type will come in Impl.)
		return Type(atys), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		// We already did all of our checking inside Type, so our only remaining
		// job now is to wrap the expression closure up inside a value of
		// our capsule type.
		closure := args[0].EncapsulatedValue().(*customdecode.ExpressionClosure)
		tv := &templateVal{
			expr: closure.Expression,
			ctx:  closure.EvalContext,
		}
		return cty.CapsuleVal(retType, tv), nil
	},
})

// TODO: Consider also a "templatefromfile" function that compiles a separate
// file as a template in a similar way that "templatefile" does, but which
// returns a template value rather than immediately evaluating the template.
// This would then be more convenient for situations where the expected
// template is quite large in itself and thus worth factoring out into a
// separate file.

var EvalTemplateFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "template",
			// We need to type-check this dynamically because there is an
			// infinite number of possible template types.
			Type: cty.DynamicPseudoType,
		},
		{
			Name: "args",
			// We also need to type-check _this_ dynamically, because
			// we expect an object type whose attributes depend on the
			// template type.
			Type:        cty.DynamicPseudoType,
			AllowMarked: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		template := args[0]
		rawArgsObj, retMarks := args[1].Unmark()

		tv := template.EncapsulatedValue().(*templateVal)
		atys := TypeArgs(template.Type())

		// The given arguments object must be compatible with the expected
		// argument types. This'll catch if the call lacks any arguments
		// that the template requires, or if any of them are of an unsuitable
		// type.
		argsObj, err := convert.Convert(rawArgsObj, cty.Object(atys))
		if err != nil {
			return cty.NilVal, function.NewArgError(1, err)
		}

		ctx := tv.ctx.NewChild()
		ctx.Variables = map[string]cty.Value{
			"template": argsObj,
		}
		v, diags := tv.expr.Value(ctx)
		if diags.HasErrors() {
			return cty.NilVal, function.NewArgErrorf(0, "incompatible template: %s", diags.Error())
		}

		v, err = convert.Convert(v, retType)
		if err != nil {
			return cty.NilVal, function.NewArgErrorf(0, "template must produce a string result")
		}

		return v.WithMarks(retMarks), nil
	},
})
