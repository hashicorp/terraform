package gval

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"text/scanner"
)

//ParseExpression scans an expression into an Evaluable.
func (p *Parser) ParseExpression(c context.Context) (eval Evaluable, err error) {
	stack := stageStack{}
	for {
		eval, err = p.ParseNextExpression(c)
		if err != nil {
			return nil, err
		}

		if stage, err := p.parseOperator(c, &stack, eval); err != nil {
			return nil, err
		} else if err = stack.push(stage); err != nil {
			return nil, err
		}

		if stack.peek().infixBuilder == nil {
			return stack.pop().Evaluable, nil
		}
	}
}

//ParseNextExpression scans the expression ignoring following operators
func (p *Parser) ParseNextExpression(c context.Context) (eval Evaluable, err error) {
	scan := p.Scan()
	ex, ok := p.prefixes[scan]
	if !ok {
		return nil, p.Expected("extensions")
	}
	return ex(c, p)
}

func parseString(c context.Context, p *Parser) (Evaluable, error) {
	s, err := strconv.Unquote(p.TokenText())
	if err != nil {
		return nil, fmt.Errorf("could not parse string: %s", err)
	}
	return p.Const(s), nil
}

func parseNumber(c context.Context, p *Parser) (Evaluable, error) {
	n, err := strconv.ParseFloat(p.TokenText(), 64)
	if err != nil {
		return nil, err
	}
	return p.Const(n), nil
}

func parseParentheses(c context.Context, p *Parser) (Evaluable, error) {
	eval, err := p.ParseExpression(c)
	if err != nil {
		return nil, err
	}
	switch p.Scan() {
	case ')':
		return eval, nil
	default:
		return nil, p.Expected("parentheses", ')')
	}
}

func (p *Parser) parseOperator(c context.Context, stack *stageStack, eval Evaluable) (st stage, err error) {
	for {
		scan := p.Scan()
		op := p.TokenText()
		mustOp := false
		if p.isSymbolOperation(scan) {
			scan = p.Peek()
			for p.isSymbolOperation(scan) {
				mustOp = true
				op += string(scan)
				p.Next()
				scan = p.Peek()
			}
		} else if scan != scanner.Ident {
			p.Camouflage("operator")
			return stage{Evaluable: eval}, nil
		}
		operator, _ := p.operators[op]
		switch operator := operator.(type) {
		case *infix:
			return stage{
				Evaluable:          eval,
				infixBuilder:       operator.builder,
				operatorPrecedence: operator.operatorPrecedence,
			}, nil
		case directInfix:
			return stage{
				Evaluable:          eval,
				infixBuilder:       operator.infixBuilder,
				operatorPrecedence: operator.operatorPrecedence,
			}, nil
		case postfix:
			if err = stack.push(stage{
				operatorPrecedence: operator.operatorPrecedence,
				Evaluable:          eval,
			}); err != nil {
				return stage{}, err
			}
			eval, err = operator.f(c, p, stack.pop().Evaluable, operator.operatorPrecedence)
			if err != nil {
				return
			}
			continue
		}

		if !mustOp {
			p.Camouflage("operator")
			return stage{Evaluable: eval}, nil
		}
		return stage{}, fmt.Errorf("unknown operator %s", op)
	}
}

func parseIdent(c context.Context, p *Parser) (call string, alternative func() (Evaluable, error), err error) {
	token := p.TokenText()
	return token,
		func() (Evaluable, error) {
			fullname := token

			keys := []Evaluable{p.Const(token)}
			for {
				scan := p.Scan()
				switch scan {
				case '.':
					scan = p.Scan()
					switch scan {
					case scanner.Ident:
						token = p.TokenText()
						keys = append(keys, p.Const(token))
					default:
						return nil, p.Expected("field", scanner.Ident)
					}
				case '(':
					args, err := p.parseArguments(c)
					if err != nil {
						return nil, err
					}
					return p.callEvaluable(fullname, p.Var(keys...), args...), nil
				case '[':
					key, err := p.ParseExpression(c)
					if err != nil {
						return nil, err
					}
					switch p.Scan() {
					case ']':
						keys = append(keys, key)
					default:
						return nil, p.Expected("array key", ']')
					}
				default:
					p.Camouflage("variable", '.', '(', '[')
					return p.Var(keys...), nil
				}
			}
		}, nil

}

func (p *Parser) parseArguments(c context.Context) (args []Evaluable, err error) {
	if p.Scan() == ')' {
		return
	}
	p.Camouflage("scan arguments", ')')
	for {
		arg, err := p.ParseExpression(c)
		args = append(args, arg)
		if err != nil {
			return nil, err
		}
		switch p.Scan() {
		case ')':
			return args, nil
		case ',':
		default:
			return nil, p.Expected("arguments", ')', ',')
		}
	}
}

func inArray(a, b interface{}) (interface{}, error) {
	col, ok := b.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected type []interface{} for in operator but got %T", b)
	}
	for _, value := range col {
		if reflect.DeepEqual(a, value) {
			return true, nil
		}
	}
	return false, nil
}

func parseIf(c context.Context, p *Parser, e Evaluable) (Evaluable, error) {
	a, err := p.ParseExpression(c)
	if err != nil {
		return nil, err
	}
	b := p.Const(nil)
	switch p.Scan() {
	case ':':
		b, err = p.ParseExpression(c)
		if err != nil {
			return nil, err
		}
	case scanner.EOF:
	default:
		return nil, p.Expected("<> ? <> : <>", ':', scanner.EOF)
	}
	return func(c context.Context, v interface{}) (interface{}, error) {
		x, err := e(c, v)
		if err != nil {
			return nil, err
		}
		if x == false || x == nil {
			return b(c, v)
		}
		return a(c, v)
	}, nil
}

func parseJSONArray(c context.Context, p *Parser) (Evaluable, error) {
	evals := []Evaluable{}
	for {
		switch p.Scan() {
		default:
			p.Camouflage("array", ',', ']')
			eval, err := p.ParseExpression(c)
			if err != nil {
				return nil, err
			}
			evals = append(evals, eval)
		case ',':
		case ']':
			return func(c context.Context, v interface{}) (interface{}, error) {
				vs := make([]interface{}, len(evals))
				for i, e := range evals {
					eval, err := e(c, v)
					if err != nil {
						return nil, err
					}
					vs[i] = eval
				}

				return vs, nil
			}, nil
		}
	}
}

func parseJSONObject(c context.Context, p *Parser) (Evaluable, error) {
	type kv struct {
		key   Evaluable
		value Evaluable
	}
	evals := []kv{}
	for {
		switch p.Scan() {
		default:
			p.Camouflage("object", ',', '}')
			key, err := p.ParseExpression(c)
			if err != nil {
				return nil, err
			}
			if p.Scan() != ':' {
				if err != nil {
					return nil, p.Expected("object", ':')
				}
			}
			value, err := p.ParseExpression(c)
			if err != nil {
				return nil, err
			}
			evals = append(evals, kv{key, value})
		case ',':
		case '}':
			return func(c context.Context, v interface{}) (interface{}, error) {
				vs := map[string]interface{}{}
				for _, e := range evals {
					value, err := e.value(c, v)
					if err != nil {
						return nil, err
					}
					key, err := e.key.EvalString(c, v)
					if err != nil {
						return nil, err
					}
					vs[key] = value
				}
				return vs, nil
			}, nil
		}
	}
}
