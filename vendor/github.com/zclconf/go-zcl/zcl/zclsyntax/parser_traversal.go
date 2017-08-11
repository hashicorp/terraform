package zclsyntax

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-zcl/zcl"
)

// ParseTraversalAbs parses an absolute traversal that is assumed to consume
// all of the remaining tokens in the peeker. The usual parser recovery
// behavior is not supported here because traversals are not expected to
// be parsed as part of a larger program.
func (p *parser) ParseTraversalAbs() (zcl.Traversal, zcl.Diagnostics) {
	var ret zcl.Traversal
	var diags zcl.Diagnostics

	// Absolute traversal must always begin with a variable name
	varTok := p.Read()
	if varTok.Type != TokenIdent {
		diags = append(diags, &zcl.Diagnostic{
			Severity: zcl.DiagError,
			Summary:  "Variable name required",
			Detail:   "Must begin with a variable name.",
			Subject:  &varTok.Range,
		})
		return ret, diags
	}

	varName := string(varTok.Bytes)
	ret = append(ret, zcl.TraverseRoot{
		Name:     varName,
		SrcRange: varTok.Range,
	})

	for {
		next := p.Peek()

		if next.Type == TokenEOF {
			return ret, diags
		}

		switch next.Type {
		case TokenDot:
			// Attribute access
			dot := p.Read() // eat dot
			nameTok := p.Read()
			if nameTok.Type != TokenIdent {
				if nameTok.Type == TokenStar {
					diags = append(diags, &zcl.Diagnostic{
						Severity: zcl.DiagError,
						Summary:  "Attribute name required",
						Detail:   "Splat expressions (.*) may not be used here.",
						Subject:  &nameTok.Range,
						Context:  zcl.RangeBetween(varTok.Range, nameTok.Range).Ptr(),
					})
				} else {
					diags = append(diags, &zcl.Diagnostic{
						Severity: zcl.DiagError,
						Summary:  "Attribute name required",
						Detail:   "Dot must be followed by attribute name.",
						Subject:  &nameTok.Range,
						Context:  zcl.RangeBetween(varTok.Range, nameTok.Range).Ptr(),
					})
				}
				return ret, diags
			}

			attrName := string(nameTok.Bytes)
			ret = append(ret, zcl.TraverseAttr{
				Name:     attrName,
				SrcRange: zcl.RangeBetween(dot.Range, nameTok.Range),
			})
		case TokenOBrack:
			// Index
			open := p.Read() // eat open bracket
			next := p.Peek()

			switch next.Type {
			case TokenNumberLit:
				tok := p.Read() // eat number
				numVal, numDiags := p.numberLitValue(tok)
				diags = append(diags, numDiags...)

				close := p.Read()
				if close.Type != TokenCBrack {
					diags = append(diags, &zcl.Diagnostic{
						Severity: zcl.DiagError,
						Summary:  "Unclosed index brackets",
						Detail:   "Index key must be followed by a closing bracket.",
						Subject:  &close.Range,
						Context:  zcl.RangeBetween(open.Range, close.Range).Ptr(),
					})
				}

				ret = append(ret, zcl.TraverseIndex{
					Key:      numVal,
					SrcRange: zcl.RangeBetween(open.Range, close.Range),
				})

				if diags.HasErrors() {
					return ret, diags
				}

			case TokenOQuote:
				str, _, strDiags := p.parseQuotedStringLiteral()
				diags = append(diags, strDiags...)

				close := p.Read()
				if close.Type != TokenCBrack {
					diags = append(diags, &zcl.Diagnostic{
						Severity: zcl.DiagError,
						Summary:  "Unclosed index brackets",
						Detail:   "Index key must be followed by a closing bracket.",
						Subject:  &close.Range,
						Context:  zcl.RangeBetween(open.Range, close.Range).Ptr(),
					})
				}

				ret = append(ret, zcl.TraverseIndex{
					Key:      cty.StringVal(str),
					SrcRange: zcl.RangeBetween(open.Range, close.Range),
				})

				if diags.HasErrors() {
					return ret, diags
				}

			default:
				if next.Type == TokenStar {
					diags = append(diags, &zcl.Diagnostic{
						Severity: zcl.DiagError,
						Summary:  "Attribute name required",
						Detail:   "Splat expressions ([*]) may not be used here.",
						Subject:  &next.Range,
						Context:  zcl.RangeBetween(varTok.Range, next.Range).Ptr(),
					})
				} else {
					diags = append(diags, &zcl.Diagnostic{
						Severity: zcl.DiagError,
						Summary:  "Index value required",
						Detail:   "Index brackets must contain either a literal number or a literal string.",
						Subject:  &next.Range,
						Context:  zcl.RangeBetween(varTok.Range, next.Range).Ptr(),
					})
				}
				return ret, diags
			}

		default:
			diags = append(diags, &zcl.Diagnostic{
				Severity: zcl.DiagError,
				Summary:  "Invalid character",
				Detail:   "Expected an attribute access or an index operator.",
				Subject:  &next.Range,
				Context:  zcl.RangeBetween(varTok.Range, next.Range).Ptr(),
			})
			return ret, diags
		}
	}
}
