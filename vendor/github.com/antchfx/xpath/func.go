package xpath

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// The XPath function list.

func predicate(q query) func(NodeNavigator) bool {
	type Predicater interface {
		Test(NodeNavigator) bool
	}
	if p, ok := q.(Predicater); ok {
		return p.Test
	}
	return func(NodeNavigator) bool { return true }
}

// positionFunc is a XPath Node Set functions position().
func positionFunc(q query, t iterator) interface{} {
	var (
		count = 1
		node  = t.Current()
	)
	test := predicate(q)
	for node.MoveToPrevious() {
		if test(node) {
			count++
		}
	}
	return float64(count)
}

// lastFunc is a XPath Node Set functions last().
func lastFunc(q query, t iterator) interface{} {
	var (
		count = 0
		node  = t.Current()
	)
	node.MoveToFirst()
	test := predicate(q)
	for {
		if test(node) {
			count++
		}
		if !node.MoveToNext() {
			break
		}
	}
	return float64(count)
}

// countFunc is a XPath Node Set functions count(node-set).
func countFunc(q query, t iterator) interface{} {
	var count = 0
	test := predicate(q)
	switch typ := q.Evaluate(t).(type) {
	case query:
		for node := typ.Select(t); node != nil; node = typ.Select(t) {
			if test(node) {
				count++
			}
		}
	}
	return float64(count)
}

// sumFunc is a XPath Node Set functions sum(node-set).
func sumFunc(q query, t iterator) interface{} {
	var sum float64
	switch typ := q.Evaluate(t).(type) {
	case query:
		for node := typ.Select(t); node != nil; node = typ.Select(t) {
			if v, err := strconv.ParseFloat(node.Value(), 64); err == nil {
				sum += v
			}
		}
	case float64:
		sum = typ
	case string:
		v, err := strconv.ParseFloat(typ, 64)
		if err != nil {
			panic(errors.New("sum() function argument type must be a node-set or number"))
		}
		sum = v
	}
	return sum
}

func asNumber(t iterator, o interface{}) float64 {
	switch typ := o.(type) {
	case query:
		node := typ.Select(t)
		if node == nil {
			return float64(0)
		}
		if v, err := strconv.ParseFloat(node.Value(), 64); err == nil {
			return v
		}
	case float64:
		return typ
	case string:
		v, err := strconv.ParseFloat(typ, 64)
		if err != nil {
			panic(errors.New("ceiling() function argument type must be a node-set or number"))
		}
		return v
	}
	return 0
}

// ceilingFunc is a XPath Node Set functions ceiling(node-set).
func ceilingFunc(q query, t iterator) interface{} {
	val := asNumber(t, q.Evaluate(t))
	return math.Ceil(val)
}

// floorFunc is a XPath Node Set functions floor(node-set).
func floorFunc(q query, t iterator) interface{} {
	val := asNumber(t, q.Evaluate(t))
	return math.Floor(val)
}

// roundFunc is a XPath Node Set functions round(node-set).
func roundFunc(q query, t iterator) interface{} {
	val := asNumber(t, q.Evaluate(t))
	//return math.Round(val)
	return round(val)
}

// nameFunc is a XPath functions name([node-set]).
func nameFunc(q query, t iterator) interface{} {
	v := q.Select(t)
	if v == nil {
		return ""
	}
	ns := v.Prefix()
	if ns == "" {
		return v.LocalName()
	}
	return ns + ":" + v.LocalName()
}

// localNameFunc is a XPath functions local-name([node-set]).
func localNameFunc(q query, t iterator) interface{} {
	v := q.Select(t)
	if v == nil {
		return ""
	}
	return v.LocalName()
}

// namespaceFunc is a XPath functions namespace-uri([node-set]).
func namespaceFunc(q query, t iterator) interface{} {
	v := q.Select(t)
	if v == nil {
		return ""
	}
	return v.Prefix()
}

func asBool(t iterator, v interface{}) bool {
	switch v := v.(type) {
	case nil:
		return false
	case *NodeIterator:
		return v.MoveNext()
	case bool:
		return bool(v)
	case float64:
		return v != 0
	case string:
		return v != ""
	case query:
		return v.Select(t) != nil
	default:
		panic(fmt.Errorf("unexpected type: %T", v))
	}
}

func asString(t iterator, v interface{}) string {
	switch v := v.(type) {
	case nil:
		return ""
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case string:
		return v
	case query:
		node := v.Select(t)
		if node == nil {
			return ""
		}
		return node.Value()
	default:
		panic(fmt.Errorf("unexpected type: %T", v))
	}
}

// booleanFunc is a XPath functions boolean([node-set]).
func booleanFunc(q query, t iterator) interface{} {
	v := q.Evaluate(t)
	return asBool(t, v)
}

// numberFunc is a XPath functions number([node-set]).
func numberFunc(q query, t iterator) interface{} {
	v := q.Evaluate(t)
	return asNumber(t, v)
}

// stringFunc is a XPath functions string([node-set]).
func stringFunc(q query, t iterator) interface{} {
	v := q.Evaluate(t)
	return asString(t, v)
}

// startwithFunc is a XPath functions starts-with(string, string).
func startwithFunc(arg1, arg2 query) func(query, iterator) interface{} {
	return func(q query, t iterator) interface{} {
		var (
			m, n string
			ok   bool
		)
		switch typ := arg1.Evaluate(t).(type) {
		case string:
			m = typ
		case query:
			node := typ.Select(t)
			if node == nil {
				return false
			}
			m = node.Value()
		default:
			panic(errors.New("starts-with() function argument type must be string"))
		}
		n, ok = arg2.Evaluate(t).(string)
		if !ok {
			panic(errors.New("starts-with() function argument type must be string"))
		}
		return strings.HasPrefix(m, n)
	}
}

// endwithFunc is a XPath functions ends-with(string, string).
func endwithFunc(arg1, arg2 query) func(query, iterator) interface{} {
	return func(q query, t iterator) interface{} {
		var (
			m, n string
			ok   bool
		)
		switch typ := arg1.Evaluate(t).(type) {
		case string:
			m = typ
		case query:
			node := typ.Select(t)
			if node == nil {
				return false
			}
			m = node.Value()
		default:
			panic(errors.New("ends-with() function argument type must be string"))
		}
		n, ok = arg2.Evaluate(t).(string)
		if !ok {
			panic(errors.New("ends-with() function argument type must be string"))
		}
		return strings.HasSuffix(m, n)
	}
}

// containsFunc is a XPath functions contains(string or @attr, string).
func containsFunc(arg1, arg2 query) func(query, iterator) interface{} {
	return func(q query, t iterator) interface{} {
		var (
			m, n string
			ok   bool
		)

		switch typ := arg1.Evaluate(t).(type) {
		case string:
			m = typ
		case query:
			node := typ.Select(t)
			if node == nil {
				return false
			}
			m = node.Value()
		default:
			panic(errors.New("contains() function argument type must be string"))
		}

		n, ok = arg2.Evaluate(t).(string)
		if !ok {
			panic(errors.New("contains() function argument type must be string"))
		}

		return strings.Contains(m, n)
	}
}

var (
	regnewline  = regexp.MustCompile(`[\r\n\t]`)
	regseqspace = regexp.MustCompile(`\s{2,}`)
)

// normalizespaceFunc is XPath functions normalize-space(string?)
func normalizespaceFunc(q query, t iterator) interface{} {
	var m string
	switch typ := q.Evaluate(t).(type) {
	case string:
		m = typ
	case query:
		node := typ.Select(t)
		if node == nil {
			return ""
		}
		m = node.Value()
	}
	m = strings.TrimSpace(m)
	m = regnewline.ReplaceAllString(m, " ")
	m = regseqspace.ReplaceAllString(m, " ")
	return m
}

// substringFunc is XPath functions substring function returns a part of a given string.
func substringFunc(arg1, arg2, arg3 query) func(query, iterator) interface{} {
	return func(q query, t iterator) interface{} {
		var m string
		switch typ := arg1.Evaluate(t).(type) {
		case string:
			m = typ
		case query:
			node := typ.Select(t)
			if node == nil {
				return ""
			}
			m = node.Value()
		}

		var start, length float64
		var ok bool

		if start, ok = arg2.Evaluate(t).(float64); !ok {
			panic(errors.New("substring() function first argument type must be int"))
		} else if start < 1 {
			panic(errors.New("substring() function first argument type must be >= 1"))
		}
		start--
		if arg3 != nil {
			if length, ok = arg3.Evaluate(t).(float64); !ok {
				panic(errors.New("substring() function second argument type must be int"))
			}
		}
		if (len(m) - int(start)) < int(length) {
			panic(errors.New("substring() function start and length argument out of range"))
		}
		if length > 0 {
			return m[int(start):int(length+start)]
		}
		return m[int(start):]
	}
}

// substringIndFunc is XPath functions substring-before/substring-after function returns a part of a given string.
func substringIndFunc(arg1, arg2 query, after bool) func(query, iterator) interface{} {
	return func(q query, t iterator) interface{} {
		var str string
		switch v := arg1.Evaluate(t).(type) {
		case string:
			str = v
		case query:
			node := v.Select(t)
			if node == nil {
				return ""
			}
			str = node.Value()
		}
		var word string
		switch v := arg2.Evaluate(t).(type) {
		case string:
			word = v
		case query:
			node := v.Select(t)
			if node == nil {
				return ""
			}
			word = node.Value()
		}
		if word == "" {
			return ""
		}

		i := strings.Index(str, word)
		if i < 0 {
			return ""
		}
		if after {
			return str[i+len(word):]
		}
		return str[:i]
	}
}

// stringLengthFunc is XPATH string-length( [string] ) function that returns a number
// equal to the number of characters in a given string.
func stringLengthFunc(arg1 query) func(query, iterator) interface{} {
	return func(q query, t iterator) interface{} {
		switch v := arg1.Evaluate(t).(type) {
		case string:
			return float64(len(v))
		case query:
			node := v.Select(t)
			if node == nil {
				break
			}
			return float64(len(node.Value()))
		}
		return float64(0)
	}
}

// translateFunc is XPath functions translate() function returns a replaced string.
func translateFunc(arg1, arg2, arg3 query) func(query, iterator) interface{} {
	return func(q query, t iterator) interface{} {
		str := asString(t, arg1.Evaluate(t))
		src := asString(t, arg2.Evaluate(t))
		dst := asString(t, arg3.Evaluate(t))

		var replace []string
		for i, s := range src {
			d := ""
			if i < len(dst) {
				d = string(dst[i])
			}
			replace = append(replace, string(s), d)
		}
		return strings.NewReplacer(replace...).Replace(str)
	}
}

// notFunc is XPATH functions not(expression) function operation.
func notFunc(q query, t iterator) interface{} {
	switch v := q.Evaluate(t).(type) {
	case bool:
		return !v
	case query:
		node := v.Select(t)
		return node == nil
	default:
		return false
	}
}

// concatFunc is the concat function concatenates two or more
// strings and returns the resulting string.
// concat( string1 , string2 [, stringn]* )
func concatFunc(args ...query) func(query, iterator) interface{} {
	return func(q query, t iterator) interface{} {
		var a []string
		for _, v := range args {
			switch v := v.Evaluate(t).(type) {
			case string:
				a = append(a, v)
			case query:
				node := v.Select(t)
				if node != nil {
					a = append(a, node.Value())
				}
			}
		}
		return strings.Join(a, "")
	}
}
