// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	hcljson "github.com/hashicorp/hcl/v2/json"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

const initialLanguageEdition = "TFStack2023"

// File represents the content of a single .tfcomponent.hcl or .tfcomponent.json
// file before it's been merged with its siblings in the same directory to
// produce the overall [Stack] object.
type File struct {
	// SourceAddr is the source location for this particular file, meaning
	// that the "sub-path" portion of the address should always be populated
	// and refer to a particular file rather than to a directory.
	SourceAddr sourceaddrs.FinalSource

	Declarations
}

// DecodeFileBody takes a body that is assumed to represent the root of a
// .tfcomponent.hcl or .tfcomponent.json file and decodes the declarations
// inside.
//
// If you have a []byte containing source code then consider using [ParseFile]
// instead, which parses the source code and then delegates to this function.
//
// This is exported for unusual situations where it's useful to analyze just
// a single file in isolation, without considering its context in a
// configuration tree. Some fields of the objects representing declarations in
// the configuration will be unpopulated when loading through this entry point.
// Prefer [LoadConfigDir] in most cases.
func DecodeFileBody(body hcl.Body, fileAddr sourceaddrs.FinalSource) (*File, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &File{
		SourceAddr:   fileAddr,
		Declarations: makeDeclarations(),
	}

	content, hclDiags := body.Content(rootConfigSchema)
	diags = diags.Append(hclDiags)
	if content == nil {
		return ret, diags
	}
	// Even if there are some errors we'll still try to analyze a partial
	// result, in case it allows us to give the user more context to work
	// with when resolving the errors detected so far.

	if langAttr, ok := content.Attributes["language"]; ok {
		// For now there is only one edition of the language and so we'll just
		// reject anything other than the current version. If we add other
		// editions later then we'll probably need to move the check for this
		// up into LoadSingleStackConfig so we can make sure that all of the
		// files in a directory agree on a language edition to use.
		editionKW := hcl.ExprAsKeyword(langAttr.Expr)
		if editionKW != initialLanguageEdition {
			var extra string
			if strings.HasPrefix(editionKW, "TFStack") {
				extra = "\n\nThis stack configuration might be intended for a newer version of Terraform."
			}
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid language edition",
				Detail: fmt.Sprintf(
					"If you declare an explicit language edition then it must currently be the keyword %s, because no other editions are supported.%s",
					initialLanguageEdition, extra,
				),
			})
			// We'll halt processing here if it's not for our current edition,
			// because we'll probably encounter language features from whatever
			// later language edition this config was written for.
			return ret, diags
		}
	}

	for _, block := range content.Blocks {
		switch block.Type {

		case "component":
			decl, moreDiags := decodeComponentBlock(block)
			diags = diags.Append(moreDiags)
			diags = diags.Append(
				ret.Declarations.addComponent(decl),
			)

		case "stack":
			decl, moreDiags := decodeEmbeddedStackBlock(block)
			diags = diags.Append(moreDiags)
			diags = diags.Append(
				ret.Declarations.addEmbeddedStack(decl),
			)

		case "variable":
			decl, moreDiags := decodeInputVariableBlock(block)
			diags = diags.Append(moreDiags)
			diags = diags.Append(
				ret.Declarations.addInputVariable(decl),
			)

		case "locals":
			decls, moreDiags := decodeLocalValuesBlock(block)
			diags = diags.Append(moreDiags)
			for _, decl := range decls {
				diags = diags.Append(
					ret.Declarations.addLocalValue(decl),
				)
			}

		case "output":
			decl, moreDiags := decodeOutputValueBlock(block)
			diags = diags.Append(moreDiags)
			diags = diags.Append(
				ret.Declarations.addOutputValue(decl),
			)

		case "provider":
			decl, moreDiags := decodeProviderConfigBlock(block)
			diags = diags.Append(moreDiags)
			diags = diags.Append(
				ret.Declarations.addProviderConfig(decl),
			)

		case "required_providers":
			decl, moreDiags := decodeProviderRequirementsBlock(block)
			diags = diags.Append(moreDiags)
			diags = diags.Append(
				ret.Declarations.addRequiredProviders(decl),
			)

		case "removed":
			decl, moreDiags := decodeRemovedBlock(block)
			diags = diags.Append(moreDiags)
			diags = diags.Append(
				ret.Declarations.addRemoved(decl),
			)

		default:
			// Should not get here because the cases above should be exhaustive
			// for everything declared in rootConfigSchema.
			panic(fmt.Sprintf("unhandled block type %q", block.Type))
		}
	}

	return ret, diags
}

// ParseFileSource parses the given source code as the content of a Stacks
// component file and then delegates the result to [DecodeFileBody] for
// analysis, returning that final result.
//
// For now, we support both the .tfstack. and .tfcomponent. file suffixes while
// the former is still in the process of being deprecated. It is not valid for
// a single stack to mix and match the two options. We are, therefore, accepting
// a "prior suffix" argument which lets us know which suffix was used by other
// files in this stack.
//
// ParseFileSource chooses between native vs. JSON syntax based on the suffix
// of the filename in the given source address, which must be either
// ".[tfstack|tfcomponent].hcl" or ".[tfstack|tfcomponent].json".
func ParseFileSource(src []byte, suffix string, fileAddr sourceaddrs.FinalSource) (*File, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	filename := sourceaddrs.FinalSourceFilename(fileAddr)

	var body hcl.Body
	switch {
	case strings.HasSuffix(filename, fmt.Sprintf(".%s.hcl", suffix)):
		hclFile, hclDiags := hclsyntax.ParseConfig(src, fileAddr.String(), hcl.InitialPos)
		diags = diags.Append(hclDiags)
		if diags.HasErrors() {
			return nil, diags
		}
		body = hclFile.Body
	case strings.HasSuffix(filename, fmt.Sprintf(".%s.json", suffix)):
		hclFile, hclDiags := hcljson.Parse(src, fileAddr.String())
		diags = diags.Append(hclDiags)
		if diags.HasErrors() {
			return nil, diags
		}
		body = hclFile.Body
	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported file type",
			fmt.Sprintf(
				"Cannot load %s as a stack configuration file: filename must have either a .%s.hcl or .%s.json suffix.",
				fileAddr, suffix, suffix,
			),
		))
		return nil, diags
	}

	ret, moreDiags := DecodeFileBody(body, fileAddr)
	diags = diags.Append(moreDiags)
	return ret, diags
}

// validFilenameSuffix returns ".tfcomponent.hcl" or ".tfcomponent.json" if the
// given filename ends with that suffix, and otherwise returns an empty
// string to indicate that the suffix was invalid.
//
// We still support the deprecated .tfstack suffix for the time being.
func validFilenameSuffix(filename string) string {
	const nativeSuffix = ".tfcomponent.hcl"
	const jsonSuffix = ".tfcomponent.json"
	const deprecatedNativeSuffix = ".tfstack.hcl"
	const deprecatedJsonSuffix = ".tfstack.json"

	switch {
	case strings.HasSuffix(filename, deprecatedNativeSuffix):
		return "tfstack"
	case strings.HasSuffix(filename, deprecatedJsonSuffix):
		return "tfstack"
	case strings.HasSuffix(filename, nativeSuffix):
		return "tfcomponent"
	case strings.HasSuffix(filename, jsonSuffix):
		return "tfcomponent"
	default:
		return ""
	}
}

var rootConfigSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "language"},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "stack", LabelNames: []string{"name"}},
		{Type: "component", LabelNames: []string{"name"}},
		{Type: "variable", LabelNames: []string{"name"}},
		{Type: "locals"},
		{Type: "output", LabelNames: []string{"name"}},
		{Type: "provider", LabelNames: []string{"type", "name"}},
		{Type: "required_providers"},
		{Type: "removed"},
	},
}
