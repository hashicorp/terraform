package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/typeexpr"
)

// A consistent detail message for all "not a valid identifier" diagnostics.
const badIdentifierDetail = "A name must start with a letter or underscore and may contain only letters, digits, underscores, and dashes."

// Variable represents a "variable" block in a module or file.
type Variable struct {
	Name        string
	Description string
	Default     cty.Value

	// Type is the concrete type of the variable value.
	Type cty.Type
	// ConstraintType is used for decoding and type conversions, and may
	// contain nested ObjectWithOptionalAttr types.
	ConstraintType cty.Type

	ParsingMode VariableParsingMode
	Validations []*CheckRule
	Sensitive   bool

	DescriptionSet bool
	SensitiveSet   bool

	// Nullable indicates that null is a valid value for this variable. Setting
	// Nullable to false means that the module can expect this variable to
	// never be null.
	Nullable    bool
	NullableSet bool

	DeclRange hcl.Range
}

func decodeVariableBlock(block *hcl.Block, override bool) (*Variable, hcl.Diagnostics) {
	v := &Variable{
		Name:      block.Labels[0],
		DeclRange: block.DefRange,
	}

	// Unless we're building an override, we'll set some defaults
	// which we might override with attributes below. We leave these
	// as zero-value in the override case so we can recognize whether
	// or not they are set when we merge.
	if !override {
		v.Type = cty.DynamicPseudoType
		v.ConstraintType = cty.DynamicPseudoType
		v.ParsingMode = VariableParseLiteral
	}

	content, diags := block.Body.Content(variableBlockSchema)

	if !hclsyntax.ValidIdentifier(v.Name) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid variable name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[0],
		})
	}

	// Don't allow declaration of variables that would conflict with the
	// reserved attribute and block type names in a "module" block, since
	// these won't be usable for child modules.
	for _, attr := range moduleBlockSchema.Attributes {
		if attr.Name == v.Name {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid variable name",
				Detail:   fmt.Sprintf("The variable name %q is reserved due to its special meaning inside module blocks.", attr.Name),
				Subject:  &block.LabelRanges[0],
			})
		}
	}
	for _, blockS := range moduleBlockSchema.Blocks {
		if blockS.Type == v.Name {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid variable name",
				Detail:   fmt.Sprintf("The variable name %q is reserved due to its special meaning inside module blocks.", blockS.Type),
				Subject:  &block.LabelRanges[0],
			})
		}
	}

	if attr, exists := content.Attributes["description"]; exists {
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &v.Description)
		diags = append(diags, valDiags...)
		v.DescriptionSet = true
	}

	if attr, exists := content.Attributes["type"]; exists {
		ty, parseMode, tyDiags := decodeVariableType(attr.Expr)
		diags = append(diags, tyDiags...)
		v.ConstraintType = ty
		v.Type = ty.WithoutOptionalAttributesDeep()
		v.ParsingMode = parseMode
	}

	if attr, exists := content.Attributes["sensitive"]; exists {
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &v.Sensitive)
		diags = append(diags, valDiags...)
		v.SensitiveSet = true
	}

	if attr, exists := content.Attributes["nullable"]; exists {
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &v.Nullable)
		diags = append(diags, valDiags...)
		v.NullableSet = true
	} else {
		// The current default is true, which is subject to change in a future
		// language edition.
		v.Nullable = true
	}

	if attr, exists := content.Attributes["default"]; exists {
		val, valDiags := attr.Expr.Value(nil)
		diags = append(diags, valDiags...)

		// Convert the default to the expected type so we can catch invalid
		// defaults early and allow later code to assume validity.
		// Note that this depends on us having already processed any "type"
		// attribute above.
		// However, we can't do this if we're in an override file where
		// the type might not be set; we'll catch that during merge.
		if v.ConstraintType != cty.NilType {
			var err error
			val, err = convert.Convert(val, v.ConstraintType)
			if err != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid default value for variable",
					Detail:   fmt.Sprintf("This default value is not compatible with the variable's type constraint: %s.", err),
					Subject:  attr.Expr.Range().Ptr(),
				})
				val = cty.DynamicVal
			}
		}

		if !v.Nullable && val.IsNull() {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid default value for variable",
				Detail:   "A null default value is not valid when nullable=false.",
				Subject:  attr.Expr.Range().Ptr(),
			})
		}

		v.Default = val
	}

	for _, block := range content.Blocks {
		switch block.Type {

		case "validation":
			vv, moreDiags := decodeVariableValidationBlock(v.Name, block, override)
			diags = append(diags, moreDiags...)
			v.Validations = append(v.Validations, vv)

		default:
			// The above cases should be exhaustive for all block types
			// defined in variableBlockSchema
			panic(fmt.Sprintf("unhandled block type %q", block.Type))
		}
	}

	return v, diags
}

func decodeVariableType(expr hcl.Expression) (cty.Type, VariableParsingMode, hcl.Diagnostics) {
	if exprIsNativeQuotedString(expr) {
		// If a user provides the pre-0.12 form of variable type argument where
		// the string values "string", "list" and "map" are accepted, we
		// provide an error to point the user towards using the type system
		// correctly has a hint.
		// Only the native syntax ends up in this codepath; we handle the
		// JSON syntax (which is, of course, quoted within the type system)
		// in the normal codepath below.
		val, diags := expr.Value(nil)
		if diags.HasErrors() {
			return cty.DynamicPseudoType, VariableParseHCL, diags
		}
		str := val.AsString()
		switch str {
		case "string":
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid quoted type constraints",
				Detail:   "Terraform 0.11 and earlier required type constraints to be given in quotes, but that form is now deprecated and will be removed in a future version of Terraform. Remove the quotes around \"string\".",
				Subject:  expr.Range().Ptr(),
			})
			return cty.DynamicPseudoType, VariableParseLiteral, diags
		case "list":
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid quoted type constraints",
				Detail:   "Terraform 0.11 and earlier required type constraints to be given in quotes, but that form is now deprecated and will be removed in a future version of Terraform. Remove the quotes around \"list\" and write list(string) instead to explicitly indicate that the list elements are strings.",
				Subject:  expr.Range().Ptr(),
			})
			return cty.DynamicPseudoType, VariableParseHCL, diags
		case "map":
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid quoted type constraints",
				Detail:   "Terraform 0.11 and earlier required type constraints to be given in quotes, but that form is now deprecated and will be removed in a future version of Terraform. Remove the quotes around \"map\" and write map(string) instead to explicitly indicate that the map elements are strings.",
				Subject:  expr.Range().Ptr(),
			})
			return cty.DynamicPseudoType, VariableParseHCL, diags
		default:
			return cty.DynamicPseudoType, VariableParseHCL, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Invalid legacy variable type hint",
				Detail:   `To provide a full type expression, remove the surrounding quotes and give the type expression directly.`,
				Subject:  expr.Range().Ptr(),
			}}
		}
	}

	// First we'll deal with some shorthand forms that the HCL-level type
	// expression parser doesn't include. These both emulate pre-0.12 behavior
	// of allowing a list or map of any element type as long as all of the
	// elements are consistent. This is the same as list(any) or map(any).
	switch hcl.ExprAsKeyword(expr) {
	case "list":
		return cty.List(cty.DynamicPseudoType), VariableParseHCL, nil
	case "map":
		return cty.Map(cty.DynamicPseudoType), VariableParseHCL, nil
	}

	ty, diags := typeexpr.TypeConstraint(expr)
	if diags.HasErrors() {
		return cty.DynamicPseudoType, VariableParseHCL, diags
	}

	switch {
	case ty.IsPrimitiveType():
		// Primitive types use literal parsing.
		return ty, VariableParseLiteral, diags
	default:
		// Everything else uses HCL parsing
		return ty, VariableParseHCL, diags
	}
}

// Required returns true if this variable is required to be set by the caller,
// or false if there is a default value that will be used when it isn't set.
func (v *Variable) Required() bool {
	return v.Default == cty.NilVal
}

// VariableParsingMode defines how values of a particular variable given by
// text-only mechanisms (command line arguments and environment variables)
// should be parsed to produce the final value.
type VariableParsingMode rune

// VariableParseLiteral is a variable parsing mode that just takes the given
// string directly as a cty.String value.
const VariableParseLiteral VariableParsingMode = 'L'

// VariableParseHCL is a variable parsing mode that attempts to parse the given
// string as an HCL expression and returns the result.
const VariableParseHCL VariableParsingMode = 'H'

// Parse uses the receiving parsing mode to process the given variable value
// string, returning the result along with any diagnostics.
//
// A VariableParsingMode does not know the expected type of the corresponding
// variable, so it's the caller's responsibility to attempt to convert the
// result to the appropriate type and return to the user any diagnostics that
// conversion may produce.
//
// The given name is used to create a synthetic filename in case any diagnostics
// must be generated about the given string value. This should be the name
// of the root module variable whose value will be populated from the given
// string.
//
// If the returned diagnostics has errors, the returned value may not be
// valid.
func (m VariableParsingMode) Parse(name, value string) (cty.Value, hcl.Diagnostics) {
	switch m {
	case VariableParseLiteral:
		return cty.StringVal(value), nil
	case VariableParseHCL:
		fakeFilename := fmt.Sprintf("<value for var.%s>", name)
		expr, diags := hclsyntax.ParseExpression([]byte(value), fakeFilename, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			return cty.DynamicVal, diags
		}
		val, valDiags := expr.Value(nil)
		diags = append(diags, valDiags...)
		return val, diags
	default:
		// Should never happen
		panic(fmt.Errorf("Parse called on invalid VariableParsingMode %#v", m))
	}
}

// decodeVariableValidationBlock is a wrapper around decodeCheckRuleBlock
// that imposes the additional rule that the condition expression can refer
// only to an input variable of the given name.
func decodeVariableValidationBlock(varName string, block *hcl.Block, override bool) (*CheckRule, hcl.Diagnostics) {
	vv, diags := decodeCheckRuleBlock(block, override)
	if vv.Condition != nil {
		// The validation condition can only refer to the variable itself,
		// to ensure that the variable declaration can't create additional
		// edges in the dependency graph.
		goodRefs := 0
		for _, traversal := range vv.Condition.Variables() {
			ref, moreDiags := addrs.ParseRef(traversal)
			if !moreDiags.HasErrors() {
				if addr, ok := ref.Subject.(addrs.InputVariable); ok {
					if addr.Name == varName {
						goodRefs++
						continue // Reference is valid
					}
				}
			}
			// If we fall out here then the reference is invalid.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference in variable validation",
				Detail:   fmt.Sprintf("The condition for variable %q can only refer to the variable itself, using var.%s.", varName, varName),
				Subject:  traversal.SourceRange().Ptr(),
			})
		}
		if goodRefs < 1 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid variable validation condition",
				Detail:   fmt.Sprintf("The condition for variable %q must refer to var.%s in order to test incoming values.", varName, varName),
				Subject:  vv.Condition.Range().Ptr(),
			})
		}
	}

	if vv.ErrorMessage != nil {
		// The same applies to the validation error message, except that
		// references are not required. A string literal is a valid error
		// message.
		goodRefs := 0
		for _, traversal := range vv.ErrorMessage.Variables() {
			ref, moreDiags := addrs.ParseRef(traversal)
			if !moreDiags.HasErrors() {
				if addr, ok := ref.Subject.(addrs.InputVariable); ok {
					if addr.Name == varName {
						goodRefs++
						continue // Reference is valid
					}
				}
			}
			// If we fall out here then the reference is invalid.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference in variable validation",
				Detail:   fmt.Sprintf("The error message for variable %q can only refer to the variable itself, using var.%s.", varName, varName),
				Subject:  traversal.SourceRange().Ptr(),
			})
		}
	}

	return vv, diags
}

// Output represents an "output" block in a module or file.
type Output struct {
	Name        string
	Description string
	Expr        hcl.Expression
	DependsOn   []hcl.Traversal
	Sensitive   bool

	Preconditions []*CheckRule

	DescriptionSet bool
	SensitiveSet   bool

	DeclRange hcl.Range
}

func decodeOutputBlock(block *hcl.Block, override bool) (*Output, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	o := &Output{
		Name:      block.Labels[0],
		DeclRange: block.DefRange,
	}

	schema := outputBlockSchema
	if override {
		schema = schemaForOverrides(schema)
	}

	content, moreDiags := block.Body.Content(schema)
	diags = append(diags, moreDiags...)

	if !hclsyntax.ValidIdentifier(o.Name) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid output name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[0],
		})
	}

	if attr, exists := content.Attributes["description"]; exists {
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &o.Description)
		diags = append(diags, valDiags...)
		o.DescriptionSet = true
	}

	if attr, exists := content.Attributes["value"]; exists {
		o.Expr = attr.Expr
	}

	if attr, exists := content.Attributes["sensitive"]; exists {
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &o.Sensitive)
		diags = append(diags, valDiags...)
		o.SensitiveSet = true
	}

	if attr, exists := content.Attributes["depends_on"]; exists {
		deps, depsDiags := decodeDependsOn(attr)
		diags = append(diags, depsDiags...)
		o.DependsOn = append(o.DependsOn, deps...)
	}

	for _, block := range content.Blocks {
		switch block.Type {
		case "precondition":
			cr, moreDiags := decodeCheckRuleBlock(block, override)
			diags = append(diags, moreDiags...)
			o.Preconditions = append(o.Preconditions, cr)
		case "postcondition":
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Postconditions are not allowed",
				Detail:   "Output values can only have preconditions, not postconditions.",
				Subject:  block.TypeRange.Ptr(),
			})
		default:
			// The cases above should be exhaustive for all block types
			// defined in the block type schema, so this shouldn't happen.
			panic(fmt.Sprintf("unexpected lifecycle sub-block type %q", block.Type))
		}
	}

	return o, diags
}

// Local represents a single entry from a "locals" block in a module or file.
// The "locals" block itself is not represented, because it serves only to
// provide context for us to interpret its contents.
type Local struct {
	Name string
	Expr hcl.Expression

	DeclRange hcl.Range
}

func decodeLocalsBlock(block *hcl.Block) ([]*Local, hcl.Diagnostics) {
	attrs, diags := block.Body.JustAttributes()
	if len(attrs) == 0 {
		return nil, diags
	}

	locals := make([]*Local, 0, len(attrs))
	for name, attr := range attrs {
		if !hclsyntax.ValidIdentifier(name) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid local value name",
				Detail:   badIdentifierDetail,
				Subject:  &attr.NameRange,
			})
		}

		locals = append(locals, &Local{
			Name:      name,
			Expr:      attr.Expr,
			DeclRange: attr.Range,
		})
	}
	return locals, diags
}

// Addr returns the address of the local value declared by the receiver,
// relative to its containing module.
func (l *Local) Addr() addrs.LocalValue {
	return addrs.LocalValue{
		Name: l.Name,
	}
}

var variableBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "description",
		},
		{
			Name: "default",
		},
		{
			Name: "type",
		},
		{
			Name: "sensitive",
		},
		{
			Name: "nullable",
		},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "validation",
		},
	},
}

var outputBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "description",
		},
		{
			Name:     "value",
			Required: true,
		},
		{
			Name: "depends_on",
		},
		{
			Name: "sensitive",
		},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "precondition"},
		{Type: "postcondition"},
	},
}
