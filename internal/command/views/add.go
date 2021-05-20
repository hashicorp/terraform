package views

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// Add is the view interface for the "terraform add" command.
type Add interface {
	Resource(addrs.AbsResourceInstance, *configschema.Block, string, cty.Value) error
	Diagnostics(tfdiags.Diagnostics)
}

// NewAdd returns an initialized Validate implementation for the given ViewType.
func NewAdd(vt arguments.ViewType, view *View, args *arguments.Add) Add {
	return &addHuman{
		view:     view,
		optional: args.Optional,
		outPath:  args.OutPath,
	}
}

type addHuman struct {
	view     *View
	optional bool
	outPath  string
}

func (v *addHuman) Resource(addr addrs.AbsResourceInstance, schema *configschema.Block, provider string, stateVal cty.Value) error {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("resource %q %q {\n", addr.Resource.Resource.Type, addr.Resource.Resource.Name))
	if provider != "" {
		buf.WriteString(strings.Repeat(" ", 2))
		buf.WriteString(fmt.Sprintf("provider = %s\n", provider))
	}

	if stateVal.RawEquals(cty.NilVal) {
		if err := v.writeConfigAttributes(&buf, schema.Attributes, 2); err != nil {
			return err
		}
		if err := v.writeConfigBlocks(&buf, schema.BlockTypes, 2); err != nil {
			return err
		}
	} else {
		if err := v.writeConfigAttributesFromExisting(&buf, stateVal, schema.Attributes, 2); err != nil {
			return err
		}
		if err := v.writeConfigBlocksFromExisting(&buf, stateVal, schema.BlockTypes, 2); err != nil {
			return err
		}
	}

	buf.WriteString("}")

	// The output better be valid HCL which can be parsed and formatted.
	formatted := hclwrite.Format([]byte(buf.String()))

	var err error
	if v.outPath == "" {
		_, err = v.view.streams.Println(string(formatted))
		return err
	} else {
		// The Println call above adds this newline automatically; we add it manually here.
		formatted = append(formatted, '\n')
		return os.WriteFile(v.outPath, formatted, 0600)
	}
}

func (v *addHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *addHuman) writeConfigAttributes(buf *strings.Builder, attrs map[string]*configschema.Attribute, indent int) error {
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
			if err := v.writeConfigNestedTypeAttribute(buf, name, attrS, indent); err != nil {
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
		} else if attrS.Optional && v.optional {
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

func (v *addHuman) writeConfigAttributesFromExisting(buf *strings.Builder, stateVal cty.Value, attrs map[string]*configschema.Attribute, indent int) error {
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
			if err := v.writeConfigNestedTypeAttributeFromExisting(buf, name, attrS, stateVal, indent); err != nil {
				return err
			}
			continue
		}
		if attrS.Required {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = ", name))

			var val cty.Value
			if stateVal.Type().HasAttribute(name) {
				val = stateVal.GetAttr(name)
			} else {
				val = attrS.EmptyValue()
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

		} else if attrS.Optional && v.optional {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = ", name))

			var val cty.Value
			if !stateVal.RawEquals(cty.NilVal) && stateVal.Type().HasAttribute(name) {
				val = stateVal.GetAttr(name)
			} else {
				val = attrS.EmptyValue()
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

func (v *addHuman) writeConfigBlocks(buf *strings.Builder, blocks map[string]*configschema.NestedBlock, indent int) error {
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

		if blockS.MinItems > 0 {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s {\n", name))
			if len(blockS.Attributes) > 0 {
				if err := v.writeConfigAttributes(buf, blockS.Attributes, indent+2); err != nil {
					return err
				}
			}
			if len(blockS.BlockTypes) > 0 {
				if err := v.writeConfigBlocks(buf, blockS.BlockTypes, indent+2); err != nil {
					return err
				}
			}
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString("}\n")
		}
	}
	return nil
}

func (v *addHuman) writeConfigNestedTypeAttribute(buf *strings.Builder, name string, schema *configschema.Attribute, indent int) error {
	if schema.Required == false && v.optional == false {
		return nil
	}

	buf.WriteString(strings.Repeat(" ", indent))
	buf.WriteString(fmt.Sprintf("%s = ", name))

	switch schema.NestedType.Nesting {
	case configschema.NestingSingle:
		buf.WriteString("{")
		writeAttrTypeConstraint(buf, schema)
		if err := v.writeConfigAttributes(buf, schema.NestedType.Attributes, indent+2); err != nil {
			return err
		}
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("}\n")
		return nil
	case configschema.NestingList, configschema.NestingSet:
		buf.WriteString("[{")
		writeAttrTypeConstraint(buf, schema)
		if err := v.writeConfigAttributes(buf, schema.NestedType.Attributes, indent+2); err != nil {
			return err
		}
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("}]\n")
		return nil
	case configschema.NestingMap:
		buf.WriteString("{")
		writeAttrTypeConstraint(buf, schema)
		buf.WriteString(strings.Repeat(" ", indent+2))
		buf.WriteString("key = {\n")
		if err := v.writeConfigAttributes(buf, schema.NestedType.Attributes, indent+4); err != nil {
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

func (v *addHuman) writeConfigBlocksFromExisting(buf *strings.Builder, stateVal cty.Value, blocks map[string]*configschema.NestedBlock, indent int) error {
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
		if err := v.writeConfigNestedBlockFromExisting(buf, name, blockS, blockVal, indent); err != nil {
			return err
		}
	}

	return nil
}

func (v *addHuman) writeConfigNestedTypeAttributeFromExisting(buf *strings.Builder, name string, schema *configschema.Attribute, stateVal cty.Value, indent int) error {
	switch schema.NestedType.Nesting {
	case configschema.NestingSingle:
		if schema.Sensitive {
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
		if err := v.writeConfigAttributesFromExisting(buf, nestedVal, schema.NestedType.Attributes, indent+2); err != nil {
			return err
		}
		buf.WriteString("}\n")
		return nil
	case configschema.NestingList, configschema.NestingSet:
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s = [\n", name))

		if schema.Sensitive || stateVal.IsMarked() {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString("] # sensitive\n")
			return nil
		}

		listVals := ctyCollectionValues(stateVal.GetAttr(name))
		for i := range listVals {
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString("{\n")
			if err := v.writeConfigAttributesFromExisting(buf, listVals[i], schema.NestedType.Attributes, indent+4); err != nil {
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
		buf.WriteString(fmt.Sprintf("%s = {\n", name))

		if schema.Sensitive || stateVal.IsMarked() {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString("} # sensitive\n")
			return nil
		}

		vals := stateVal.GetAttr(name).AsValueMap()
		keys := make([]string, 0, len(vals))
		for key := range vals {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString(fmt.Sprintf("%s = {\n", key))
			if err := v.writeConfigAttributesFromExisting(buf, vals[key], schema.NestedType.Attributes, indent+4); err != nil {
				return err
			}
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString("},\n")
		}
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("}\n")
		return nil
	default:
		// This should not happen, the above should be exhaustive.
		return fmt.Errorf("unsupported NestingMode %s", schema.NestedType.Nesting.String())
	}
}

func (v *addHuman) writeConfigNestedBlockFromExisting(buf *strings.Builder, name string, schema *configschema.NestedBlock, stateVal cty.Value, indent int) error {
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
		if err := v.writeConfigAttributesFromExisting(buf, stateVal, schema.Attributes, indent+2); err != nil {
			return err
		}
		if err := v.writeConfigBlocksFromExisting(buf, stateVal, schema.BlockTypes, indent+2); err != nil {
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
			if err := v.writeConfigAttributesFromExisting(buf, listVals[i], schema.Attributes, indent+2); err != nil {
				return err
			}
			if err := v.writeConfigBlocksFromExisting(buf, listVals[i], schema.BlockTypes, indent+2); err != nil {
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

			if err := v.writeConfigAttributesFromExisting(buf, vals[key], schema.Attributes, indent+2); err != nil {
				return err
			}
			if err := v.writeConfigBlocksFromExisting(buf, vals[key], schema.BlockTypes, indent+2); err != nil {
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
