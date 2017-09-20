package json

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hashicorp/hcl2/hcl"
)

func parseFileContent(buf []byte, filename string) (node, hcl.Diagnostics) {
	tokens := scan(buf, pos{
		Filename: filename,
		Pos: hcl.Pos{
			Byte:   0,
			Line:   1,
			Column: 1,
		},
	})
	p := newPeeker(tokens)
	node, diags := parseValue(p)
	if len(diags) == 0 && p.Peek().Type != tokenEOF {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Extraneous data after value",
			Detail:   "Extra characters appear after the JSON value.",
			Subject:  p.Peek().Range.Ptr(),
		})
	}
	return node, diags
}

func parseValue(p *peeker) (node, hcl.Diagnostics) {
	tok := p.Peek()

	wrapInvalid := func(n node, diags hcl.Diagnostics) (node, hcl.Diagnostics) {
		if n != nil {
			return n, diags
		}
		return invalidVal{tok.Range}, diags
	}

	switch tok.Type {
	case tokenBraceO:
		return wrapInvalid(parseObject(p))
	case tokenBrackO:
		return wrapInvalid(parseArray(p))
	case tokenNumber:
		return wrapInvalid(parseNumber(p))
	case tokenString:
		return wrapInvalid(parseString(p))
	case tokenKeyword:
		return wrapInvalid(parseKeyword(p))
	case tokenBraceC:
		return wrapInvalid(nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Missing attribute value",
				Detail:   "A JSON value must start with a brace, a bracket, a number, a string, or a keyword.",
				Subject:  &tok.Range,
			},
		})
	case tokenBrackC:
		return wrapInvalid(nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Missing array element value",
				Detail:   "A JSON value must start with a brace, a bracket, a number, a string, or a keyword.",
				Subject:  &tok.Range,
			},
		})
	case tokenEOF:
		return wrapInvalid(nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Missing value",
				Detail:   "The JSON data ends prematurely.",
				Subject:  &tok.Range,
			},
		})
	default:
		return wrapInvalid(nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Invalid start of value",
				Detail:   "A JSON value must start with a brace, a bracket, a number, a string, or a keyword.",
				Subject:  &tok.Range,
			},
		})
	}
}

func tokenCanStartValue(tok token) bool {
	switch tok.Type {
	case tokenBraceO, tokenBrackO, tokenNumber, tokenString, tokenKeyword:
		return true
	default:
		return false
	}
}

func parseObject(p *peeker) (node, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	open := p.Read()
	attrs := map[string]*objectAttr{}

	// recover is used to shift the peeker to what seems to be the end of
	// our object, so that when we encounter an error we leave the peeker
	// at a reasonable point in the token stream to continue parsing.
	recover := func(tok token) {
		open := 1
		for {
			switch tok.Type {
			case tokenBraceO:
				open++
			case tokenBraceC:
				open--
				if open <= 1 {
					return
				}
			case tokenEOF:
				// Ran out of source before we were able to recover,
				// so we'll bail here and let the caller deal with it.
				return
			}
			tok = p.Read()
		}
	}

Token:
	for {
		if p.Peek().Type == tokenBraceC {
			break Token
		}

		keyNode, keyDiags := parseValue(p)
		diags = diags.Extend(keyDiags)
		if keyNode == nil {
			return nil, diags
		}

		keyStrNode, ok := keyNode.(*stringVal)
		if !ok {
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid object attribute name",
				Detail:   "A JSON object attribute name must be a string",
				Subject:  keyNode.StartRange().Ptr(),
			})
		}

		key := keyStrNode.Value

		colon := p.Read()
		if colon.Type != tokenColon {
			recover(colon)

			if colon.Type == tokenBraceC || colon.Type == tokenComma {
				// Catch common mistake of using braces instead of brackets
				// for an object.
				return nil, diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing object value",
					Detail:   "A JSON object attribute must have a value, introduced by a colon.",
					Subject:  &colon.Range,
				})
			}

			if colon.Type == tokenEquals {
				// Possible confusion with native zcl syntax.
				return nil, diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing attribute value colon",
					Detail:   "JSON uses a colon as its name/value delimiter, not an equals sign.",
					Subject:  &colon.Range,
				})
			}

			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing attribute value colon",
				Detail:   "A colon must appear between an object attribute's name and its value.",
				Subject:  &colon.Range,
			})
		}

		valNode, valDiags := parseValue(p)
		diags = diags.Extend(valDiags)
		if valNode == nil {
			return nil, diags
		}

		if existing := attrs[key]; existing != nil {
			// Generate a diagnostic for the duplicate key, but continue parsing
			// anyway since this is a semantic error we can recover from.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate JSON object property",
				Detail: fmt.Sprintf(
					"An property named %q was previously introduced at %s",
					key, existing.NameRange.String(),
				),
				Subject: &keyStrNode.SrcRange,
			})
		}
		attrs[key] = &objectAttr{
			Name:      key,
			Value:     valNode,
			NameRange: keyStrNode.SrcRange,
		}

		switch p.Peek().Type {
		case tokenComma:
			comma := p.Read()
			if p.Peek().Type == tokenBraceC {
				// Special error message for this common mistake
				return nil, diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Trailing comma in object",
					Detail:   "JSON does not permit a trailing comma after the final attribute in an object.",
					Subject:  &comma.Range,
				})
			}
			continue Token
		case tokenEOF:
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unclosed object",
				Detail:   "No closing brace was found for this JSON object.",
				Subject:  &open.Range,
			})
		case tokenBrackC:
			// Consume the bracket anyway, so that we don't return with the peeker
			// at a strange place.
			p.Read()
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Mismatched braces",
				Detail:   "A JSON object must be closed with a brace, not a bracket.",
				Subject:  p.Peek().Range.Ptr(),
			})
		case tokenBraceC:
			break Token
		default:
			recover(p.Read())
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing attribute seperator comma",
				Detail:   "A comma must appear between each attribute declaration in an object.",
				Subject:  p.Peek().Range.Ptr(),
			})
		}

	}

	close := p.Read()
	return &objectVal{
		Attrs:      attrs,
		SrcRange:   hcl.RangeBetween(open.Range, close.Range),
		OpenRange:  open.Range,
		CloseRange: close.Range,
	}, diags
}

func parseArray(p *peeker) (node, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	open := p.Read()
	vals := []node{}

	// recover is used to shift the peeker to what seems to be the end of
	// our array, so that when we encounter an error we leave the peeker
	// at a reasonable point in the token stream to continue parsing.
	recover := func(tok token) {
		open := 1
		for {
			switch tok.Type {
			case tokenBrackO:
				open++
			case tokenBrackC:
				open--
				if open <= 1 {
					return
				}
			case tokenEOF:
				// Ran out of source before we were able to recover,
				// so we'll bail here and let the caller deal with it.
				return
			}
			tok = p.Read()
		}
	}

Token:
	for {
		if p.Peek().Type == tokenBrackC {
			break Token
		}

		valNode, valDiags := parseValue(p)
		diags = diags.Extend(valDiags)
		if valNode == nil {
			return nil, diags
		}

		vals = append(vals, valNode)

		switch p.Peek().Type {
		case tokenComma:
			comma := p.Read()
			if p.Peek().Type == tokenBrackC {
				// Special error message for this common mistake
				return nil, diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Trailing comma in array",
					Detail:   "JSON does not permit a trailing comma after the final attribute in an array.",
					Subject:  &comma.Range,
				})
			}
			continue Token
		case tokenColon:
			recover(p.Read())
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid array value",
				Detail:   "A colon is not used to introduce values in a JSON array.",
				Subject:  p.Peek().Range.Ptr(),
			})
		case tokenEOF:
			recover(p.Read())
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unclosed object",
				Detail:   "No closing bracket was found for this JSON array.",
				Subject:  &open.Range,
			})
		case tokenBraceC:
			recover(p.Read())
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Mismatched brackets",
				Detail:   "A JSON array must be closed with a bracket, not a brace.",
				Subject:  p.Peek().Range.Ptr(),
			})
		case tokenBrackC:
			break Token
		default:
			recover(p.Read())
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing attribute seperator comma",
				Detail:   "A comma must appear between each value in an array.",
				Subject:  p.Peek().Range.Ptr(),
			})
		}

	}

	close := p.Read()
	return &arrayVal{
		Values:    vals,
		SrcRange:  hcl.RangeBetween(open.Range, close.Range),
		OpenRange: open.Range,
	}, diags
}

func parseNumber(p *peeker) (node, hcl.Diagnostics) {
	tok := p.Read()

	// Use encoding/json to validate the number syntax.
	// TODO: Do this more directly to produce better diagnostics.
	var num json.Number
	err := json.Unmarshal(tok.Bytes, &num)
	if err != nil {
		return nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Invalid JSON number",
				Detail:   fmt.Sprintf("There is a syntax error in the given JSON number."),
				Subject:  &tok.Range,
			},
		}
	}

	f, _, err := (&big.Float{}).Parse(string(num), 10)
	if err != nil {
		// Should never happen if above passed, since JSON numbers are a subset
		// of what big.Float can parse...
		return nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Invalid JSON number",
				Detail:   fmt.Sprintf("There is a syntax error in the given JSON number."),
				Subject:  &tok.Range,
			},
		}
	}

	return &numberVal{
		Value:    f,
		SrcRange: tok.Range,
	}, nil
}

func parseString(p *peeker) (node, hcl.Diagnostics) {
	tok := p.Read()
	var str string
	err := json.Unmarshal(tok.Bytes, &str)

	if err != nil {
		var errRange hcl.Range
		if serr, ok := err.(*json.SyntaxError); ok {
			errOfs := serr.Offset
			errPos := tok.Range.Start
			errPos.Byte += int(errOfs)

			// TODO: Use the byte offset to properly count unicode
			// characters for the column, and mark the whole of the
			// character that was wrong as part of our range.
			errPos.Column += int(errOfs)

			errEndPos := errPos
			errEndPos.Byte++
			errEndPos.Column++

			errRange = hcl.Range{
				Filename: tok.Range.Filename,
				Start:    errPos,
				End:      errEndPos,
			}
		} else {
			errRange = tok.Range
		}

		var contextRange *hcl.Range
		if errRange != tok.Range {
			contextRange = &tok.Range
		}

		// FIXME: Eventually we should parse strings directly here so
		// we can produce a more useful error message in the face fo things
		// such as invalid escapes, etc.
		return nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Invalid JSON string",
				Detail:   fmt.Sprintf("There is a syntax error in the given JSON string."),
				Subject:  &errRange,
				Context:  contextRange,
			},
		}
	}

	return &stringVal{
		Value:    str,
		SrcRange: tok.Range,
	}, nil
}

func parseKeyword(p *peeker) (node, hcl.Diagnostics) {
	tok := p.Read()
	s := string(tok.Bytes)

	switch s {
	case "true":
		return &booleanVal{
			Value:    true,
			SrcRange: tok.Range,
		}, nil
	case "false":
		return &booleanVal{
			Value:    false,
			SrcRange: tok.Range,
		}, nil
	case "null":
		return &nullVal{
			SrcRange: tok.Range,
		}, nil
	case "undefined", "NaN", "Infinity":
		return nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Invalid JSON keyword",
				Detail:   fmt.Sprintf("The JavaScript identifier %q cannot be used in JSON.", s),
				Subject:  &tok.Range,
			},
		}
	default:
		var dym string
		if suggest := keywordSuggestion(s); suggest != "" {
			dym = fmt.Sprintf(" Did you mean %q?", suggest)
		}

		return nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Invalid JSON keyword",
				Detail:   fmt.Sprintf("%q is not a valid JSON keyword.%s", s, dym),
				Subject:  &tok.Range,
			},
		}
	}
}
