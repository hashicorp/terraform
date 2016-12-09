package parser

import (
	"strconv"
	"unicode/utf8"

	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/hil/scanner"
)

func Parse(ch <-chan *scanner.Token) (ast.Node, error) {
	peeker := scanner.NewPeeker(ch)
	parser := &parser{peeker}
	output, err := parser.ParseTopLevel()
	peeker.Close()
	return output, err
}

type parser struct {
	peeker *scanner.Peeker
}

func (p *parser) ParseTopLevel() (ast.Node, error) {
	return p.parseInterpolationSeq(false)
}

func (p *parser) ParseQuoted() (ast.Node, error) {
	return p.parseInterpolationSeq(true)
}

// parseInterpolationSeq parses either the top-level sequence of literals
// and interpolation expressions or a similar sequence within a quoted
// string inside an interpolation expression. The latter case is requested
// by setting 'quoted' to true.
func (p *parser) parseInterpolationSeq(quoted bool) (ast.Node, error) {
	literalType := scanner.LITERAL
	endType := scanner.EOF
	if quoted {
		// exceptions for quoted sequences
		literalType = scanner.STRING
		endType = scanner.CQUOTE
	}

	startPos := p.peeker.Peek().Pos

	if quoted {
		tok := p.peeker.Read()
		if tok.Type != scanner.OQUOTE {
			return nil, ExpectationError("open quote", tok)
		}
	}

	var exprs []ast.Node
	for {
		tok := p.peeker.Read()

		if tok.Type == endType {
			break
		}

		switch tok.Type {
		case literalType:
			val, err := p.parseStringToken(tok)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, &ast.LiteralNode{
				Value: val,
				Typex: ast.TypeString,
				Posx:  tok.Pos,
			})
		case scanner.BEGIN:
			expr, err := p.ParseInterpolation()
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)
		default:
			return nil, ExpectationError(`"${"`, tok)
		}
	}

	if len(exprs) == 0 {
		// If we have no parts at all then the input must've
		// been an empty string.
		exprs = append(exprs, &ast.LiteralNode{
			Value: "",
			Typex: ast.TypeString,
			Posx:  startPos,
		})
	}

	// As a special case, if our "Output" contains only one expression
	// and it's a literal string then we'll hoist it up to be our
	// direct return value, so callers can easily recognize a string
	// that has no interpolations at all.
	if len(exprs) == 1 {
		if lit, ok := exprs[0].(*ast.LiteralNode); ok {
			if lit.Typex == ast.TypeString {
				return lit, nil
			}
		}
	}

	return &ast.Output{
		Exprs: exprs,
		Posx:  startPos,
	}, nil
}

// parseStringToken takes a token of either LITERAL or STRING type and
// returns the interpreted string, after processing any relevant
// escape sequences.
func (p *parser) parseStringToken(tok *scanner.Token) (string, error) {
	var backslashes bool
	switch tok.Type {
	case scanner.LITERAL:
		backslashes = false
	case scanner.STRING:
		backslashes = true
	default:
		panic("unsupported string token type")
	}

	raw := []byte(tok.Content)
	buf := make([]byte, 0, len(raw))

	for i := 0; i < len(raw); i++ {
		b := raw[i]
		more := len(raw) > (i + 1)

		if b == '$' {
			if more && raw[i+1] == '$' {
				// skip over the second dollar sign
				i++
			}
		} else if backslashes && b == '\\' {
			if !more {
				return "", Errorf(
					ast.Pos{
						Column: tok.Pos.Column + utf8.RuneCount(raw[:i]),
						Line:   tok.Pos.Line,
					},
					`unfinished backslash escape sequence`,
				)
			}
			escapeType := raw[i+1]
			switch escapeType {
			case '\\':
				// skip over the second slash
				i++
			case 'n':
				b = '\n'
				i++
			case '"':
				b = '"'
				i++
			default:
				return "", Errorf(
					ast.Pos{
						Column: tok.Pos.Column + utf8.RuneCount(raw[:i]),
						Line:   tok.Pos.Line,
					},
					`invalid backslash escape sequence`,
				)
			}
		}

		buf = append(buf, b)
	}

	return string(buf), nil
}

func (p *parser) ParseInterpolation() (ast.Node, error) {
	// By the time we're called, we're already "inside" the ${ sequence
	// because the caller consumed the ${ token.

	expr, err := p.ParseExpression()
	if err != nil {
		return nil, err
	}

	err = p.requireTokenType(scanner.END, `"}"`)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (p *parser) ParseExpression() (ast.Node, error) {
	return p.parseTernaryCond()
}

func (p *parser) parseTernaryCond() (ast.Node, error) {
	// The ternary condition operator (.. ? .. : ..) behaves somewhat
	// like a binary operator except that the "operator" is itself
	// an expression enclosed in two punctuation characters.
	// The middle expression is parsed as if the ? and : symbols
	// were parentheses. The "rhs" (the "false expression") is then
	// treated right-associatively so it behaves similarly to the
	// middle in terms of precedence.

	startPos := p.peeker.Peek().Pos

	var cond, trueExpr, falseExpr ast.Node
	var err error

	cond, err = p.parseBinaryOps(binaryOps)
	if err != nil {
		return nil, err
	}

	next := p.peeker.Peek()
	if next.Type != scanner.QUESTION {
		return cond, nil
	}

	p.peeker.Read() // eat question mark

	trueExpr, err = p.ParseExpression()
	if err != nil {
		return nil, err
	}

	colon := p.peeker.Read()
	if colon.Type != scanner.COLON {
		return nil, ExpectationError(":", colon)
	}

	falseExpr, err = p.ParseExpression()
	if err != nil {
		return nil, err
	}

	return &ast.Conditional{
		CondExpr:  cond,
		TrueExpr:  trueExpr,
		FalseExpr: falseExpr,
		Posx:      startPos,
	}, nil
}

// parseBinaryOps calls itself recursively to work through all of the
// operator precedence groups, and then eventually calls ParseExpressionTerm
// for each operand.
func (p *parser) parseBinaryOps(ops []map[scanner.TokenType]ast.ArithmeticOp) (ast.Node, error) {
	if len(ops) == 0 {
		// We've run out of operators, so now we'll just try to parse a term.
		return p.ParseExpressionTerm()
	}

	thisLevel := ops[0]
	remaining := ops[1:]

	startPos := p.peeker.Peek().Pos

	var lhs, rhs ast.Node
	operator := ast.ArithmeticOpInvalid
	var err error

	// parse a term that might be the first operand of a binary
	// expression or it might just be a standalone term, but
	// we won't know until we've parsed it and can look ahead
	// to see if there's an operator token.
	lhs, err = p.parseBinaryOps(remaining)
	if err != nil {
		return nil, err
	}

	// We'll keep eating up arithmetic operators until we run
	// out, so that operators with the same precedence will combine in a
	// left-associative manner:
	// a+b+c => (a+b)+c, not a+(b+c)
	//
	// Should we later want to have right-associative operators, a way
	// to achieve that would be to call back up to ParseExpression here
	// instead of iteratively parsing only the remaining operators.
	for {
		next := p.peeker.Peek()
		var newOperator ast.ArithmeticOp
		var ok bool
		if newOperator, ok = thisLevel[next.Type]; !ok {
			break
		}

		// Are we extending an expression started on
		// the previous iteration?
		if operator != ast.ArithmeticOpInvalid {
			lhs = &ast.Arithmetic{
				Op:    operator,
				Exprs: []ast.Node{lhs, rhs},
				Posx:  startPos,
			}
		}

		operator = newOperator
		p.peeker.Read() // eat operator token
		rhs, err = p.parseBinaryOps(remaining)
		if err != nil {
			return nil, err
		}
	}

	if operator != ast.ArithmeticOpInvalid {
		return &ast.Arithmetic{
			Op:    operator,
			Exprs: []ast.Node{lhs, rhs},
			Posx:  startPos,
		}, nil
	} else {
		return lhs, nil
	}
}

func (p *parser) ParseExpressionTerm() (ast.Node, error) {

	next := p.peeker.Peek()

	switch next.Type {

	case scanner.OPAREN:
		p.peeker.Read()
		expr, err := p.ParseExpression()
		if err != nil {
			return nil, err
		}
		err = p.requireTokenType(scanner.CPAREN, `")"`)
		return expr, err

	case scanner.OQUOTE:
		return p.ParseQuoted()

	case scanner.INTEGER:
		tok := p.peeker.Read()
		val, err := strconv.Atoi(tok.Content)
		if err != nil {
			return nil, TokenErrorf(tok, "invalid integer: %s", err)
		}
		return &ast.LiteralNode{
			Value: val,
			Typex: ast.TypeInt,
			Posx:  tok.Pos,
		}, nil

	case scanner.FLOAT:
		tok := p.peeker.Read()
		val, err := strconv.ParseFloat(tok.Content, 64)
		if err != nil {
			return nil, TokenErrorf(tok, "invalid float: %s", err)
		}
		return &ast.LiteralNode{
			Value: val,
			Typex: ast.TypeFloat,
			Posx:  tok.Pos,
		}, nil

	case scanner.BOOL:
		tok := p.peeker.Read()
		// the scanner guarantees that tok.Content is either "true" or "false"
		var val bool
		if tok.Content[0] == 't' {
			val = true
		} else {
			val = false
		}
		return &ast.LiteralNode{
			Value: val,
			Typex: ast.TypeBool,
			Posx:  tok.Pos,
		}, nil

	case scanner.MINUS:
		opTok := p.peeker.Read()
		// important to use ParseExpressionTerm rather than ParseExpression
		// here, otherwise we can capture a following binary expression into
		// our negation.
		// e.g. -46+5 should parse as (0-46)+5, not 0-(46+5)
		operand, err := p.ParseExpressionTerm()
		if err != nil {
			return nil, err
		}
		// The AST currently represents negative numbers as
		// a binary subtraction of the number from zero.
		return &ast.Arithmetic{
			Op: ast.ArithmeticOpSub,
			Exprs: []ast.Node{
				&ast.LiteralNode{
					Value: 0,
					Typex: ast.TypeInt,
					Posx:  opTok.Pos,
				},
				operand,
			},
			Posx: opTok.Pos,
		}, nil

	case scanner.BANG:
		opTok := p.peeker.Read()
		// important to use ParseExpressionTerm rather than ParseExpression
		// here, otherwise we can capture a following binary expression into
		// our negation.
		operand, err := p.ParseExpressionTerm()
		if err != nil {
			return nil, err
		}
		// The AST currently represents binary negation as an equality
		// test with "false".
		return &ast.Arithmetic{
			Op: ast.ArithmeticOpEqual,
			Exprs: []ast.Node{
				&ast.LiteralNode{
					Value: false,
					Typex: ast.TypeBool,
					Posx:  opTok.Pos,
				},
				operand,
			},
			Posx: opTok.Pos,
		}, nil

	case scanner.IDENTIFIER:
		return p.ParseScopeInteraction()

	default:
		return nil, ExpectationError("expression", next)
	}
}

// ParseScopeInteraction parses the expression types that interact
// with the evaluation scope: variable access, function calls, and
// indexing.
//
// Indexing should actually be a distinct operator in its own right,
// so that e.g. it can be applied to the result of a function call,
// but for now we're preserving the behavior of the older yacc-based
// parser.
func (p *parser) ParseScopeInteraction() (ast.Node, error) {
	first := p.peeker.Read()
	startPos := first.Pos
	if first.Type != scanner.IDENTIFIER {
		return nil, ExpectationError("identifier", first)
	}

	next := p.peeker.Peek()
	if next.Type == scanner.OPAREN {
		// function call
		funcName := first.Content
		p.peeker.Read() // eat paren
		var args []ast.Node

		for {
			if p.peeker.Peek().Type == scanner.CPAREN {
				break
			}

			arg, err := p.ParseExpression()
			if err != nil {
				return nil, err
			}

			args = append(args, arg)

			if p.peeker.Peek().Type == scanner.COMMA {
				p.peeker.Read() // eat comma
				continue
			} else {
				break
			}
		}

		err := p.requireTokenType(scanner.CPAREN, `")"`)
		if err != nil {
			return nil, err
		}

		return &ast.Call{
			Func: funcName,
			Args: args,
			Posx: startPos,
		}, nil
	}

	varNode := &ast.VariableAccess{
		Name: first.Content,
		Posx: startPos,
	}

	if p.peeker.Peek().Type == scanner.OBRACKET {
		// index operator
		startPos := p.peeker.Read().Pos // eat bracket
		indexExpr, err := p.ParseExpression()
		if err != nil {
			return nil, err
		}
		err = p.requireTokenType(scanner.CBRACKET, `"]"`)
		if err != nil {
			return nil, err
		}
		return &ast.Index{
			Target: varNode,
			Key:    indexExpr,
			Posx:   startPos,
		}, nil
	}

	return varNode, nil
}

// requireTokenType consumes the next token an returns an error if its
// type does not match the given type. nil is returned if the type matches.
//
// This is a helper around peeker.Read() for situations where the parser just
// wants to assert that a particular token type must be present.
func (p *parser) requireTokenType(wantType scanner.TokenType, wantName string) error {
	token := p.peeker.Read()
	if token.Type != wantType {
		return ExpectationError(wantName, token)
	}
	return nil
}
