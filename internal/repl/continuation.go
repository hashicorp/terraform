// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package repl

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ExpressionEntryCouldContinue is a helper for terraform console's interactive
// mode which serves as a heuristic for whether it seems like the author might
// be trying to split an expression over multiple lines of input.
//
// The current heuristic is whether there's at least one bracketing delimiter
// that isn't closed, but only if any closing brackets already present are
// properly balanced.
//
// This function also always returns false if the last line entered is empty,
// because that seems likely to represent a user trying to force Terraform to
// accept something that didn't pass the heuristic for some reason, at which
// point Terraform can try to evaluate the expression and return an error if
// it's invalid syntax.
func ExpressionEntryCouldContinue(linesSoFar []string) bool {
	if len(linesSoFar) == 0 || strings.TrimSpace(linesSoFar[len(linesSoFar)-1]) == "" {
		// If there's no input at all or if the last line is empty other than
		// spaces, we assume the user is trying to force Terraform to evaluate
		// what they entered so far without any further continuation.
		return false
	}

	// We use capacity 8 here as a compromise assuming that most reasonable
	// input entered at the console prompt will not use more than eight
	// levels of nesting, but even if it does then we'll just reallocate the
	// slice and so it's not a big deal.
	delimStack := make([]hclsyntax.TokenType, 0, 8)
	push := func(typ hclsyntax.TokenType) {
		delimStack = append(delimStack, typ)
	}
	pop := func() hclsyntax.TokenType {
		if len(delimStack) == 0 {
			return hclsyntax.TokenInvalid
		}
		ret := delimStack[len(delimStack)-1]
		delimStack = delimStack[:len(delimStack)-1]
		return ret
	}
	// We need to scan this all as one string because the HCL lexer has a few
	// special cases where it tracks open/close state itself, such as in heredocs.
	all := strings.Join(linesSoFar, "\n") + "\n"
	toks, diags := hclsyntax.LexExpression([]byte(all), "", hcl.InitialPos)
	if diags.HasErrors() {
		return false // bail early if the input is already invalid
	}
	for _, tok := range toks {
		switch tok.Type {
		case hclsyntax.TokenOBrace, hclsyntax.TokenOBrack, hclsyntax.TokenOParen, hclsyntax.TokenOHeredoc, hclsyntax.TokenTemplateInterp, hclsyntax.TokenTemplateControl:
			// Opening delimiters go on our stack so that we can hopefully
			// match them with closing delimiters later.
			push(tok.Type)
		case hclsyntax.TokenCBrace:
			open := pop()
			if open != hclsyntax.TokenOBrace {
				return false
			}
		case hclsyntax.TokenCBrack:
			open := pop()
			if open != hclsyntax.TokenOBrack {
				return false
			}
		case hclsyntax.TokenCParen:
			open := pop()
			if open != hclsyntax.TokenOParen {
				return false
			}
		case hclsyntax.TokenCHeredoc:
			open := pop()
			if open != hclsyntax.TokenOHeredoc {
				return false
			}
		case hclsyntax.TokenTemplateSeqEnd:
			open := pop()
			if open != hclsyntax.TokenTemplateInterp && open != hclsyntax.TokenTemplateControl {
				return false
			}
		}
	}

	// If we get here without returning early then all of the closing delimeters
	// were matched by opening delimiters. If our stack still contains at least
	// one opening bracket then it seems like the user is intending to type
	// more.
	return len(delimStack) != 0
}
