// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package genconfig

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

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

	stateVal = omitUnknowns(stateVal)
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
				// SHAMELESS HACK: If we have "" for an optional value, assume
				// it is actually null, due to the legacy SDK.
				if !val.IsNull() && attrS.Optional && len(val.AsString()) == 0 {
					val = attrS.EmptyValue()
				}
			}
			if attrS.Sensitive || val.IsMarked() {
				buf.WriteString("null # sensitive")
			} else {
				tok := hclwrite.TokensForValue(val)
				if _, err := tok.WriteTo(buf); err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  "Skipped part of config generation",
						Detail:   fmt.Sprintf("Could not create attribute %s in %s when generating import configuration. The plan will likely report the missing attribute as being deleted.", name, addr),
						Extra:    err,
					})
					continue
				}
			}

			buf.WriteString("\n")
		}
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
			buf.WriteString(fmt.Sprintf("%s = {", key))

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
