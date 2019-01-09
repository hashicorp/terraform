package tfconfig

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl2/hcl/hclsyntax"

	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func loadModule(dir string) (*Module, Diagnostics) {
	mod := newModule(dir)
	primaryPaths, diags := dirFiles(dir)

	parser := hclparse.NewParser()

	for _, filename := range primaryPaths {
		var file *hcl.File
		var fileDiags hcl.Diagnostics
		if strings.HasSuffix(filename, ".json") {
			file, fileDiags = parser.ParseJSONFile(filename)
		} else {
			file, fileDiags = parser.ParseHCLFile(filename)
		}
		diags = append(diags, fileDiags...)
		if file == nil {
			continue
		}

		content, _, contentDiags := file.Body.PartialContent(rootSchema)
		diags = append(diags, contentDiags...)

		for _, block := range content.Blocks {
			switch block.Type {

			case "terraform":
				content, _, contentDiags := block.Body.PartialContent(terraformBlockSchema)
				diags = append(diags, contentDiags...)

				if attr, defined := content.Attributes["required_version"]; defined {
					var version string
					valDiags := gohcl.DecodeExpression(attr.Expr, nil, &version)
					diags = append(diags, valDiags...)
					if !valDiags.HasErrors() {
						mod.RequiredCore = append(mod.RequiredCore, version)
					}
				}

				for _, block := range content.Blocks {
					// Our schema only allows required_providers here, so we
					// assume that we'll only get that block type.
					attrs, attrDiags := block.Body.JustAttributes()
					diags = append(diags, attrDiags...)

					for name, attr := range attrs {
						var version string
						valDiags := gohcl.DecodeExpression(attr.Expr, nil, &version)
						diags = append(diags, valDiags...)
						if !valDiags.HasErrors() {
							mod.RequiredProviders[name] = append(mod.RequiredProviders[name], version)
						}
					}
				}

			case "variable":
				content, _, contentDiags := block.Body.PartialContent(variableSchema)
				diags = append(diags, contentDiags...)

				name := block.Labels[0]
				v := &Variable{
					Name: name,
					Pos:  sourcePosHCL(block.DefRange),
				}

				mod.Variables[name] = v

				if attr, defined := content.Attributes["type"]; defined {
					// We handle this particular attribute in a somewhat-tricky way:
					// since Terraform may evolve its type expression syntax in
					// future versions, we don't want to be overly-strict in how
					// we handle it here, and so we'll instead just take the raw
					// source provided by the user, using the source location
					// information in the expression object.
					//
					// However, older versions of Terraform expected the type
					// to be a string containing a keyword, so we'll need to
					// handle that as a special case first for backward compatibility.

					var typeExpr string

					var typeExprAsStr string
					valDiags := gohcl.DecodeExpression(attr.Expr, nil, &typeExprAsStr)
					if !valDiags.HasErrors() {
						typeExpr = typeExprAsStr
					} else {

						rng := attr.Expr.Range()
						sourceFilename := rng.Filename
						source, exists := parser.Sources()[sourceFilename]
						if exists {
							typeExpr = string(rng.SliceBytes(source))
						} else {
							// This should never happen, so we'll just warn about it and leave the type unspecified.
							diags = append(diags, &hcl.Diagnostic{
								Severity: hcl.DiagError,
								Summary:  "Source code not available",
								Detail:   fmt.Sprintf("Source code is not available for the file %q, which declares the variable %q.", sourceFilename, name),
								Subject:  &block.DefRange,
							})
							typeExpr = ""
						}

					}

					v.Type = typeExpr
				}

				if attr, defined := content.Attributes["description"]; defined {
					var description string
					valDiags := gohcl.DecodeExpression(attr.Expr, nil, &description)
					diags = append(diags, valDiags...)
					v.Description = description
				}

				if attr, defined := content.Attributes["default"]; defined {
					// To avoid the caller needing to deal with cty here, we'll
					// use its JSON encoding to convert into an
					// approximately-equivalent plain Go interface{} value
					// to return.
					val, valDiags := attr.Expr.Value(nil)
					diags = append(diags, valDiags...)
					if val.IsWhollyKnown() { // should only be false if there are errors in the input
						valJSON, err := ctyjson.Marshal(val, val.Type())
						if err != nil {
							// Should never happen, since all possible known
							// values have a JSON mapping.
							panic(fmt.Errorf("failed to serialize default value as JSON: %s", err))
						}
						var def interface{}
						err = json.Unmarshal(valJSON, &def)
						if err != nil {
							// Again should never happen, because valJSON is
							// guaranteed valid by ctyjson.Marshal.
							panic(fmt.Errorf("failed to re-parse default value from JSON: %s", err))
						}
						v.Default = def
					}
				}

			case "output":

				content, _, contentDiags := block.Body.PartialContent(outputSchema)
				diags = append(diags, contentDiags...)

				name := block.Labels[0]
				o := &Output{
					Name: name,
					Pos:  sourcePosHCL(block.DefRange),
				}

				mod.Outputs[name] = o

				if attr, defined := content.Attributes["description"]; defined {
					var description string
					valDiags := gohcl.DecodeExpression(attr.Expr, nil, &description)
					diags = append(diags, valDiags...)
					o.Description = description
				}

			case "provider":

				content, _, contentDiags := block.Body.PartialContent(providerConfigSchema)
				diags = append(diags, contentDiags...)

				name := block.Labels[0]

				if attr, defined := content.Attributes["version"]; defined {
					var version string
					valDiags := gohcl.DecodeExpression(attr.Expr, nil, &version)
					diags = append(diags, valDiags...)
					if !valDiags.HasErrors() {
						mod.RequiredProviders[name] = append(mod.RequiredProviders[name], version)
					}
				}

				// Even if there wasn't an explicit version required, we still
				// need an entry in our map to signal the unversioned dependency.
				if _, exists := mod.RequiredProviders[name]; !exists {
					mod.RequiredProviders[name] = []string{}
				}

			case "resource", "data":

				content, _, contentDiags := block.Body.PartialContent(resourceSchema)
				diags = append(diags, contentDiags...)

				typeName := block.Labels[0]
				name := block.Labels[1]

				r := &Resource{
					Type: typeName,
					Name: name,
					Pos:  sourcePosHCL(block.DefRange),
				}

				var resourcesMap map[string]*Resource

				switch block.Type {
				case "resource":
					r.Mode = ManagedResourceMode
					resourcesMap = mod.ManagedResources
				case "data":
					r.Mode = DataResourceMode
					resourcesMap = mod.DataResources
				}

				key := r.MapKey()

				resourcesMap[key] = r

				if attr, defined := content.Attributes["provider"]; defined {
					// New style here is to provide this as a naked traversal
					// expression, but we also support quoted references for
					// older configurations that predated this convention.
					traversal, travDiags := hcl.AbsTraversalForExpr(attr.Expr)
					if travDiags.HasErrors() {
						traversal = nil // in case we got any partial results

						// Fall back on trying to parse as a string
						var travStr string
						valDiags := gohcl.DecodeExpression(attr.Expr, nil, &travStr)
						if !valDiags.HasErrors() {
							var strDiags hcl.Diagnostics
							traversal, strDiags = hclsyntax.ParseTraversalAbs([]byte(travStr), "", hcl.Pos{})
							if strDiags.HasErrors() {
								traversal = nil
							}
						}
					}

					// If we get out here with a nil traversal then we didn't
					// succeed in processing the input.
					if len(traversal) > 0 {
						providerName := traversal.RootName()
						alias := ""
						if len(traversal) > 1 {
							if getAttr, ok := traversal[1].(hcl.TraverseAttr); ok {
								alias = getAttr.Name
							}
						}
						r.Provider = ProviderRef{
							Name:  providerName,
							Alias: alias,
						}
					} else {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid provider reference",
							Detail:   "Provider argument requires a provider name followed by an optional alias, like \"aws.foo\".",
							Subject:  attr.Expr.Range().Ptr(),
						})
					}
				} else {
					// If provider _isn't_ set then we'll infer it from the
					// resource type.
					r.Provider = ProviderRef{
						Name: resourceTypeDefaultProviderName(r.Type),
					}
				}

			case "module":

				content, _, contentDiags := block.Body.PartialContent(moduleCallSchema)
				diags = append(diags, contentDiags...)

				name := block.Labels[0]
				mc := &ModuleCall{
					Name: block.Labels[0],
					Pos:  sourcePosHCL(block.DefRange),
				}

				mod.ModuleCalls[name] = mc

				if attr, defined := content.Attributes["source"]; defined {
					var source string
					valDiags := gohcl.DecodeExpression(attr.Expr, nil, &source)
					diags = append(diags, valDiags...)
					mc.Source = source
				}

				if attr, defined := content.Attributes["version"]; defined {
					var version string
					valDiags := gohcl.DecodeExpression(attr.Expr, nil, &version)
					diags = append(diags, valDiags...)
					mc.Version = version
				}

			default:
				// Should never happen because our cases above should be
				// exhaustive for our schema.
				panic(fmt.Errorf("unhandled block type %q", block.Type))
			}
		}
	}

	return mod, diagnosticsHCL(diags)
}
