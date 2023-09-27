// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package typeexpr

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	hcltypeexpr "github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/stackconfigtypes"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

const invalidTypeSummary = "Invalid type specification"

type Defaults = hcltypeexpr.Defaults

// TypeConstraint attempts to parse the given expression as a type constraint
// and, if successful, returns the resulting type. If unsuccessful, error
// diagnostics are returned.
//
// A type constraint has the same structure as a type, but it additionally
// allows the keyword "any" to represent cty.DynamicPseudoType, which is often
// used as a wildcard in type checking and type conversion operations.
func TypeConstraint(expr hcl.Expression, typeInfo TypeInformation) (cty.Type, *Defaults, hcl.Diagnostics) {
	return getType(expr, typeInfo, true, true)
}

func getType(expr hcl.Expression, typeInfo TypeInformation, constraint, withDefaults bool) (cty.Type, *Defaults, hcl.Diagnostics) {
	// First we'll try for one of our keywords
	kw := hcl.ExprAsKeyword(expr)
	switch kw {
	case "bool":
		return cty.Bool, nil, nil
	case "string":
		return cty.String, nil, nil
	case "number":
		return cty.Number, nil, nil
	case "any":
		return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   "There is no automatic type inference placeholder \"any\". Write an explicit type constraint instead.",
			Subject:  expr.Range().Ptr(),
		}}
	case "list", "map", "set":
		return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   fmt.Sprintf("The %s type constructor requires one argument specifying the element type.", kw),
			Subject:  expr.Range().Ptr(),
		}}
	case "object":
		return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   "The object type constructor requires one argument specifying the attribute types and values as a map.",
			Subject:  expr.Range().Ptr(),
		}}
	case "tuple":
		return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   "The tuple type constructor requires one argument specifying the element types as a list.",
			Subject:  expr.Range().Ptr(),
		}}
	case "providerconfig":
		return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   "The providerconfig type constructor requires one argument specifying the local name of the provider this configuration is for.",
			Subject:  expr.Range().Ptr(),
		}}
	case "":
		// okay! we'll fall through and try processing as a call, then.
	default:
		return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   fmt.Sprintf("The keyword %q is not a valid type specification.", kw),
			Subject:  expr.Range().Ptr(),
		}}
	}

	// If we get down here then our expression isn't just a keyword, so we'll
	// try to process it as a call instead.
	call, diags := hcl.ExprCall(expr)
	if diags.HasErrors() {
		return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   "A type specification is either a primitive type keyword (bool, number, string) or a complex type constructor call, like list(string).",
			Subject:  expr.Range().Ptr(),
		}}
	}

	switch call.Name {
	case "bool", "string", "number":
		return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   fmt.Sprintf("Primitive type keyword %q does not expect arguments.", call.Name),
			Subject:  &call.ArgsRange,
		}}
	case "any":
		return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   fmt.Sprintf("Type constraint keyword %q does not expect arguments.", call.Name),
			Subject:  &call.ArgsRange,
		}}
	}

	if len(call.Arguments) != 1 {
		contextRange := call.ArgsRange
		subjectRange := call.ArgsRange
		if len(call.Arguments) > 1 {
			// If we have too many arguments (as opposed to too _few_) then
			// we'll highlight the extraneous arguments as the diagnostic
			// subject.
			subjectRange = hcl.RangeBetween(call.Arguments[1].Range(), call.Arguments[len(call.Arguments)-1].Range())
		}

		switch call.Name {
		case "list", "set", "map":
			return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  invalidTypeSummary,
				Detail:   fmt.Sprintf("The %s type constructor requires one argument specifying the element type.", call.Name),
				Subject:  &subjectRange,
				Context:  &contextRange,
			}}
		case "object":
			return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  invalidTypeSummary,
				Detail:   "The object type constructor requires one argument specifying the attribute types and values as a map.",
				Subject:  &subjectRange,
				Context:  &contextRange,
			}}
		case "tuple":
			return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  invalidTypeSummary,
				Detail:   "The tuple type constructor requires one argument specifying the element types as a list.",
				Subject:  &subjectRange,
				Context:  &contextRange,
			}}
		case "providerconfig":
			return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  invalidTypeSummary,
				Detail:   "The providerconfig type constructor requires one argument specifying the local name of the provider this configuration is for.",
				Subject:  &subjectRange,
				Context:  &contextRange,
			}}
		}
	}

	switch call.Name {

	case "list":
		ety, defaults, diags := getType(call.Arguments[0], typeInfo, constraint, withDefaults)
		ty := cty.List(ety)
		return ty, collectionDefaults(ty, defaults), diags
	case "set":
		ety, defaults, diags := getType(call.Arguments[0], typeInfo, constraint, withDefaults)
		ty := cty.Set(ety)
		return ty, collectionDefaults(ty, defaults), diags
	case "map":
		ety, defaults, diags := getType(call.Arguments[0], typeInfo, constraint, withDefaults)
		ty := cty.Map(ety)
		return ty, collectionDefaults(ty, defaults), diags
	case "object":
		attrDefs, diags := hcl.ExprMap(call.Arguments[0])
		if diags.HasErrors() {
			return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  invalidTypeSummary,
				Detail:   "Object type constructor requires a map whose keys are attribute names and whose values are the corresponding attribute types.",
				Subject:  call.Arguments[0].Range().Ptr(),
				Context:  expr.Range().Ptr(),
			}}
		}

		atys := make(map[string]cty.Type)
		defaultValues := make(map[string]cty.Value)
		children := make(map[string]*Defaults)
		var optAttrs []string
		for _, attrDef := range attrDefs {
			attrName := hcl.ExprAsKeyword(attrDef.Key)
			if attrName == "" {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  invalidTypeSummary,
					Detail:   "Object constructor map keys must be attribute names.",
					Subject:  attrDef.Key.Range().Ptr(),
					Context:  expr.Range().Ptr(),
				})
				continue
			}
			atyExpr := attrDef.Value

			// the attribute type expression might be wrapped in the special
			// modifier optional(...) to indicate an optional attribute. If
			// so, we'll unwrap that first and make a note about it being
			// optional for when we construct the type below.
			var defaultExpr hcl.Expression
			if call, callDiags := hcl.ExprCall(atyExpr); !callDiags.HasErrors() {
				if call.Name == "optional" {
					if len(call.Arguments) < 1 {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  invalidTypeSummary,
							Detail:   "Optional attribute modifier requires the attribute type as its argument.",
							Subject:  call.ArgsRange.Ptr(),
							Context:  atyExpr.Range().Ptr(),
						})
						continue
					}
					if constraint {
						if withDefaults {
							switch len(call.Arguments) {
							case 2:
								defaultExpr = call.Arguments[1]
								defaultVal, defaultDiags := defaultExpr.Value(nil)
								diags = append(diags, defaultDiags...)
								if !defaultDiags.HasErrors() {
									optAttrs = append(optAttrs, attrName)
									defaultValues[attrName] = defaultVal
								}
							case 1:
								optAttrs = append(optAttrs, attrName)
							default:
								diags = append(diags, &hcl.Diagnostic{
									Severity: hcl.DiagError,
									Summary:  invalidTypeSummary,
									Detail:   "Optional attribute modifier expects at most two arguments: the attribute type, and a default value.",
									Subject:  call.ArgsRange.Ptr(),
									Context:  atyExpr.Range().Ptr(),
								})
							}
						} else {
							if len(call.Arguments) == 1 {
								optAttrs = append(optAttrs, attrName)
							} else {
								diags = append(diags, &hcl.Diagnostic{
									Severity: hcl.DiagError,
									Summary:  invalidTypeSummary,
									Detail:   "Optional attribute modifier expects only one argument: the attribute type.",
									Subject:  call.ArgsRange.Ptr(),
									Context:  atyExpr.Range().Ptr(),
								})
							}
						}
					} else {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  invalidTypeSummary,
							Detail:   "Optional attribute modifier is only for type constraints, not for exact types.",
							Subject:  call.NameRange.Ptr(),
							Context:  atyExpr.Range().Ptr(),
						})
					}
					atyExpr = call.Arguments[0]
				}
			}

			aty, aDefaults, attrDiags := getType(atyExpr, typeInfo, constraint, withDefaults)
			diags = append(diags, attrDiags...)

			// If a default is set for an optional attribute, verify that it is
			// convertible to the attribute type.
			if defaultVal, ok := defaultValues[attrName]; ok {
				convertedDefaultVal, err := convert.Convert(defaultVal, aty)
				if err != nil {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid default value for optional attribute",
						Detail:   fmt.Sprintf("This default value is not compatible with the attribute's type constraint: %s.", err),
						Subject:  defaultExpr.Range().Ptr(),
					})
					delete(defaultValues, attrName)
				} else {
					defaultValues[attrName] = convertedDefaultVal
				}
			}

			atys[attrName] = aty
			if aDefaults != nil {
				children[attrName] = aDefaults
			}
		}
		ty := cty.ObjectWithOptionalAttrs(atys, optAttrs)
		return ty, structuredDefaults(ty, defaultValues, children), diags
	case "tuple":
		elemDefs, diags := hcl.ExprList(call.Arguments[0])
		if diags.HasErrors() {
			return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  invalidTypeSummary,
				Detail:   "Tuple type constructor requires a list of element types.",
				Subject:  call.Arguments[0].Range().Ptr(),
				Context:  expr.Range().Ptr(),
			}}
		}
		etys := make([]cty.Type, len(elemDefs))
		children := make(map[string]*Defaults, len(elemDefs))
		for i, defExpr := range elemDefs {
			ety, elemDefaults, elemDiags := getType(defExpr, typeInfo, constraint, withDefaults)
			diags = append(diags, elemDiags...)
			etys[i] = ety
			if elemDefaults != nil {
				children[fmt.Sprintf("%d", i)] = elemDefaults
			}
		}
		ty := cty.Tuple(etys)
		return ty, structuredDefaults(ty, nil, children), diags
	case "providerconfig":
		localName := hcl.ExprAsKeyword(call.Arguments[0])
		if localName == "" {
			return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  invalidTypeSummary,
				Detail:   "The argument to providerconfig must be just the local name of the provider this configuration is for, given as an identifier.",
				Subject:  call.Arguments[0].Range().Ptr(),
			}}
		}
		providerAddr, ok := typeInfo.ProviderForLocalName(localName)
		if !ok {
			return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  invalidTypeSummary,
				Detail:   fmt.Sprintf("The name %q does not match any provider local name defined in this configuration's required_providers block.", localName),
				Subject:  call.Arguments[0].Range().Ptr(),
			}}
		}
		ty := typeInfo.ProviderConfigType(providerAddr)
		if ty == cty.NilType {
			ty = stackconfigtypes.ProviderConfigType(providerAddr)
			typeInfo.SetProviderConfigType(providerAddr, ty)
		}
		return ty, nil, diags
	case "optional":
		return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   fmt.Sprintf("Keyword %q is valid only as a modifier for object type attributes.", call.Name),
			Subject:  call.NameRange.Ptr(),
		}}
	default:
		// Can't access call.Arguments in this path because we've not validated
		// that it contains exactly one expression here.
		return cty.DynamicPseudoType, nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  invalidTypeSummary,
			Detail:   fmt.Sprintf("Keyword %q is not a valid type constructor.", call.Name),
			Subject:  expr.Range().Ptr(),
		}}
	}
}

func collectionDefaults(ty cty.Type, defaults *Defaults) *Defaults {
	if defaults == nil {
		return nil
	}
	return &Defaults{
		Type: ty,
		Children: map[string]*Defaults{
			"": defaults,
		},
	}
}

func structuredDefaults(ty cty.Type, defaultValues map[string]cty.Value, children map[string]*Defaults) *Defaults {
	if len(defaultValues) == 0 && len(children) == 0 {
		return nil
	}

	defaults := &Defaults{
		Type: ty,
	}
	if len(defaultValues) > 0 {
		defaults.DefaultValues = defaultValues
	}
	if len(children) > 0 {
		defaults.Children = children
	}

	return defaults
}
