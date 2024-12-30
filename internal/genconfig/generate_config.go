// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package genconfig

import (
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
	var buf strings.Builder

	var diags tfdiags.Diagnostics

	if pc.LocalName != addr.Resource.Resource.ImpliedProvider() || pc.Alias != "" {
		buf.WriteString(strings.Repeat(" ", 2))
		buf.WriteString(fmt.Sprintf("provider = %s\n", pc.StringCompact()))
	}

	if stateVal.RawEquals(cty.NilVal) {
		diags = diags.Append(writeConfigAttributes(addr, &buf, schema.Attributes, 2))
		diags = diags.Append(writeConfigBlocks(addr, &buf, schema.BlockTypes, 2))
	} else {
		diags = diags.Append(writeConfigAttributesFromExisting(addr, &buf, stateVal, schema.Attributes, 2))
		diags = diags.Append(writeConfigBlocksFromExisting(addr, &buf, stateVal, schema.BlockTypes, 2))
	}

	// The output better be valid HCL which can be parsed and formatted.
	formatted := hclwrite.Format([]byte(buf.String()))
	return string(formatted), diags
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

func writeConfigAttributes(addr addrs.AbsResourceInstance, buf *strings.Builder, attrs map[string]*configschema.Attribute, indent int) tfdiags.Diagnostics {
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
			diags = diags.Append(writeConfigNestedTypeAttribute(addr, buf, name, attrS, indent))
			continue
		}
		if attrS.Required {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = ", name))
			tok := hclwrite.TokensForValue(attrS.EmptyValue())
			if _, err := tok.WriteTo(buf); err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Skipped part of config generation",
					Detail:   fmt.Sprintf("Could not create attribute %s in %s when generating import configuration. The plan will likely report the missing attribute as being deleted.", name, addr),
					Extra:    err,
				})
				continue
			}
			writeAttrTypeConstraint(buf, attrS)
		} else if attrS.Optional {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = ", name))
			tok := hclwrite.TokensForValue(attrS.EmptyValue())
			if _, err := tok.WriteTo(buf); err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Skipped part of config generation",
					Detail:   fmt.Sprintf("Could not create attribute %s in %s when generating import configuration. The plan will likely report the missing attribute as being deleted.", name, addr),
					Extra:    err,
				})
				continue
			}
			writeAttrTypeConstraint(buf, attrS)
		}
	}
	return diags
}

func writeConfigAttributesFromExisting(addr addrs.AbsResourceInstance, buf *strings.Builder, stateVal cty.Value, attrs map[string]*configschema.Attribute, indent int) tfdiags.Diagnostics {
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
			writeConfigNestedTypeAttributeFromExisting(addr, buf, name, attrS, stateVal, indent)
			continue
		}

		// Exclude computed-only attributes
		if attrS.Required || attrS.Optional {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = ", name))

			var val cty.Value
			if !stateVal.IsNull() && stateVal.Type().HasAttribute(name) {
				val = stateVal.GetAttr(name)
			} else {
				val = attrS.EmptyValue()
			}
			if val.Type() == cty.String {
				// Before we inspect the string, take off any marks.
				unmarked, marks := val.Unmark()

				// SHAMELESS HACK: If we have "" for an optional value, assume
				// it is actually null, due to the legacy SDK.
				if !unmarked.IsNull() && attrS.Optional && len(unmarked.AsString()) == 0 {
					unmarked = attrS.EmptyValue()
				}

				// Before we carry on, add the marks back.
				val = unmarked.WithMarks(marks)
			}
			if attrS.Sensitive || val.IsMarked() {
				buf.WriteString("null # sensitive")
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

					// Lone deserializable primitive types are valid json, but should be treated as strings
					if ctyValue.Type().IsPrimitiveType() {
						if d := writeTokens(val, buf); d != nil {
							diags = diags.Append(d)
							continue
						}
					} else {
						buf.WriteString("jsonencode(")

						if d := writeTokens(ctyValue.Value, buf); d != nil {
							diags = diags.Append(d)
							continue
						}

						buf.WriteString(")")
					}
				} else {
					if d := writeTokens(val, buf); d != nil {
						diags = diags.Append(d)
						continue
					}
				}
			}

			buf.WriteString("\n")
		}
	}
	return diags
}

func writeTokens(val cty.Value, buf *strings.Builder) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	tok := hclwrite.TokensForValue(val)
	if _, err := tok.WriteTo(buf); err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Skipped part of config generation",
			Detail:   "Could not create attribute in import configuration. The plan will likely report the missing attribute as being deleted.",
			Extra:    err,
		})
	}
	return diags
}

func writeConfigBlocks(addr addrs.AbsResourceInstance, buf *strings.Builder, blocks map[string]*configschema.NestedBlock, indent int) tfdiags.Diagnostics {
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
		diags = diags.Append(writeConfigNestedBlock(addr, buf, name, blockS, indent))
	}
	return diags
}

func writeConfigNestedBlock(addr addrs.AbsResourceInstance, buf *strings.Builder, name string, schema *configschema.NestedBlock, indent int) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	switch schema.Nesting {
	case configschema.NestingSingle, configschema.NestingGroup:
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s {", name))
		writeBlockTypeConstraint(buf, schema)
		diags = diags.Append(writeConfigAttributes(addr, buf, schema.Attributes, indent+2))
		diags = diags.Append(writeConfigBlocks(addr, buf, schema.BlockTypes, indent+2))
		buf.WriteString("}\n")
		return diags
	case configschema.NestingList, configschema.NestingSet:
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s {", name))
		writeBlockTypeConstraint(buf, schema)
		diags = diags.Append(writeConfigAttributes(addr, buf, schema.Attributes, indent+2))
		diags = diags.Append(writeConfigBlocks(addr, buf, schema.BlockTypes, indent+2))
		buf.WriteString("}\n")
		return diags
	case configschema.NestingMap:
		buf.WriteString(strings.Repeat(" ", indent))
		// we use an arbitrary placeholder key (block label) "key"
		buf.WriteString(fmt.Sprintf("%s \"key\" {", name))
		writeBlockTypeConstraint(buf, schema)
		diags = diags.Append(writeConfigAttributes(addr, buf, schema.Attributes, indent+2))
		diags = diags.Append(writeConfigBlocks(addr, buf, schema.BlockTypes, indent+2))
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("}\n")
		return diags
	default:
		// This should not happen, the above should be exhaustive.
		panic(fmt.Errorf("unsupported NestingMode %s", schema.Nesting.String()))
	}
}

func writeConfigNestedTypeAttribute(addr addrs.AbsResourceInstance, buf *strings.Builder, name string, schema *configschema.Attribute, indent int) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	buf.WriteString(strings.Repeat(" ", indent))
	buf.WriteString(fmt.Sprintf("%s = ", name))

	switch schema.NestedType.Nesting {
	case configschema.NestingSingle:
		buf.WriteString("{")
		writeAttrTypeConstraint(buf, schema)
		diags = diags.Append(writeConfigAttributes(addr, buf, schema.NestedType.Attributes, indent+2))
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("}\n")
		return diags
	case configschema.NestingList, configschema.NestingSet:
		buf.WriteString("[{")
		writeAttrTypeConstraint(buf, schema)
		diags = diags.Append(writeConfigAttributes(addr, buf, schema.NestedType.Attributes, indent+2))
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("}]\n")
		return diags
	case configschema.NestingMap:
		buf.WriteString("{")
		writeAttrTypeConstraint(buf, schema)
		buf.WriteString(strings.Repeat(" ", indent+2))
		// we use an arbitrary placeholder key "key"
		buf.WriteString("key = {\n")
		diags = diags.Append(writeConfigAttributes(addr, buf, schema.NestedType.Attributes, indent+4))
		buf.WriteString(strings.Repeat(" ", indent+2))
		buf.WriteString("}\n")
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("}\n")
		return diags
	default:
		// This should not happen, the above should be exhaustive.
		panic(fmt.Errorf("unsupported NestingMode %s", schema.NestedType.Nesting.String()))
	}
}

func writeConfigBlocksFromExisting(addr addrs.AbsResourceInstance, buf *strings.Builder, stateVal cty.Value, blocks map[string]*configschema.NestedBlock, indent int) tfdiags.Diagnostics {
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
		diags = diags.Append(writeConfigNestedBlockFromExisting(addr, buf, name, blockS, blockVal, indent))
	}

	return diags
}

func writeConfigNestedTypeAttributeFromExisting(addr addrs.AbsResourceInstance, buf *strings.Builder, name string, schema *configschema.Attribute, stateVal cty.Value, indent int) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	switch schema.NestedType.Nesting {
	case configschema.NestingSingle:
		if schema.Sensitive || stateVal.IsMarked() {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = {} # sensitive\n", name))
			return diags
		}

		// This shouldn't happen in real usage; state always has all values (set
		// to null as needed), but it protects against panics in tests (and any
		// really weird and unlikely cases).
		if !stateVal.Type().HasAttribute(name) {
			return diags
		}
		nestedVal := stateVal.GetAttr(name)

		if nestedVal.IsNull() {
			// There is a difference between a null object, and an object with
			// no attributes.
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = null\n", name))
			return diags
		}

		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s = {\n", name))
		diags = diags.Append(writeConfigAttributesFromExisting(addr, buf, nestedVal, schema.NestedType.Attributes, indent+2))
		buf.WriteString("}\n")
		return diags

	case configschema.NestingList, configschema.NestingSet:

		if schema.Sensitive || stateVal.IsMarked() {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = [] # sensitive\n", name))
			return diags
		}

		listVals := ctyCollectionValues(stateVal.GetAttr(name))
		if listVals == nil {
			// There is a difference between an empty list and a null list
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = null\n", name))
			return diags
		}

		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s = [\n", name))
		for i := range listVals {
			buf.WriteString(strings.Repeat(" ", indent+2))

			// The entire element is marked.
			if listVals[i].IsMarked() {
				buf.WriteString("{}, # sensitive\n")
				continue
			}

			buf.WriteString("{\n")
			diags = diags.Append(writeConfigAttributesFromExisting(addr, buf, listVals[i], schema.NestedType.Attributes, indent+4))
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString("},\n")
		}
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("]\n")
		return diags

	case configschema.NestingMap:
		if schema.Sensitive || stateVal.IsMarked() {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = {} # sensitive\n", name))
			return diags
		}

		attr := stateVal.GetAttr(name)
		if attr.IsNull() {
			// There is a difference between an empty map and a null map.
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = null\n", name))
			return diags
		}

		vals := attr.AsValueMap()
		keys := make([]string, 0, len(vals))
		for key := range vals {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s = {\n", name))
		for _, key := range keys {
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString(fmt.Sprintf("%s = {", hclEscapeString(key)))

			// This entire value is marked
			if vals[key].IsMarked() {
				buf.WriteString("} # sensitive\n")
				continue
			}

			buf.WriteString("\n")
			diags = diags.Append(writeConfigAttributesFromExisting(addr, buf, vals[key], schema.NestedType.Attributes, indent+4))
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString("}\n")
		}
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("}\n")
		return diags

	default:
		// This should not happen, the above should be exhaustive.
		panic(fmt.Errorf("unsupported NestingMode %s", schema.NestedType.Nesting.String()))
	}
}

func writeConfigNestedBlockFromExisting(addr addrs.AbsResourceInstance, buf *strings.Builder, name string, schema *configschema.NestedBlock, stateVal cty.Value, indent int) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	switch schema.Nesting {
	case configschema.NestingSingle, configschema.NestingGroup:
		if stateVal.IsNull() {
			return diags
		}
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s {", name))

		// If the entire value is marked, don't print any nested attributes
		if stateVal.IsMarked() {
			buf.WriteString("} # sensitive\n")
			return diags
		}
		buf.WriteString("\n")
		diags = diags.Append(writeConfigAttributesFromExisting(addr, buf, stateVal, schema.Attributes, indent+2))
		diags = diags.Append(writeConfigBlocksFromExisting(addr, buf, stateVal, schema.BlockTypes, indent+2))
		buf.WriteString("}\n")
		return diags
	case configschema.NestingList, configschema.NestingSet:
		if stateVal.IsMarked() {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s {} # sensitive\n", name))
			return diags
		}
		listVals := ctyCollectionValues(stateVal)
		for i := range listVals {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s {\n", name))
			diags = diags.Append(writeConfigAttributesFromExisting(addr, buf, listVals[i], schema.Attributes, indent+2))
			diags = diags.Append(writeConfigBlocksFromExisting(addr, buf, listVals[i], schema.BlockTypes, indent+2))
			buf.WriteString("}\n")
		}
		return diags
	case configschema.NestingMap:
		// If the entire value is marked, don't print any nested attributes
		if stateVal.IsMarked() {
			buf.WriteString(fmt.Sprintf("%s {} # sensitive\n", name))
			return diags
		}

		vals := stateVal.AsValueMap()
		keys := make([]string, 0, len(vals))
		for key := range vals {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s %q {", name, key))
			// This entire map element is marked
			if vals[key].IsMarked() {
				buf.WriteString("} # sensitive\n")
				return diags
			}
			buf.WriteString("\n")
			diags = diags.Append(writeConfigAttributesFromExisting(addr, buf, vals[key], schema.Attributes, indent+2))
			diags = diags.Append(writeConfigBlocksFromExisting(addr, buf, vals[key], schema.BlockTypes, indent+2))
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString("}\n")
		}
		return diags
	default:
		// This should not happen, the above should be exhaustive.
		panic(fmt.Errorf("unsupported NestingMode %s", schema.Nesting.String()))
	}
}

func writeAttrTypeConstraint(buf *strings.Builder, schema *configschema.Attribute) {
	if schema.Required {
		buf.WriteString(" # REQUIRED ")
	} else {
		buf.WriteString(" # OPTIONAL ")
	}

	if schema.NestedType != nil {
		buf.WriteString(fmt.Sprintf("%s\n", schema.NestedType.ImpliedType().FriendlyName()))
	} else {
		buf.WriteString(fmt.Sprintf("%s\n", schema.Type.FriendlyName()))
	}
}

func writeBlockTypeConstraint(buf *strings.Builder, schema *configschema.NestedBlock) {
	if schema.MinItems > 0 {
		buf.WriteString(" # REQUIRED block\n")
	} else {
		buf.WriteString(" # OPTIONAL block\n")
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

// hclEscapeString formats the input string into a format that is safe for
// rendering within HCL.
//
// Note, this function doesn't actually do a very good job of this currently. We
// need to expose some internal functions from HCL in a future version and call
// them from here. For now, just use "%q" formatting.
//
// Note, the similar function in jsonformat/computed/renderers/map.go is doing
// something similar.
func hclEscapeString(str string) string {
	// TODO: Replace this with more complete HCL logic instead of the simple
	// go workaround.
	if !hclsyntax.ValidIdentifier(str) {
		return fmt.Sprintf("%q", str)
	}
	return str
}
