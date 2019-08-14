package jsonpath

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"text/scanner"

	"github.com/PaesslerAG/gval"
)

type keyValueVisitor func(key string, value interface{})

type jsonObject interface {
	visitElements(c context.Context, v interface{}, visit keyValueVisitor) error
}

type jsonObjectSlice []jsonObject

type keyValuePair struct {
	key   gval.Evaluable
	value gval.Evaluable
}

type keyValueMatcher struct {
	key     gval.Evaluable
	matcher func(c context.Context, r interface{}, visit pathMatcher)
}

func parseJSONObject(ctx context.Context, p *gval.Parser) (gval.Evaluable, error) {
	evals := jsonObjectSlice{}
	for {
		switch p.Scan() {
		default:
			hasWildcard := false

			p.Camouflage("object", ',', '}')
			key, err := p.ParseExpression(context.WithValue(ctx, hasPlaceholdersContextKey{}, &hasWildcard))
			if err != nil {
				return nil, err
			}
			if p.Scan() != ':' {
				if err != nil {
					return nil, p.Expected("object", ':')
				}
			}
			e, err := parseJSONObjectElement(ctx, p, hasWildcard, key)
			if err != nil {
				return nil, err
			}
			evals.addElements(e)
		case ',':
		case '}':
			return evals.evaluable, nil
		}
	}
}

func parseJSONObjectElement(ctx context.Context, gParser *gval.Parser, hasWildcard bool, key gval.Evaluable) (jsonObject, error) {
	if hasWildcard {
		p := newParser(gParser)
		switch gParser.Scan() {
		case '$':
		case '@':
			p.appendPlainSelector(currentElementSelector())
		default:
			return nil, p.Expected("JSONPath key and value")
		}

		if err := p.parsePath(ctx); err != nil {
			return nil, err
		}
		return keyValueMatcher{key, p.path.visitMatchs}, nil
	}
	value, err := gParser.ParseExpression(ctx)
	if err != nil {
		return nil, err
	}
	return keyValuePair{key, value}, nil
}

func (kv keyValuePair) visitElements(c context.Context, v interface{}, visit keyValueVisitor) error {
	value, err := kv.value(c, v)
	if err != nil {
		return err
	}
	key, err := kv.key.EvalString(c, v)
	if err != nil {
		return err
	}
	visit(key, value)
	return nil
}

func (kv keyValueMatcher) visitElements(c context.Context, v interface{}, visit keyValueVisitor) (err error) {
	kv.matcher(c, v, func(keys []interface{}, match interface{}) {
		key, er := kv.key.EvalString(context.WithValue(c, placeholdersContextKey{}, keys), v)
		if er != nil {
			err = er
		}
		visit(key, match)
	})
	return
}

func (j *jsonObjectSlice) addElements(e jsonObject) {
	*j = append(*j, e)
}

func (j jsonObjectSlice) evaluable(c context.Context, v interface{}) (interface{}, error) {
	vs := map[string]interface{}{}

	err := j.visitElements(c, v, func(key string, value interface{}) { vs[key] = value })
	if err != nil {
		return nil, err
	}
	return vs, nil
}

func (j jsonObjectSlice) visitElements(ctx context.Context, v interface{}, visit keyValueVisitor) (err error) {
	for _, e := range j {
		if err := e.visitElements(ctx, v, visit); err != nil {
			return err
		}
	}
	return nil
}

func parsePlaceholder(c context.Context, p *gval.Parser) (gval.Evaluable, error) {
	hasWildcard := c.Value(hasPlaceholdersContextKey{})
	if hasWildcard == nil {
		return nil, fmt.Errorf("JSONPath placeholder must only be used in an JSON object key")
	}
	*(hasWildcard.(*bool)) = true
	switch p.Scan() {
	case scanner.Int:
		id, err := strconv.Atoi(p.TokenText())
		if err != nil {
			return nil, err
		}
		return placeholder(id).evaluable, nil
	default:
		p.Camouflage("JSONPath placeholder")
		return allPlaceholders.evaluable, nil
	}
}

type hasPlaceholdersContextKey struct{}

type placeholdersContextKey struct{}

type placeholder int

const allPlaceholders = placeholder(-1)

func (key placeholder) evaluable(c context.Context, v interface{}) (interface{}, error) {
	wildcards, ok := c.Value(placeholdersContextKey{}).([]interface{})
	if !ok || len(wildcards) <= int(key) {
		return nil, fmt.Errorf("JSONPath placeholder #%d is not available", key)
	}
	if key == allPlaceholders {
		sb := bytes.Buffer{}
		sb.WriteString("$")
		quoteWildcardValues(&sb, wildcards)
		return sb.String(), nil
	}
	return wildcards[int(key)], nil
}

func quoteWildcardValues(sb *bytes.Buffer, wildcards []interface{}) {
	for _, w := range wildcards {
		if wildcards, ok := w.([]interface{}); ok {
			quoteWildcardValues(sb, wildcards)
			continue
		}
		sb.WriteString(fmt.Sprintf("[%v]",
			strconv.Quote(fmt.Sprint(w)),
		))
	}
}
