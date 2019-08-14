package jsonpath

import (
	"context"
	"fmt"
	"math"
	"text/scanner"

	"github.com/PaesslerAG/gval"
)

type parser struct {
	*gval.Parser
	path path
}

func parseRootPath(ctx context.Context, gParser *gval.Parser) (r gval.Evaluable, err error) {
	p := newParser(gParser)
	return p.parse(ctx)
}

func parseCurrentPath(ctx context.Context, gParser *gval.Parser) (r gval.Evaluable, err error) {
	p := newParser(gParser)
	p.appendPlainSelector(currentElementSelector())
	return p.parse(ctx)
}

func newParser(p *gval.Parser) *parser {
	return &parser{Parser: p, path: plainPath{}}
}

func (p *parser) parse(c context.Context) (r gval.Evaluable, err error) {
	err = p.parsePath(c)

	if err != nil {
		return nil, err
	}
	return p.path.evaluate, nil
}

func (p *parser) parsePath(c context.Context) error {
	switch p.Scan() {
	case '.':
		return p.parseSelect(c)
	case '[':
		keys, seperator, err := p.parseBracket(c)

		if err != nil {
			return err
		}

		switch seperator {
		case ':':
			if len(keys) > 3 {
				return fmt.Errorf("range query has at least the parameter [min:max:step]")
			}
			keys = append(keys, []gval.Evaluable{
				p.Const(0), p.Const(float64(math.MaxInt32)), p.Const(1)}[len(keys):]...)
			p.appendAmbiguousSelector(rangeSelector(keys[0], keys[1], keys[2]))
		case '?':
			if len(keys) != 1 {
				return fmt.Errorf("filter needs exactly one key")
			}
			p.appendAmbiguousSelector(filterSelector(keys[0]))
		default:
			if len(keys) == 1 {
				p.appendPlainSelector(directSelector(keys[0]))
			} else {
				p.appendAmbiguousSelector(multiSelector(keys))
			}
		}
		return p.parsePath(c)
	case '(':
		return p.parseScript(c)
	default:
		p.Camouflage("jsonpath", '.', '[', '(')
		return nil
	}
}

func (p *parser) parseSelect(c context.Context) error {
	scan := p.Scan()
	switch scan {
	case scanner.Ident:
		p.appendPlainSelector(directSelector(p.Const(p.TokenText())))
		return p.parsePath(c)
	case '.':
		p.appendAmbiguousSelector(mapperSelector())
		return p.parseMapper(c)
	case '*':
		p.appendAmbiguousSelector(starSelector())
		return p.parsePath(c)
	default:
		return p.Expected("JSON select", scanner.Ident, '.', '*')
	}
}

func (p *parser) parseBracket(c context.Context) (keys []gval.Evaluable, seperator rune, err error) {
	for {
		scan := p.Scan()
		skipScan := false
		switch scan {
		case '?':
			skipScan = true
		case ':':
			i := float64(0)
			if len(keys) == 1 {
				i = math.MaxInt32
			}
			keys = append(keys, p.Const(i))
			skipScan = true
		case '*':
			if p.Scan() != ']' {
				return nil, 0, p.Expected("JSON bracket star", ']')
			}
			return []gval.Evaluable{}, 0, nil
		case ']':
			if seperator == ':' {
				skipScan = true
				break
			}
			fallthrough
		default:
			p.Camouflage("jsonpath brackets")
			key, err := p.ParseExpression(c)
			if err != nil {
				return nil, 0, err
			}
			keys = append(keys, key)
		}
		if !skipScan {
			scan = p.Scan()
		}
		if seperator == 0 {
			seperator = scan
		}
		switch scan {
		case ':', ',':
		case ']':
			return
		case '?':
			if len(keys) != 0 {
				return nil, 0, p.Expected("JSON filter", ']')
			}
		default:
			return nil, 0, p.Expected("JSON bracket separator", ':', ',')
		}
		if seperator != scan {
			return nil, 0, fmt.Errorf("mixed %v and %v in JSON bracket", seperator, scan)
		}
	}
}

func (p *parser) parseMapper(c context.Context) error {
	scan := p.Scan()
	switch scan {
	case scanner.Ident:
		p.appendPlainSelector(directSelector(p.Const(p.TokenText())))
	case '[':
		keys, seperator, err := p.parseBracket(c)

		if err != nil {
			return err
		}
		switch seperator {
		case ':':
			return fmt.Errorf("mapper can not be combined with range query")
		case '?':
			if len(keys) != 1 {
				return fmt.Errorf("filter needs exactly one key")
			}
			p.appendAmbiguousSelector(filterSelector(keys[0]))
		default:
			p.appendAmbiguousSelector(multiSelector(keys))
		}
	case '*':
		p.appendAmbiguousSelector(starSelector())
	case '(':
		return p.parseScript(c)
	default:
		return p.Expected("JSON mapper", '[', scanner.Ident, '*')
	}
	return p.parsePath(c)
}

func (p *parser) parseScript(c context.Context) error {
	script, err := p.ParseExpression(c)
	if err != nil {
		return err
	}
	if p.Scan() != ')' {
		return p.Expected("jsnopath script", ')')
	}
	p.appendPlainSelector(newScript(script))
	return p.parsePath(c)
}

func (p *parser) appendPlainSelector(next plainSelector) {
	p.path = p.path.withPlainSelector(next)
}

func (p *parser) appendAmbiguousSelector(next ambiguousSelector) {
	p.path = p.path.withAmbiguousSelector(next)
}
