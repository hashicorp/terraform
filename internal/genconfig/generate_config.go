package genconfig

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

func GenerateAttributesForResource(addr addrs.AbsResourceInstance,
	schema *configschema.Block,
	stateVal cty.Value) (string, error) {
	var buf strings.Builder

	// KEM: This doesn't quite work yet as the provider pc is not plumbed in.
	// if pc.LocalName != addr.Resource.Resource.ImpliedProvider() || pc.Alias != "" {
	// 	buf.WriteString(strings.Repeat(" ", 2))
	// 	buf.WriteString(fmt.Sprintf("provider = %s\n", pc.StringCompact()))
	// }

	stateVal = omitUnknowns(stateVal)

	if stateVal.RawEquals(cty.NilVal) {
		if err := writeConfigAttributes(&buf, schema.Attributes, 2); err != nil {
			return "", err
		}
		if err := writeConfigBlocks(&buf, schema.BlockTypes, 2); err != nil {
			return "", err
		}
	} else {
		if err := writeConfigAttributesFromExisting(&buf, stateVal, schema.Attributes, 2); err != nil {
			return "", err
		}
		if err := writeConfigBlocksFromExisting(&buf, stateVal, schema.BlockTypes, 2); err != nil {
			return "", err
		}
	}

	// The output better be valid HCL which can be parsed and formatted.
	formatted := hclwrite.Format([]byte(buf.String()))

	return string(formatted), nil
}

func GenerateConfigForResource(addr addrs.AbsResourceInstance,
	schema *configschema.Block,
	// pc addrs.LocalProviderConfig,
	stateVal cty.Value) (string, error) {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("resource %q %q {\n", addr.Resource.Resource.Type, addr.Resource.Resource.Name))

	// FIXME KEM
	// if pc.LocalName != addr.Resource.Resource.ImpliedProvider() || pc.Alias != "" {
	// 	buf.WriteString(strings.Repeat(" ", 2))
	// 	buf.WriteString(fmt.Sprintf("provider = %s\n", pc.StringCompact()))
	// }

	stateVal = omitUnknowns(stateVal)

	if stateVal.RawEquals(cty.NilVal) {
		if err := writeConfigAttributes(&buf, schema.Attributes, 2); err != nil {
			return "", err
		}
		if err := writeConfigBlocks(&buf, schema.BlockTypes, 2); err != nil {
			return "", err
		}
	} else {
		if err := writeConfigAttributesFromExisting(&buf, stateVal, schema.Attributes, 2); err != nil {
			return "", err
		}
		if err := writeConfigBlocksFromExisting(&buf, stateVal, schema.BlockTypes, 2); err != nil {
			return "", err
		}
	}

	buf.WriteString("}")

	// The output better be valid HCL which can be parsed and formatted.
	formatted := hclwrite.Format([]byte(buf.String()))

	return string(formatted), nil
}

func writeConfigAttributes(buf *strings.Builder, attrs map[string]*configschema.Attribute, indent int) error {
	if len(attrs) == 0 {
		return nil
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
			if err := writeConfigNestedTypeAttribute(buf, name, attrS, indent); err != nil {
				return err
			}
			continue
		}
		if attrS.Required {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = ", name))
			tok := hclwrite.TokensForValue(attrS.EmptyValue())
			if _, err := tok.WriteTo(buf); err != nil {
				return err
			}
			writeAttrTypeConstraint(buf, attrS)
		} else if attrS.Optional {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = ", name))
			tok := hclwrite.TokensForValue(attrS.EmptyValue())
			if _, err := tok.WriteTo(buf); err != nil {
				return err
			}
			writeAttrTypeConstraint(buf, attrS)
		}
	}
	return nil
}

func writeConfigAttributesFromExisting(buf *strings.Builder, stateVal cty.Value, attrs map[string]*configschema.Attribute, indent int) error {
	if len(attrs) == 0 {
		return nil
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
			if err := writeConfigNestedTypeAttributeFromExisting(buf, name, attrS, stateVal, indent); err != nil {
				return err
			}
			continue
		}

		// Exclude computed-only attributes
		if attrS.Required || attrS.Optional {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = ", name))

			var val cty.Value
			if stateVal.Type().HasAttribute(name) {
				val = stateVal.GetAttr(name)
			} else {
				val = attrS.EmptyValue()
			}
			if val.Type() == cty.String {
				// SHAMELESS HACK: If we have "" for an optional value, assume
				// it is actually null, due to the legacy SDK.
				if attrS.Optional && len(val.AsString()) == 0 {
					val = attrS.EmptyValue()
				}
			}
			if attrS.Sensitive || val.IsMarked() {
				buf.WriteString("null # sensitive")
			} else {
				tok := hclwrite.TokensForValue(val)
				if _, err := tok.WriteTo(buf); err != nil {
					return err
				}
			}

			buf.WriteString("\n")
		}
	}
	return nil
}

func writeConfigBlocks(buf *strings.Builder, blocks map[string]*configschema.NestedBlock, indent int) error {
	if len(blocks) == 0 {
		return nil
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
		if err := writeConfigNestedBlock(buf, name, blockS, indent); err != nil {
			return err
		}
	}
	return nil
}

func writeConfigNestedBlock(buf *strings.Builder, name string, schema *configschema.NestedBlock, indent int) error {
	switch schema.Nesting {
	case configschema.NestingSingle, configschema.NestingGroup:
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s {", name))
		writeBlockTypeConstraint(buf, schema)
		if err := writeConfigAttributes(buf, schema.Attributes, indent+2); err != nil {
			return err
		}
		if err := writeConfigBlocks(buf, schema.BlockTypes, indent+2); err != nil {
			return err
		}
		buf.WriteString("}\n")
		return nil
	case configschema.NestingList, configschema.NestingSet:
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s {", name))
		writeBlockTypeConstraint(buf, schema)
		if err := writeConfigAttributes(buf, schema.Attributes, indent+2); err != nil {
			return err
		}
		if err := writeConfigBlocks(buf, schema.BlockTypes, indent+2); err != nil {
			return err
		}
		buf.WriteString("}\n")
		return nil
	case configschema.NestingMap:
		buf.WriteString(strings.Repeat(" ", indent))
		// we use an arbitrary placeholder key (block label) "key"
		buf.WriteString(fmt.Sprintf("%s \"key\" {", name))
		writeBlockTypeConstraint(buf, schema)
		if err := writeConfigAttributes(buf, schema.Attributes, indent+2); err != nil {
			return err
		}
		if err := writeConfigBlocks(buf, schema.BlockTypes, indent+2); err != nil {
			return err
		}
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("}\n")
		return nil
	default:
		// This should not happen, the above should be exhaustive.
		return fmt.Errorf("unsupported NestingMode %s", schema.Nesting.String())
	}
}

func writeConfigNestedTypeAttribute(buf *strings.Builder, name string, schema *configschema.Attribute, indent int) error {
	buf.WriteString(strings.Repeat(" ", indent))
	buf.WriteString(fmt.Sprintf("%s = ", name))

	switch schema.NestedType.Nesting {
	case configschema.NestingSingle:
		buf.WriteString("{")
		writeAttrTypeConstraint(buf, schema)
		if err := writeConfigAttributes(buf, schema.NestedType.Attributes, indent+2); err != nil {
			return err
		}
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("}\n")
		return nil
	case configschema.NestingList, configschema.NestingSet:
		buf.WriteString("[{")
		writeAttrTypeConstraint(buf, schema)
		if err := writeConfigAttributes(buf, schema.NestedType.Attributes, indent+2); err != nil {
			return err
		}
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("}]\n")
		return nil
	case configschema.NestingMap:
		buf.WriteString("{")
		writeAttrTypeConstraint(buf, schema)
		buf.WriteString(strings.Repeat(" ", indent+2))
		// we use an arbitrary placeholder key "key"
		buf.WriteString("key = {\n")
		if err := writeConfigAttributes(buf, schema.NestedType.Attributes, indent+4); err != nil {
			return err
		}
		buf.WriteString(strings.Repeat(" ", indent+2))
		buf.WriteString("}\n")
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("}\n")
		return nil
	default:
		// This should not happen, the above should be exhaustive.
		return fmt.Errorf("unsupported NestingMode %s", schema.NestedType.Nesting.String())
	}
}

func writeConfigBlocksFromExisting(buf *strings.Builder, stateVal cty.Value, blocks map[string]*configschema.NestedBlock, indent int) error {
	if len(blocks) == 0 {
		return nil
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
		if err := writeConfigNestedBlockFromExisting(buf, name, blockS, blockVal, indent); err != nil {
			return err
		}
	}

	return nil
}

func writeConfigNestedTypeAttributeFromExisting(buf *strings.Builder, name string, schema *configschema.Attribute, stateVal cty.Value, indent int) error {
	switch schema.NestedType.Nesting {
	case configschema.NestingSingle:
		if schema.Sensitive || stateVal.IsMarked() {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = {} # sensitive\n", name))
			return nil
		}
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s = {\n", name))

		// This shouldn't happen in real usage; state always has all values (set
		// to null as needed), but it protects against panics in tests (and any
		// really weird and unlikely cases).
		if !stateVal.Type().HasAttribute(name) {
			return nil
		}
		nestedVal := stateVal.GetAttr(name)
		if err := writeConfigAttributesFromExisting(buf, nestedVal, schema.NestedType.Attributes, indent+2); err != nil {
			return err
		}
		buf.WriteString("}\n")
		return nil

	case configschema.NestingList, configschema.NestingSet:
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s = [", name))

		if schema.Sensitive || stateVal.IsMarked() {
			buf.WriteString("] # sensitive\n")
			return nil
		}

		buf.WriteString("\n")

		listVals := ctyCollectionValues(stateVal.GetAttr(name))
		for i := range listVals {
			buf.WriteString(strings.Repeat(" ", indent+2))

			// The entire element is marked.
			if listVals[i].IsMarked() {
				buf.WriteString("{}, # sensitive\n")
				continue
			}

			buf.WriteString("{\n")
			if err := writeConfigAttributesFromExisting(buf, listVals[i], schema.NestedType.Attributes, indent+4); err != nil {
				return err
			}
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString("},\n")
		}
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("]\n")
		return nil

	case configschema.NestingMap:
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s = {", name))

		if schema.Sensitive || stateVal.IsMarked() {
			buf.WriteString(" } # sensitive\n")
			return nil
		}

		buf.WriteString("\n")

		vals := stateVal.GetAttr(name).AsValueMap()
		keys := make([]string, 0, len(vals))
		for key := range vals {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString(fmt.Sprintf("%s = {", key))

			// This entire value is marked
			if vals[key].IsMarked() {
				buf.WriteString("} # sensitive\n")
				continue
			}

			buf.WriteString("\n")
			if err := writeConfigAttributesFromExisting(buf, vals[key], schema.NestedType.Attributes, indent+4); err != nil {
				return err
			}
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString("}\n")
		}
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("}\n")
		return nil

	default:
		// This should not happen, the above should be exhaustive.
		return fmt.Errorf("unsupported NestingMode %s", schema.NestedType.Nesting.String())
	}
}

func writeConfigNestedBlockFromExisting(buf *strings.Builder, name string, schema *configschema.NestedBlock, stateVal cty.Value, indent int) error {
	switch schema.Nesting {
	case configschema.NestingSingle, configschema.NestingGroup:
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s {", name))

		// If the entire value is marked, don't print any nested attributes
		if stateVal.IsMarked() {
			buf.WriteString("} # sensitive\n")
			return nil
		}
		buf.WriteString("\n")
		if err := writeConfigAttributesFromExisting(buf, stateVal, schema.Attributes, indent+2); err != nil {
			return err
		}
		if err := writeConfigBlocksFromExisting(buf, stateVal, schema.BlockTypes, indent+2); err != nil {
			return err
		}
		buf.WriteString("}\n")
		return nil
	case configschema.NestingList, configschema.NestingSet:
		if stateVal.IsMarked() {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s {} # sensitive\n", name))
			return nil
		}
		listVals := ctyCollectionValues(stateVal)
		for i := range listVals {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s {\n", name))
			if err := writeConfigAttributesFromExisting(buf, listVals[i], schema.Attributes, indent+2); err != nil {
				return err
			}
			if err := writeConfigBlocksFromExisting(buf, listVals[i], schema.BlockTypes, indent+2); err != nil {
				return err
			}
			buf.WriteString("}\n")
		}
		return nil
	case configschema.NestingMap:
		// If the entire value is marked, don't print any nested attributes
		if stateVal.IsMarked() {
			buf.WriteString(fmt.Sprintf("%s {} # sensitive\n", name))
			return nil
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
				return nil
			}
			buf.WriteString("\n")

			if err := writeConfigAttributesFromExisting(buf, vals[key], schema.Attributes, indent+2); err != nil {
				return err
			}
			if err := writeConfigBlocksFromExisting(buf, vals[key], schema.BlockTypes, indent+2); err != nil {
				return err
			}
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString("}\n")
		}
		return nil
	default:
		// This should not happen, the above should be exhaustive.
		return fmt.Errorf("unsupported NestingMode %s", schema.Nesting.String())
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
	return
}

func writeBlockTypeConstraint(buf *strings.Builder, schema *configschema.NestedBlock) {
	if schema.MinItems > 0 {
		buf.WriteString(" # REQUIRED block\n")
	} else {
		buf.WriteString(" # OPTIONAL block\n")
	}
	return
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
