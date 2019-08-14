package gval

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Evaluable evaluates given parameter
type Evaluable func(c context.Context, parameter interface{}) (interface{}, error)

//EvalInt evaluates given parameter to an int
func (e Evaluable) EvalInt(c context.Context, parameter interface{}) (int, error) {
	v, err := e(c, parameter)
	if err != nil {
		return 0, err
	}

	f, ok := convertToFloat(v)
	if !ok {
		return 0, fmt.Errorf("expected number but got %v (%T)", v, v)
	}
	return int(f), nil
}

//EvalFloat64 evaluates given parameter to an int
func (e Evaluable) EvalFloat64(c context.Context, parameter interface{}) (float64, error) {
	v, err := e(c, parameter)
	if err != nil {
		return 0, err
	}

	f, ok := convertToFloat(v)
	if !ok {
		return 0, fmt.Errorf("expected number but got %v (%T)", v, v)
	}
	return f, nil
}

//EvalBool evaluates given parameter to a bool
func (e Evaluable) EvalBool(c context.Context, parameter interface{}) (bool, error) {
	v, err := e(c, parameter)
	if err != nil {
		return false, err
	}

	b, ok := convertToBool(v)
	if !ok {
		return false, fmt.Errorf("expected bool but got %v (%T)", v, v)
	}
	return b, nil
}

//EvalString evaluates given parameter to a string
func (e Evaluable) EvalString(c context.Context, parameter interface{}) (string, error) {
	o, err := e(c, parameter)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", o), nil
}

//Const Evaluable represents given constant
func (*Parser) Const(value interface{}) Evaluable {
	return constant(value)
}

func constant(value interface{}) Evaluable {
	return func(c context.Context, v interface{}) (interface{}, error) {
		return value, nil
	}
}

//Var Evaluable represents value at given path.
//It supports with default language VariableSelector:
//	map[interface{}]interface{},
//	map[string]interface{} and
// 	[]interface{} and via reflect
//	struct fields,
//	struct methods,
//	slices and
//  map with int or string key.
func (p *Parser) Var(path ...Evaluable) Evaluable {
	if p.Language.selector == nil {
		return variable(path)
	}
	return p.Language.selector(path)
}

// Evaluables is a slice of Evaluable.
type Evaluables []Evaluable

// EvalStrings evaluates given parameter to a string slice
func (evs Evaluables) EvalStrings(c context.Context, parameter interface{}) ([]string, error) {
	strs := make([]string, len(evs))
	for i, p := range evs {
		k, err := p.EvalString(c, parameter)
		if err != nil {
			return nil, err
		}
		strs[i] = k
	}
	return strs, nil
}

func variable(path Evaluables) Evaluable {
	return func(c context.Context, v interface{}) (interface{}, error) {
		keys, err := path.EvalStrings(c, v)
		if err != nil {
			return nil, err
		}
		for i, k := range keys {
			switch o := v.(type) {
			case map[interface{}]interface{}:
				v = o[k]
				continue
			case map[string]interface{}:
				v = o[k]
				continue
			case []interface{}:
				if i, err := strconv.Atoi(k); err == nil && i >= 0 && len(o) > i {
					v = o[i]
					continue
				}
			default:
				var ok bool
				v, ok = reflectSelect(k, o)
				if !ok {
					return nil, fmt.Errorf("unknown parameter %s", strings.Join(keys[:i+1], "."))
				}
			}
		}
		return v, nil
	}
}

func reflectSelect(key string, value interface{}) (selection interface{}, ok bool) {
	vv := reflect.ValueOf(value)
	vvElem := resolvePotentialPointer(vv)

	switch vvElem.Kind() {
	case reflect.Map:
		mapKey, ok := reflectConvertTo(vv.Type().Key().Kind(), key)
		if !ok {
			return nil, false
		}

		vvElem = vv.MapIndex(reflect.ValueOf(mapKey))
		vvElem = resolvePotentialPointer(vvElem)

		if vvElem.IsValid() {
			return vvElem.Interface(), true
		}
	case reflect.Slice:
		if i, err := strconv.Atoi(key); err == nil && i >= 0 && vv.Len() > i {
			vvElem = resolvePotentialPointer(vv.Index(i))
			return vvElem.Interface(), true
		}
	case reflect.Struct:
		field := vvElem.FieldByName(key)
		if field.IsValid() {
			return field.Interface(), true
		}

		method := vv.MethodByName(key)
		if method.IsValid() {
			return method.Interface(), true
		}
	}
	return nil, false
}

func resolvePotentialPointer(value reflect.Value) reflect.Value {
	if value.Kind() == reflect.Ptr {
		return value.Elem()
	}
	return value
}

func reflectConvertTo(k reflect.Kind, value string) (interface{}, bool) {
	switch k {
	case reflect.String:
		return value, true
	case reflect.Int:
		if i, err := strconv.Atoi(value); err == nil {
			return i, true
		}
	}
	return nil, false
}

func (*Parser) callFunc(fun function, args ...Evaluable) Evaluable {
	return func(c context.Context, v interface{}) (ret interface{}, err error) {
		a := make([]interface{}, len(args))
		for i, arg := range args {
			ai, err := arg(c, v)
			if err != nil {
				return nil, err
			}
			a[i] = ai
		}
		return fun(a...)
	}
}

func (*Parser) callEvaluable(fullname string, fun Evaluable, args ...Evaluable) Evaluable {
	return func(c context.Context, v interface{}) (ret interface{}, err error) {
		f, err := fun(c, v)

		if err != nil {
			return nil, fmt.Errorf("could not call function: %v", err)
		}

		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("failed to execute function '%s': %s", fullname, r)
				ret = nil
			}
		}()

		ff := reflect.ValueOf(f)

		if ff.Kind() != reflect.Func {
			return nil, fmt.Errorf("could not call '%s' type %T", fullname, f)
		}

		a := make([]reflect.Value, len(args))
		for i := range args {
			arg, err := args[i](c, v)
			if err != nil {
				return nil, err
			}
			a[i] = reflect.ValueOf(arg)
		}

		rr := ff.Call(a)

		r := make([]interface{}, len(rr))
		for i, e := range rr {
			r[i] = e.Interface()
		}

		errorInterface := reflect.TypeOf((*error)(nil)).Elem()
		if len(r) > 0 && ff.Type().Out(len(r)-1).Implements(errorInterface) {
			if r[len(r)-1] != nil {
				err = r[len(r)-1].(error)
			}
			r = r[0 : len(r)-1]
		}

		switch len(r) {
		case 0:
			return err, nil
		case 1:
			return r[0], err
		default:
			return r, err
		}
	}
}

//IsConst returns if the Evaluable is a Parser.Const() value
func (e Evaluable) IsConst() bool {
	pc := reflect.ValueOf(constant(nil)).Pointer()
	pe := reflect.ValueOf(e).Pointer()
	return pc == pe
}

func regEx(a, b Evaluable) (Evaluable, error) {
	if !b.IsConst() {
		return func(c context.Context, o interface{}) (interface{}, error) {
			a, err := a.EvalString(c, o)
			if err != nil {
				return nil, err
			}
			b, err := b.EvalString(c, o)
			if err != nil {
				return nil, err
			}
			matched, err := regexp.MatchString(b, a)
			return matched, err
		}, nil
	}
	s, err := b.EvalString(nil, nil)
	if err != nil {
		return nil, err
	}
	regex, err := regexp.Compile(s)
	if err != nil {
		return nil, err
	}
	return func(c context.Context, v interface{}) (interface{}, error) {
		s, err := a.EvalString(c, v)
		if err != nil {
			return nil, err
		}
		return regex.MatchString(s), nil
	}, nil
}

func notRegEx(a, b Evaluable) (Evaluable, error) {
	if !b.IsConst() {
		return func(c context.Context, o interface{}) (interface{}, error) {
			a, err := a.EvalString(c, o)
			if err != nil {
				return nil, err
			}
			b, err := b.EvalString(c, o)
			if err != nil {
				return nil, err
			}
			matched, err := regexp.MatchString(b, a)
			return !matched, err
		}, nil
	}
	s, err := b.EvalString(nil, nil)
	if err != nil {
		return nil, err
	}
	regex, err := regexp.Compile(s)
	if err != nil {
		return nil, err
	}
	return func(c context.Context, v interface{}) (interface{}, error) {
		s, err := a.EvalString(c, v)
		if err != nil {
			return nil, err
		}
		return !regex.MatchString(s), nil
	}, nil
}
