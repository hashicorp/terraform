// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package genconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
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

// ImportGroup represents one or more resource and import configuration blocks.
type ImportGroup struct {
	Imports []ResourceImport
}

// ResourceImport pairs up the import and associated resource when generating
// configuration, so that query output can be more structured for easier
// consumption.
type ResourceImport struct {
	ImportBody []byte
	Resource   Resource
}

type Resource struct {
	Addr addrs.AbsResourceInstance

	// HCL Body of the resource, which is the attributes and blocks
	// that are part of the resource.
	Body []byte
}

func (r Resource) String() string {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("resource %q %q {\n", r.Addr.Resource.Resource.Type, r.Addr.Resource.Resource.Name))
	buf.Write(r.Body)
	buf.WriteString("}")

	formatted := hclwrite.Format([]byte(buf.String()))
	return string(formatted)
}

func (i ImportGroup) String() string {
	var buf strings.Builder

	for _, imp := range i.Imports {
		buf.WriteString(imp.Resource.String())
		buf.WriteString("\n\n")
		buf.WriteString(string(imp.ImportBody))
		buf.WriteString("\n\n")
	}

	// The output better be valid HCL which can be parsed and formatted.
	formatted := hclwrite.Format([]byte(buf.String()))
	return string(formatted)
}

func (i ImportGroup) ResourcesString() string {
	var buf strings.Builder

	for _, imp := range i.Imports {
		buf.WriteString(imp.Resource.String())
		buf.WriteString("\n")
	}

	// The output better be valid HCL which can be parsed and formatted.
	formatted := hclwrite.Format([]byte(buf.String()))
	return string(formatted)
}

func (i ImportGroup) ImportsString() string {
	var buf strings.Builder

	for _, imp := range i.Imports {
		buf.WriteString(string(imp.ImportBody))
		buf.WriteString("\n")
	}

	// The output better be valid HCL which can be parsed and formatted.
	formatted := hclwrite.Format([]byte(buf.String()))
	return string(formatted)
}

// GenerateResourceContents generates HCL configuration code for the provided
// resource and state value.
//
// If you want to generate actual valid Terraform code you should follow this
// call up with a call to WrapResourceContents, which will place a Terraform
// resource header around the attributes and blocks returned by this function.
func GenerateResourceContents(addr addrs.AbsResourceInstance,
	schema *configschema.Block,
	pc addrs.LocalProviderConfig,
	configVal cty.Value,
	forceProviderAddr bool,
) (Resource, tfdiags.Diagnostics) {
	var buf strings.Builder

	var diags tfdiags.Diagnostics

	generateProviderAddr := pc.LocalName != addr.Resource.Resource.ImpliedProvider() || pc.Alias != ""

	if generateProviderAddr || forceProviderAddr {
		buf.WriteString(strings.Repeat(" ", 2))
		buf.WriteString(fmt.Sprintf("provider = %s\n", pc.StringCompact()))
	}

	if configVal.RawEquals(cty.NilVal) {
		diags = diags.Append(writeConfigAttributes(addr, &buf, schema.Attributes, 2))
		diags = diags.Append(writeConfigBlocks(addr, &buf, schema.BlockTypes, 2))
	} else {
		diags = diags.Append(writeConfigAttributesFromExisting(addr, &buf, configVal, schema.Attributes, 2))
		diags = diags.Append(writeConfigBlocksFromExisting(addr, &buf, configVal, schema.BlockTypes, 2))
	}

	// The output better be valid HCL which can be parsed and formatted.
	formatted := hclwrite.Format([]byte(buf.String()))
	return Resource{Addr: addr, Body: formatted}, diags
}

// ResourceListElement is a single Resource state and identity pair derived from
// a list resource response.
type ResourceListElement struct {
	// Config is the cty value extracted from the resource state which is
	// intended to be written into the HCL resource block.
	Config cty.Value

	Identity cty.Value

	// ExpansionEnum is a unique enumeration of the list resource address relative to its expanded siblings.
	ExpansionEnum int
}

func GenerateListResourceContents(addr addrs.AbsResourceInstance,
	schema *configschema.Block,
	idSchema *configschema.Object,
	pc addrs.LocalProviderConfig,
	resources []ResourceListElement,
) (ImportGroup, tfdiags.Diagnostics) {

	var diags tfdiags.Diagnostics
	ret := ImportGroup{}

	for idx, res := range resources {
		// Generate a unique resource name for each instance in the list.
		resAddr := addrs.AbsResourceInstance{
			Module: addr.Module,
			Resource: addrs.ResourceInstance{
				Resource: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: addr.Resource.Resource.Type,
				},
			},
		}

		// If the list resource instance is keyed, the expansion counter is included in the address
		// to ensure uniqueness across the entire configuration.
		if addr.Resource.Key == addrs.NoKey {
			resAddr.Resource.Resource.Name = fmt.Sprintf("%s_%d", addr.Resource.Resource.Name, idx)
		} else {
			resAddr.Resource.Resource.Name = fmt.Sprintf("%s_%d_%d", addr.Resource.Resource.Name, res.ExpansionEnum, idx)
		}

		content, gDiags := GenerateResourceContents(resAddr, schema, pc, res.Config, true)
		if gDiags.HasErrors() {
			diags = diags.Append(gDiags)
			continue
		}

		resImport := ResourceImport{
			Resource: Resource{
				Addr: resAddr,
				Body: content.Body,
			},
		}

		importContent, gDiags := GenerateImportBlock(resAddr, idSchema, pc, res.Identity)
		if gDiags.HasErrors() {
			diags = diags.Append(gDiags)
			continue
		}

		resImport.ImportBody = bytes.TrimSpace(hclwrite.Format(importContent.ImportBody))
		ret.Imports = append(ret.Imports, resImport)
	}

	return ret, diags
}

func GenerateImportBlock(addr addrs.AbsResourceInstance, idSchema *configschema.Object, pc addrs.LocalProviderConfig, identity cty.Value) (ResourceImport, tfdiags.Diagnostics) {
	var buf strings.Builder
	var diags tfdiags.Diagnostics

	buf.WriteString("\n")
	buf.WriteString("import {\n")
	buf.WriteString(fmt.Sprintf("  to = %s\n", addr.String()))
	buf.WriteString(fmt.Sprintf("  provider = %s\n", pc.StringCompact()))
	buf.WriteString("  identity = {\n")
	diags = diags.Append(writeConfigAttributesFromExisting(addr, &buf, identity, idSchema.Attributes, 2))
	buf.WriteString(strings.Repeat(" ", 2))
	buf.WriteString("}\n}\n")

	formatted := hclwrite.Format([]byte(buf.String()))
	return ResourceImport{ImportBody: formatted}, diags
}

func writeConfigAttributes(addr addrs.AbsResourceInstance, buf *strings.Builder, attrs map[string]*configschema.Attribute, indent int) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if len(attrs) == 0 {
		return diags
	}

	// Get a list of sorted attribute names so the output will be consistent between runs.
	for _, name := range slices.Sorted(maps.Keys(attrs)) {
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

	// Sort attribute names so the output will be consistent between runs.
	for _, name := range slices.Sorted(maps.Keys(attrs)) {
		attrS := attrs[name]

		var val cty.Value
		if !stateVal.IsNull() && stateVal.Type().HasAttribute(name) {
			val = stateVal.GetAttr(name)
		} else {
			val = attrS.EmptyValue()
		}

		if attrS.Computed && val.IsNull() {
			// Computed attributes should never be written in the config. These
			// will be filtered out of the given cty value if they are not also
			// optional, and we want to skip writing `null` in the config.
			continue
		}

		if attrS.Deprecated {
			// We also want to skip showing deprecated attributes as null in the HCL.
			continue
		}

		if attrS.NestedType != nil {
			writeConfigNestedTypeAttributeFromExisting(addr, buf, name, attrS, stateVal, indent)
			continue
		}

		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s = ", name))

		if attrS.Sensitive {
			buf.WriteString("null # sensitive")
		} else {
			// If the value is a string storing a JSON value we want to represent it in a terraform native way
			// and encapsulate it in `jsonencode` as it is the idiomatic representation
			if !val.IsNull() && val.Type() == cty.String && json.Valid([]byte(val.AsString())) {
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
	for _, name := range slices.Sorted(maps.Keys(blocks)) {
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

	// Sort block names so the output will be consistent between runs.
	for _, name := range slices.Sorted(maps.Keys(blocks)) {
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
		if schema.Sensitive {
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

		if schema.Sensitive {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = [] # sensitive\n", name))
			return diags
		}

		vals := stateVal.GetAttr(name)
		if vals.IsNull() {
			// There is a difference between an empty list and a null list
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = null\n", name))
			return diags
		}

		listVals := vals.AsValueSlice()

		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s = [\n", name))
		for i := range listVals {
			buf.WriteString(strings.Repeat(" ", indent+2))

			buf.WriteString("{\n")
			diags = diags.Append(writeConfigAttributesFromExisting(addr, buf, listVals[i], schema.NestedType.Attributes, indent+4))
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString("},\n")
		}
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString("]\n")
		return diags

	case configschema.NestingMap:
		if schema.Sensitive {
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

		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s = {\n", name))
		for _, key := range slices.Sorted(maps.Keys(vals)) {
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString(fmt.Sprintf("%s = {", hclEscapeString(key)))

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
	if stateVal.IsNull() {
		return diags
	}

	switch schema.Nesting {
	case configschema.NestingSingle, configschema.NestingGroup:
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(fmt.Sprintf("%s {", name))

		buf.WriteString("\n")
		diags = diags.Append(writeConfigAttributesFromExisting(addr, buf, stateVal, schema.Attributes, indent+2))
		diags = diags.Append(writeConfigBlocksFromExisting(addr, buf, stateVal, schema.BlockTypes, indent+2))
		buf.WriteString("}\n")
		return diags
	case configschema.NestingList, configschema.NestingSet:
		listVals := stateVal.AsValueSlice()
		for i := range listVals {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s {\n", name))
			diags = diags.Append(writeConfigAttributesFromExisting(addr, buf, listVals[i], schema.Attributes, indent+2))
			diags = diags.Append(writeConfigBlocksFromExisting(addr, buf, listVals[i], schema.BlockTypes, indent+2))
			buf.WriteString("}\n")
		}
		return diags
	case configschema.NestingMap:
		vals := stateVal.AsValueMap()
		for _, key := range slices.Sorted(maps.Keys(vals)) {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s %q {", name, key))
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

// ExtractLegacyConfigFromState takes the state value of a resource, and filters the
// value down to what would be acceptable as a resource configuration value.
// This is used when the provider does not implement GenerateResourceConfig to
// create a suitable value.
func ExtractLegacyConfigFromState(schema *configschema.Block, state cty.Value) cty.Value {
	config, _ := cty.Transform(state, func(path cty.Path, v cty.Value) (cty.Value, error) {
		if v.IsNull() {
			return v, nil
		}

		if len(path) == 0 {
			return v, nil
		}

		ty := v.Type()
		null := cty.NullVal(ty)

		// find the attribute or block schema representing the value
		attr := schema.AttributeByPath(path)
		block := schema.BlockByPath(path)
		switch {
		case attr != nil:
			// deprecated attributes
			if attr.Deprecated {
				return null, nil
			}

			// read-only attributes are not written in the configuration
			if attr.Computed && !attr.Optional {
				return null, nil
			}

			// The legacy SDK adds an Optional+Computed "id" attribute to the
			// resource schema even if not defined in provider code.
			// During validation, however, the presence of an extraneous "id"
			// attribute in config will cause an error.
			// Remove this attribute so we do not generate an "id" attribute
			// where there is a risk that it is not in the real resource schema.
			if path.Equals(cty.GetAttrPath("id")) && attr.Computed && attr.Optional {
				return null, nil
			}

			// If we have "" for an optional value, assume it is actually null
			// due to the legacy SDK.
			if ty == cty.String {
				if !v.IsNull() && attr.Optional && len(v.AsString()) == 0 {
					return null, nil
				}
			}
			return v, nil

		case block != nil:
			if block.Deprecated {
				return null, nil
			}
		}

		// We're only filtering out values which correspond to specific
		// attributes or blocks from the schema, anything else is passed through
		// as it will be a leaf value within a container.
		return v, nil
	})

	return config
}
