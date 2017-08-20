package xpath

import (
	"fmt"
	"reflect"
	"strconv"
)

// The XPath number operator function list.

// valueType is a return value type.
type valueType int

const (
	booleanType valueType = iota
	numberType
	stringType
	nodeSetType
)

func getValueType(i interface{}) valueType {
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Float64:
		return numberType
	case reflect.String:
		return stringType
	case reflect.Bool:
		return booleanType
	default:
		if _, ok := i.(query); ok {
			return nodeSetType
		}
	}
	panic(fmt.Errorf("xpath unknown value type: %v", v.Kind()))
}

type logical func(iterator, string, interface{}, interface{}) bool

var logicalFuncs = [][]logical{
	{cmpBooleanBoolean, nil, nil, nil},
	{nil, cmpNumericNumeric, cmpNumericString, cmpNumericNodeSet},
	{nil, cmpStringNumeric, cmpStringString, cmpStringNodeSet},
	{nil, cmpNodeSetNumeric, cmpNodeSetString, cmpNodeSetNodeSet},
}

// number vs number
func cmpNumberNumberF(op string, a, b float64) bool {
	switch op {
	case "=":
		return a == b
	case ">":
		return a > b
	case "<":
		return a < b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	case "!=":
		return a != b
	}
	return false
}

// string vs string
func cmpStringStringF(op string, a, b string) bool {
	switch op {
	case "=":
		return a == b
	case ">":
		return a > b
	case "<":
		return a < b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	case "!=":
		return a != b
	}
	return false
}

func cmpBooleanBooleanF(op string, a, b bool) bool {
	switch op {
	case "or":
		return a || b
	case "and":
		return a && b
	}
	return false
}

func cmpNumericNumeric(t iterator, op string, m, n interface{}) bool {
	a := m.(float64)
	b := n.(float64)
	return cmpNumberNumberF(op, a, b)
}

func cmpNumericString(t iterator, op string, m, n interface{}) bool {
	a := m.(float64)
	b := n.(string)
	num, err := strconv.ParseFloat(b, 64)
	if err != nil {
		panic(err)
	}
	return cmpNumberNumberF(op, a, num)
}

func cmpNumericNodeSet(t iterator, op string, m, n interface{}) bool {
	a := m.(float64)
	b := n.(query)

	for {
		node := b.Select(t)
		if node == nil {
			break
		}
		num, err := strconv.ParseFloat(node.Value(), 64)
		if err != nil {
			panic(err)
		}
		if cmpNumberNumberF(op, a, num) {
			return true
		}
	}
	return false
}

func cmpNodeSetNumeric(t iterator, op string, m, n interface{}) bool {
	a := m.(query)
	b := n.(float64)
	for {
		node := a.Select(t)
		if node == nil {
			break
		}
		num, err := strconv.ParseFloat(node.Value(), 64)
		if err != nil {
			panic(err)
		}
		if cmpNumberNumberF(op, num, b) {
			return true
		}
	}
	return false
}

func cmpNodeSetString(t iterator, op string, m, n interface{}) bool {
	a := m.(query)
	b := n.(string)
	for {
		node := a.Select(t)
		if node == nil {
			break
		}
		if cmpStringStringF(op, b, node.Value()) {
			return true
		}
	}
	return false
}

func cmpNodeSetNodeSet(t iterator, op string, m, n interface{}) bool {
	return false
}

func cmpStringNumeric(t iterator, op string, m, n interface{}) bool {
	a := m.(string)
	b := n.(float64)
	num, err := strconv.ParseFloat(a, 64)
	if err != nil {
		panic(err)
	}
	return cmpNumberNumberF(op, b, num)
}

func cmpStringString(t iterator, op string, m, n interface{}) bool {
	a := m.(string)
	b := n.(string)
	return cmpStringStringF(op, a, b)
}

func cmpStringNodeSet(t iterator, op string, m, n interface{}) bool {
	a := m.(string)
	b := n.(query)
	for {
		node := b.Select(t)
		if node == nil {
			break
		}
		if cmpStringStringF(op, a, node.Value()) {
			return true
		}
	}
	return false
}

func cmpBooleanBoolean(t iterator, op string, m, n interface{}) bool {
	a := m.(bool)
	b := n.(bool)
	return cmpBooleanBooleanF(op, a, b)
}

// eqFunc is an `=` operator.
func eqFunc(t iterator, m, n interface{}) interface{} {
	t1 := getValueType(m)
	t2 := getValueType(n)
	return logicalFuncs[t1][t2](t, "=", m, n)
}

// gtFunc is an `>` operator.
func gtFunc(t iterator, m, n interface{}) interface{} {
	t1 := getValueType(m)
	t2 := getValueType(n)
	return logicalFuncs[t1][t2](t, ">", m, n)
}

// geFunc is an `>=` operator.
func geFunc(t iterator, m, n interface{}) interface{} {
	t1 := getValueType(m)
	t2 := getValueType(n)
	return logicalFuncs[t1][t2](t, ">=", m, n)
}

// ltFunc is an `<` operator.
func ltFunc(t iterator, m, n interface{}) interface{} {
	t1 := getValueType(m)
	t2 := getValueType(n)
	return logicalFuncs[t1][t2](t, "<", m, n)
}

// leFunc is an `<=` operator.
func leFunc(t iterator, m, n interface{}) interface{} {
	t1 := getValueType(m)
	t2 := getValueType(n)
	return logicalFuncs[t1][t2](t, "<=", m, n)
}

// neFunc is an `!=` operator.
func neFunc(t iterator, m, n interface{}) interface{} {
	t1 := getValueType(m)
	t2 := getValueType(n)
	return logicalFuncs[t1][t2](t, "!=", m, n)
}

// orFunc is an `or` operator.
var orFunc = func(t iterator, m, n interface{}) interface{} {
	t1 := getValueType(m)
	t2 := getValueType(n)
	return logicalFuncs[t1][t2](t, "or", m, n)
}

func numericExpr(m, n interface{}, cb func(float64, float64) float64) float64 {
	typ := reflect.TypeOf(float64(0))
	a := reflect.ValueOf(m).Convert(typ)
	b := reflect.ValueOf(n).Convert(typ)
	return cb(a.Float(), b.Float())
}

// plusFunc is an `+` operator.
var plusFunc = func(m, n interface{}) interface{} {
	return numericExpr(m, n, func(a, b float64) float64 {
		return a + b
	})
}

// minusFunc is an `-` operator.
var minusFunc = func(m, n interface{}) interface{} {
	return numericExpr(m, n, func(a, b float64) float64 {
		return a - b
	})
}

// mulFunc is an `*` operator.
var mulFunc = func(m, n interface{}) interface{} {
	return numericExpr(m, n, func(a, b float64) float64 {
		return a * b
	})
}

// divFunc is an `DIV` operator.
var divFunc = func(m, n interface{}) interface{} {
	return numericExpr(m, n, func(a, b float64) float64 {
		return a / b
	})
}

// modFunc is an 'MOD' operator.
var modFunc = func(m, n interface{}) interface{} {
	return numericExpr(m, n, func(a, b float64) float64 {
		return float64(int(a) % int(b))
	})
}
