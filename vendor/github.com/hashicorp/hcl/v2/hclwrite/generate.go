package hclwrite

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// TokensForValue returns a sequence of tokens that represents the given
// constant value.
//
// This function only supports types that are used by HCL. In particular, it
// does not support capsule types and will panic if given one.
//
// It is not possible to express an unknown value in source code, so this
// function will panic if the given value is unknown or contains any unknown
// values. A caller can call the value's IsWhollyKnown method to verify that
// no unknown values are present before calling TokensForValue.
func TokensForValue(val cty.Value) Tokens {
	toks := appendTokensForValue(val, nil)
	format(toks) // fiddle with the SpacesBefore field to get canonical spacing
	return toks
}

// TokensForTraversal returns a sequence of tokens that represents the given
// traversal.
//
// If the traversal is absolute then the result is a self-contained, valid
// reference expression. If the traversal is relative then the returned tokens
// could be appended to some other expression tokens to traverse into the
// represented expression.
func TokensForTraversal(traversal hcl.Traversal) Tokens {
	toks := appendTokensForTraversal(traversal, nil)
	format(toks) // fiddle with the SpacesBefore field to get canonical spacing
	return toks
}

func appendTokensForValue(val cty.Value, toks Tokens) Tokens {
	switch {

	case !val.IsKnown():
		panic("cannot produce tokens for unknown value")

	case val.IsNull():
		toks = append(toks, &Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(`null`),
		})

	case val.Type() == cty.Bool:
		var src []byte
		if val.True() {
			src = []byte(`true`)
		} else {
			src = []byte(`false`)
		}
		toks = append(toks, &Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: src,
		})

	case val.Type() == cty.Number:
		bf := val.AsBigFloat()
		srcStr := bf.Text('f', -1)
		toks = append(toks, &Token{
			Type:  hclsyntax.TokenNumberLit,
			Bytes: []byte(srcStr),
		})

	case val.Type() == cty.String:
		// TODO: If it's a multi-line string ending in a newline, format
		// it as a HEREDOC instead.
		src := escapeQuotedStringLit(val.AsString())
		toks = append(toks, &Token{
			Type:  hclsyntax.TokenOQuote,
			Bytes: []byte{'"'},
		})
		if len(src) > 0 {
			toks = append(toks, &Token{
				Type:  hclsyntax.TokenQuotedLit,
				Bytes: src,
			})
		}
		toks = append(toks, &Token{
			Type:  hclsyntax.TokenCQuote,
			Bytes: []byte{'"'},
		})

	case val.Type().IsListType() || val.Type().IsSetType() || val.Type().IsTupleType():
		toks = append(toks, &Token{
			Type:  hclsyntax.TokenOBrack,
			Bytes: []byte{'['},
		})

		i := 0
		for it := val.ElementIterator(); it.Next(); {
			if i > 0 {
				toks = append(toks, &Token{
					Type:  hclsyntax.TokenComma,
					Bytes: []byte{','},
				})
			}
			_, eVal := it.Element()
			toks = appendTokensForValue(eVal, toks)
			i++
		}

		toks = append(toks, &Token{
			Type:  hclsyntax.TokenCBrack,
			Bytes: []byte{']'},
		})

	case val.Type().IsMapType() || val.Type().IsObjectType():
		toks = append(toks, &Token{
			Type:  hclsyntax.TokenOBrace,
			Bytes: []byte{'{'},
		})

		i := 0
		for it := val.ElementIterator(); it.Next(); {
			if i > 0 {
				toks = append(toks, &Token{
					Type:  hclsyntax.TokenComma,
					Bytes: []byte{','},
				})
			}
			eKey, eVal := it.Element()
			if hclsyntax.ValidIdentifier(eKey.AsString()) {
				toks = append(toks, &Token{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte(eKey.AsString()),
				})
			} else {
				toks = appendTokensForValue(eKey, toks)
			}
			toks = append(toks, &Token{
				Type:  hclsyntax.TokenEqual,
				Bytes: []byte{'='},
			})
			toks = appendTokensForValue(eVal, toks)
			i++
		}

		toks = append(toks, &Token{
			Type:  hclsyntax.TokenCBrace,
			Bytes: []byte{'}'},
		})

	default:
		panic(fmt.Sprintf("cannot produce tokens for %#v", val))
	}

	return toks
}

func appendTokensForTraversal(traversal hcl.Traversal, toks Tokens) Tokens {
	for _, step := range traversal {
		appendTokensForTraversalStep(step, toks)
	}
	return toks
}

func appendTokensForTraversalStep(step hcl.Traverser, toks Tokens) {
	switch ts := step.(type) {
	case hcl.TraverseRoot:
		toks = append(toks, &Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(ts.Name),
		})
	case hcl.TraverseAttr:
		toks = append(
			toks,
			&Token{
				Type:  hclsyntax.TokenDot,
				Bytes: []byte{'.'},
			},
			&Token{
				Type:  hclsyntax.TokenIdent,
				Bytes: []byte(ts.Name),
			},
		)
	case hcl.TraverseIndex:
		toks = append(toks, &Token{
			Type:  hclsyntax.TokenOBrack,
			Bytes: []byte{'['},
		})
		appendTokensForValue(ts.Key, toks)
		toks = append(toks, &Token{
			Type:  hclsyntax.TokenCBrack,
			Bytes: []byte{']'},
		})
	default:
		panic(fmt.Sprintf("unsupported traversal step type %T", step))
	}
}

func escapeQuotedStringLit(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	buf := make([]byte, 0, len(s))
	for i, r := range s {
		switch r {
		case '\n':
			buf = append(buf, '\\', 'n')
		case '\r':
			buf = append(buf, '\\', 'r')
		case '\t':
			buf = append(buf, '\\', 't')
		case '"':
			buf = append(buf, '\\', '"')
		case '\\':
			buf = append(buf, '\\', '\\')
		case '$', '%':
			buf = appendRune(buf, r)
			remain := s[i+1:]
			if len(remain) > 0 && remain[0] == '{' {
				// Double up our template introducer symbol to escape it.
				buf = appendRune(buf, r)
			}
		default:
			if !unicode.IsPrint(r) {
				var fmted string
				if r < 65536 {
					fmted = fmt.Sprintf("\\u%04x", r)
				} else {
					fmted = fmt.Sprintf("\\U%08x", r)
				}
				buf = append(buf, fmted...)
			} else {
				buf = appendRune(buf, r)
			}
		}
	}
	return buf
}

func appendRune(b []byte, r rune) []byte {
	l := utf8.RuneLen(r)
	for i := 0; i < l; i++ {
		b = append(b, 0) // make room at the end of our buffer
	}
	ch := b[len(b)-l:]
	utf8.EncodeRune(ch, r)
	return b
}
