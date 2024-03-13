// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package genconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// GenerateResourceContents generates HCL configuration code for the provided
// resource and state value.
//
// If you want to generate actual valid Terraform code you should follow this
// call up with a call to WrapResourceContents, which will place a Terraform
// resource header around the attributes and blocks returned by this function.
func GenerateResourceContents(addr addrs.AbsResourceInstance,
	schema *configschema.Block,
	pc addrs.LocalProviderConfig,
	stateVal cty.Value) (string, tfdiags.Diagnostics) {

	// We're not actually generating an entire file, but that's the easiest
	// way for us to get an empty HCL body to write into.
	f := hclwrite.NewEmptyFile()
	body := f.Body()

	var diags tfdiags.Diagnostics

	if pc.LocalName != addr.Resource.Resource.ImpliedProvider() || pc.Alias != "" {
		traversal := hcl.Traversal{
			hcl.TraverseRoot{
				Name: pc.LocalName,
			},
		}
		if pc.Alias != "" {
			traversal = append(traversal, hcl.TraverseAttr{
				Name: pc.Alias,
			})
		}
		body.SetAttributeTraversal("provider", traversal)
		// A newline here is suggested in our documented style conventions, but
		// the previous implementation didn't do it so we won't either.
		//body.AppendNewline()
	}

	stateVal = omitUnknowns(stateVal)
	if stateVal.RawEquals(cty.NilVal) {
		diags = diags.Append(writeConfigAttributes(addr, body, schema.Attributes))
		diags = diags.Append(writeConfigBlocks(addr, body, schema.BlockTypes))
	} else {
		diags = diags.Append(writeConfigAttributesFromExisting(addr, body, stateVal, schema.Attributes))
		diags = diags.Append(writeConfigBlocksFromExisting(addr, body, stateVal, schema.BlockTypes))
	}

	return string(f.Bytes()), diags
}

func WrapResourceContents(addr addrs.AbsResourceInstance, config string) string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("resource %q %q {\n", addr.Resource.Resource.Type, addr.Resource.Resource.Name))
	buf.WriteString(config)
	buf.WriteString("}")

	// The output better be valid HCL which can be parsed and formatted.
	formatted := hclwrite.Format([]byte(buf.String()))
	return string(formatted)
}

func writeConfigAttributes(addr addrs.AbsResourceInstance, body *hclwrite.Body, attrs map[string]*configschema.Attribute) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if len(attrs) == 0 {
		return diags
	}

	// Get a list of sorted attribute names so the output will be consistent between runs.
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i := range keys {
		name := keys[i]
		attrS := attrs[name]
		if attrS.NestedType != nil {
			tok, moreDiags := tokensForNestedObject(addr, attrS.NestedType)
			diags = diags.Append(moreDiags)
			body.SetAttributeRaw(name, tok)
			continue
		}
		tok := hclwrite.TokensForValue(attrS.EmptyValue())
		tok = append(tok, attrTypeConstraintComment(attrS)...)
		body.SetAttributeRaw(name, tok)
	}
	return diags
}

func writeConfigAttributesFromExisting(addr addrs.AbsResourceInstance, body *hclwrite.Body, stateVal cty.Value, attrs map[string]*configschema.Attribute) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if len(attrs) == 0 {
		return diags
	}

	// Get a list of sorted attribute names so the output will be consistent between runs.
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i := range keys {
		name := keys[i]
		attrS := attrs[name]

		// Exclude computed-only attributes
		if attrS.Required || attrS.Optional {
			var val cty.Value
			if !stateVal.IsNull() && stateVal.Type().HasAttribute(name) {
				val = stateVal.GetAttr(name)
			} else {
				val = attrS.EmptyValue()
			}
			if val.Type() == cty.String {
				// SHAMELESS HACK: If we have "" for an optional value, assume
				// it is actually null, due to the legacy SDK.
				if !val.IsNull() && attrS.Optional && len(val.AsString()) == 0 {
					val = attrS.EmptyValue()
				}
			}

			var tok hclwrite.Tokens
			if attrS.Sensitive || val.IsMarked() {
				tok = hclwrite.Tokens{
					{
						Type:  hclsyntax.TokenIdent,
						Bytes: []byte("null"),
					},
					{
						Type:  hclsyntax.TokenComment,
						Bytes: []byte("# sensitive\n"),
					},
				}
			} else {
				// If the value is a string storing a JSON value we want to represent it in a terraform native way
				// and encapsulate it in `jsonencode` as it is the idiomatic representation
				if val.IsKnown() && !val.IsNull() && val.Type() == cty.String && json.Valid([]byte(val.AsString())) {
					var ctyValue ctyjson.SimpleJSONValue
					err := ctyValue.UnmarshalJSON([]byte(val.AsString()))
					if err != nil {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagWarning,
							Summary:  "Failed to parse JSON",
							Detail:   fmt.Sprintf("Could not parse JSON value of attribute %s in %s when generating import configuration. The plan will likely report the missing attribute as being deleted. This is most likely a bug in Terraform, please report it.", name, addr),
							Extra:    err,
						})
						continue
					}

					argTok := hclwrite.TokensForValue(ctyValue.Value)
					tok = hclwrite.TokensForFunctionCall("jsonencode", argTok)
				} else {
					tok = hclwrite.TokensForValue(val)
				}
			}
			body.SetAttributeRaw(name, tok)
		}
	}
	return diags
}

func writeConfigBlocks(addr addrs.AbsResourceInstance, body *hclwrite.Body, blocks map[string]*configschema.NestedBlock) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if len(blocks) == 0 {
		return diags
	}

	// Get a list of sorted block names so the output will be consistent between runs.
	names := make([]string, 0, len(blocks))
	for k := range blocks {
		names = append(names, k)
	}
	sort.Strings(names)

	for i := range names {
		name := names[i]
		blockS := blocks[name]
		diags = diags.Append(writeConfigNestedBlock(addr, body, name, blockS))
	}
	return diags
}

func writeConfigNestedBlock(addr addrs.AbsResourceInstance, body *hclwrite.Body, name string, schema *configschema.NestedBlock) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	switch schema.Nesting {
	case configschema.NestingSingle, configschema.NestingGroup, configschema.NestingList, configschema.NestingSet:
		// These nesting modes all require zero labels on the blocks.
		block := body.AppendNewBlock(name, nil)
		block.Body().AppendUnstructuredTokens(blockTypeConstraintComment(schema))
		diags = diags.Append(writeConfigAttributes(addr, block.Body(), schema.Attributes))
		diags = diags.Append(writeConfigBlocks(addr, block.Body(), schema.BlockTypes))
		return diags
	case configschema.NestingMap:
		// This nesting mode requires one label that serves as the map key.
		// We use an arbitrary placeholder label "key".
		block := body.AppendNewBlock(name, []string{"key"})
		block.Body().AppendUnstructuredTokens(blockTypeConstraintComment(schema))
		diags = diags.Append(writeConfigAttributes(addr, block.Body(), schema.Attributes))
		diags = diags.Append(writeConfigBlocks(addr, block.Body(), schema.BlockTypes))
		return diags
	default:
		// This should not happen, the above should be exhaustive.
		panic(fmt.Errorf("unsupported NestingMode %s", schema.Nesting.String()))
	}
}

func tokensForNestedObject(addr addrs.AbsResourceInstance, schema *configschema.Object) (hclwrite.Tokens, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	item, moreDiags := tokensForNestedObjectItem(addr, schema.Attributes)
	diags = diags.Append(moreDiags)

	switch schema.Nesting {
	case configschema.NestingSingle, configschema.NestingGroup:
		return item, diags
	case configschema.NestingList, configschema.NestingSet:
		return hclwrite.TokensForTuple([]hclwrite.Tokens{item}), diags
	case configschema.NestingMap:
		return hclwrite.TokensForObject([]hclwrite.ObjectAttrTokens{
			{
				Name:  hclwrite.TokensForIdentifier("key"),
				Value: item,
			},
		}), diags
	default:
		// This should not happen, because the above should be exhaustive.
		panic(fmt.Errorf("unsupported NestingMode %s", schema.Nesting.String()))
	}

}

func tokensForNestedObjectItem(addr addrs.AbsResourceInstance, attrs map[string]*configschema.Attribute) (hclwrite.Tokens, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	names := make([]string, 0, len(attrs))
	for k := range attrs {
		names = append(names, k)
	}
	sort.Strings(names)

	var items []hclwrite.ObjectAttrTokens
	for _, name := range names {
		attrS := attrs[name]

		key := hclwrite.TokensForIdentifier("name")
		var value hclwrite.Tokens
		if attrS.NestedType != nil {
			var moreDiags tfdiags.Diagnostics
			value, moreDiags = tokensForNestedObject(addr, attrS.NestedType)
			diags = diags.Append(moreDiags)
			value = append(value, attrTypeConstraintComment(attrS)...)
		} else {
			value = hclwrite.TokensForValue(attrS.EmptyValue())
		}
		value = append(value, attrTypeConstraintComment(attrS)...)

		items = append(items, hclwrite.ObjectAttrTokens{
			Name:  key,
			Value: value,
		})
	}

	return hclwrite.TokensForObject(items), diags
}

func writeConfigBlocksFromExisting(addr addrs.AbsResourceInstance, body *hclwrite.Body, stateVal cty.Value, blocks map[string]*configschema.NestedBlock) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if len(blocks) == 0 {
		return diags
	}

	// Get a list of sorted block names so the output will be consistent between runs.
	names := make([]string, 0, len(blocks))
	for k := range blocks {
		names = append(names, k)
	}
	sort.Strings(names)

	for _, name := range names {
		blockS := blocks[name]
		// This shouldn't happen in real usage; state always has all values (set
		// to null as needed), but it protects against panics in tests (and any
		// really weird and unlikely cases).
		if !stateVal.Type().HasAttribute(name) {
			continue
		}
		blockVal := stateVal.GetAttr(name)
		diags = diags.Append(writeConfigNestedBlockFromExisting(addr, body, name, blockS, blockVal))
	}

	return diags
}

func writeConfigNestedBlockFromExisting(addr addrs.AbsResourceInstance, body *hclwrite.Body, name string, schema *configschema.NestedBlock, stateVal cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	switch schema.Nesting {
	case configschema.NestingSingle, configschema.NestingGroup:
		if stateVal.IsNull() {
			return diags
		}
		block := body.AppendNewBlock(name, nil)

		if stateVal.IsMarked() {
			// If the entire value is marked, don't print any nested attributes
			block.Body().AppendUnstructuredTokens(hclwrite.Tokens{
				{
					Type:  hclsyntax.TokenComment,
					Bytes: []byte("# sensitive\n"),
				},
			})
			return diags
		}
		diags = diags.Append(writeConfigAttributesFromExisting(addr, block.Body(), stateVal, schema.Attributes))
		diags = diags.Append(writeConfigBlocksFromExisting(addr, block.Body(), stateVal, schema.BlockTypes))
		return diags
	case configschema.NestingList, configschema.NestingSet:
		if stateVal.IsMarked() {
			// If the entire value is marked, don't print any nested attributes
			block := body.AppendNewBlock(name, nil)
			block.Body().AppendUnstructuredTokens(hclwrite.Tokens{
				{
					Type:  hclsyntax.TokenComment,
					Bytes: []byte("# sensitive\n"),
				},
			})
			return diags
		}
		listVals := ctyCollectionValues(stateVal)
		for i := range listVals {
			block := body.AppendNewBlock(name, nil)
			diags = diags.Append(writeConfigAttributesFromExisting(addr, block.Body(), listVals[i], schema.Attributes))
			diags = diags.Append(writeConfigBlocksFromExisting(addr, block.Body(), listVals[i], schema.BlockTypes))
		}
		return diags
	case configschema.NestingMap:
		if stateVal.IsMarked() {
			// If the entire value is marked, don't print any nested attributes
			block := body.AppendNewBlock(name, []string{"..."})
			block.Body().AppendUnstructuredTokens(hclwrite.Tokens{
				{
					Type:  hclsyntax.TokenComment,
					Bytes: []byte("# sensitive\n"),
				},
			})
			return diags
		}

		vals := stateVal.AsValueMap()
		keys := make([]string, 0, len(vals))
		for key := range vals {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			block := body.AppendNewBlock(name, []string{key})
			// This entire map element is marked
			if vals[key].IsMarked() {
				block.Body().AppendUnstructuredTokens(hclwrite.Tokens{
					{
						Type:  hclsyntax.TokenComment,
						Bytes: []byte("# sensitive\n"),
					},
				})
				return diags
			}
			diags = diags.Append(writeConfigAttributesFromExisting(addr, block.Body(), vals[key], schema.Attributes))
			diags = diags.Append(writeConfigBlocksFromExisting(addr, block.Body(), vals[key], schema.BlockTypes))
		}
		return diags
	default:
		// This should not happen, the above should be exhaustive.
		panic(fmt.Errorf("unsupported NestingMode %s", schema.Nesting.String()))
	}
}

func attrTypeConstraintComment(schema *configschema.Attribute) hclwrite.Tokens {
	var buf bytes.Buffer

	if schema.Required {
		buf.WriteString("# REQUIRED ")
	} else {
		buf.WriteString("# OPTIONAL ")
	}

	if schema.NestedType != nil {
		buf.WriteString(schema.NestedType.ImpliedType().FriendlyName())
	} else {
		buf.WriteString(schema.Type.FriendlyName())
	}

	return hclwrite.Tokens{
		{
			Type:  hclsyntax.TokenComment,
			Bytes: buf.Bytes(),
		},
	}
}

func blockTypeConstraintComment(schema *configschema.NestedBlock) hclwrite.Tokens {
	if schema.MinItems > 0 {
		return hclwrite.Tokens{
			{
				Type:  hclsyntax.TokenComment,
				Bytes: []byte("# REQUIRED block\n"),
			},
		}
	} else {
		return hclwrite.Tokens{
			{
				Type:  hclsyntax.TokenComment,
				Bytes: []byte("# OPTIONAL block\n"),
			},
		}
	}
}

// copied from command/format/diff
func ctyCollectionValues(val cty.Value) []cty.Value {
	if !val.IsKnown() || val.IsNull() {
		return nil
	}

	var len int
	if val.IsMarked() {
		val, _ = val.Unmark()
		len = val.LengthInt()
	} else {
		len = val.LengthInt()
	}

	ret := make([]cty.Value, 0, len)
	for it := val.ElementIterator(); it.Next(); {
		_, value := it.Element()
		ret = append(ret, value)
	}

	return ret
}

// omitUnknowns recursively walks the src cty.Value and returns a new cty.Value,
// omitting any unknowns.
//
// The result also normalizes some types: all sequence types are turned into
// tuple types and all mapping types are converted to object types, since we
// assume the result of this is just going to be serialized as JSON (and thus
// lose those distinctions) anyway.
func omitUnknowns(val cty.Value) cty.Value {
	ty := val.Type()
	switch {
	case val.IsNull():
		return val
	case !val.IsKnown():
		return cty.NilVal
	case ty.IsPrimitiveType():
		return val
	case ty.IsListType() || ty.IsTupleType() || ty.IsSetType():
		var vals []cty.Value
		it := val.ElementIterator()
		for it.Next() {
			_, v := it.Element()
			newVal := omitUnknowns(v)
			if newVal != cty.NilVal {
				vals = append(vals, newVal)
			} else if newVal == cty.NilVal {
				// element order is how we correlate unknownness, so we must
				// replace unknowns with nulls
				vals = append(vals, cty.NullVal(v.Type()))
			}
		}
		// We use tuple types always here, because the work we did above
		// may have caused the individual elements to have different types,
		// and we're doing this work to produce JSON anyway and JSON marshalling
		// represents all of these sequence types as an array.
		return cty.TupleVal(vals)
	case ty.IsMapType() || ty.IsObjectType():
		vals := make(map[string]cty.Value)
		it := val.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			newVal := omitUnknowns(v)
			if newVal != cty.NilVal {
				vals[k.AsString()] = newVal
			}
		}
		// We use object types always here, because the work we did above
		// may have caused the individual elements to have different types,
		// and we're doing this work to produce JSON anyway and JSON marshalling
		// represents both of these mapping types as an object.
		return cty.ObjectVal(vals)
	default:
		// Should never happen, since the above should cover all types
		panic(fmt.Sprintf("omitUnknowns cannot handle %#v", val))
	}
}
