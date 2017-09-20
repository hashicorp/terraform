package hclsyntax

import (
	"github.com/hashicorp/hcl2/hcl"
)

// ParseConfig parses the given buffer as a whole zcl config file, returning
// a *hcl.File representing its contents. If HasErrors called on the returned
// diagnostics returns true, the returned body is likely to be incomplete
// and should therefore be used with care.
//
// The body in the returned file has dynamic type *zclsyntax.Body, so callers
// may freely type-assert this to get access to the full zclsyntax API in
// situations where detailed access is required. However, most common use-cases
// should be served using the hcl.Body interface to ensure compatibility with
// other configurationg syntaxes, such as JSON.
func ParseConfig(src []byte, filename string, start hcl.Pos) (*hcl.File, hcl.Diagnostics) {
	tokens, diags := LexConfig(src, filename, start)
	peeker := newPeeker(tokens, false)
	parser := &parser{peeker: peeker}
	body, parseDiags := parser.ParseBody(TokenEOF)
	diags = append(diags, parseDiags...)
	return &hcl.File{
		Body:  body,
		Bytes: src,

		Nav: navigation{
			root: body,
		},
	}, diags
}

// ParseExpression parses the given buffer as a standalone zcl expression,
// returning it as an instance of Expression.
func ParseExpression(src []byte, filename string, start hcl.Pos) (Expression, hcl.Diagnostics) {
	tokens, diags := LexExpression(src, filename, start)
	peeker := newPeeker(tokens, false)
	parser := &parser{peeker: peeker}

	// Bare expressions are always parsed in  "ignore newlines" mode, as if
	// they were wrapped in parentheses.
	parser.PushIncludeNewlines(false)

	expr, parseDiags := parser.ParseExpression()
	diags = append(diags, parseDiags...)

	next := parser.Peek()
	if next.Type != TokenEOF && !parser.recovery {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Extra characters after expression",
			Detail:   "An expression was successfully parsed, but extra characters were found after it.",
			Subject:  &next.Range,
		})
	}

	return expr, diags
}

// ParseTemplate parses the given buffer as a standalone zcl template,
// returning it as an instance of Expression.
func ParseTemplate(src []byte, filename string, start hcl.Pos) (Expression, hcl.Diagnostics) {
	tokens, diags := LexTemplate(src, filename, start)
	peeker := newPeeker(tokens, false)
	parser := &parser{peeker: peeker}
	expr, parseDiags := parser.ParseTemplate()
	diags = append(diags, parseDiags...)
	return expr, diags
}

// ParseTraversalAbs parses the given buffer as a standalone absolute traversal.
//
// Parsing as a traversal is more limited than parsing as an expession since
// it allows only attribute and indexing operations on variables. Traverals
// are useful as a syntax for referring to objects without necessarily
// evaluating them.
func ParseTraversalAbs(src []byte, filename string, start hcl.Pos) (hcl.Traversal, hcl.Diagnostics) {
	tokens, diags := LexExpression(src, filename, start)
	peeker := newPeeker(tokens, false)
	parser := &parser{peeker: peeker}

	// Bare traverals are always parsed in  "ignore newlines" mode, as if
	// they were wrapped in parentheses.
	parser.PushIncludeNewlines(false)

	expr, parseDiags := parser.ParseTraversalAbs()
	diags = append(diags, parseDiags...)
	return expr, diags
}

// LexConfig performs lexical analysis on the given buffer, treating it as a
// whole zcl config file, and returns the resulting tokens.
//
// Only minimal validation is done during lexical analysis, so the returned
// diagnostics may include errors about lexical issues such as bad character
// encodings or unrecognized characters, but full parsing is required to
// detect _all_ syntax errors.
func LexConfig(src []byte, filename string, start hcl.Pos) (Tokens, hcl.Diagnostics) {
	tokens := scanTokens(src, filename, start, scanNormal)
	diags := checkInvalidTokens(tokens)
	return tokens, diags
}

// LexExpression performs lexical analysis on the given buffer, treating it as
// a standalone zcl expression, and returns the resulting tokens.
//
// Only minimal validation is done during lexical analysis, so the returned
// diagnostics may include errors about lexical issues such as bad character
// encodings or unrecognized characters, but full parsing is required to
// detect _all_ syntax errors.
func LexExpression(src []byte, filename string, start hcl.Pos) (Tokens, hcl.Diagnostics) {
	// This is actually just the same thing as LexConfig, since configs
	// and expressions lex in the same way.
	tokens := scanTokens(src, filename, start, scanNormal)
	diags := checkInvalidTokens(tokens)
	return tokens, diags
}

// LexTemplate performs lexical analysis on the given buffer, treating it as a
// standalone zcl template, and returns the resulting tokens.
//
// Only minimal validation is done during lexical analysis, so the returned
// diagnostics may include errors about lexical issues such as bad character
// encodings or unrecognized characters, but full parsing is required to
// detect _all_ syntax errors.
func LexTemplate(src []byte, filename string, start hcl.Pos) (Tokens, hcl.Diagnostics) {
	tokens := scanTokens(src, filename, start, scanTemplate)
	diags := checkInvalidTokens(tokens)
	return tokens, diags
}
