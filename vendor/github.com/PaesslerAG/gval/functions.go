package gval

import (
	"fmt"
	"reflect"
)

type function func(arguments ...interface{}) (interface{}, error)

func toFunc(f interface{}) function {
	if f, ok := f.(func(arguments ...interface{}) (interface{}, error)); ok {
		return function(f)
	}
	return func(args ...interface{}) (interface{}, error) {
		fun := reflect.ValueOf(f)
		t := fun.Type()

		in, err := createCallArguments(t, args)
		if err != nil {
			return nil, err
		}
		out := fun.Call(in)

		r := make([]interface{}, len(out))
		for i, e := range out {
			r[i] = e.Interface()
		}

		err = nil
		errorInterface := reflect.TypeOf((*error)(nil)).Elem()
		if len(r) > 0 && t.Out(len(r)-1).Implements(errorInterface) {
			if r[len(r)-1] != nil {
				err = r[len(r)-1].(error)
			}
			r = r[0 : len(r)-1]
		}

		switch len(r) {
		case 0:
			return nil, err
		case 1:
			return r[0], err
		default:
			return r, err
		}
	}
}

func createCallArguments(t reflect.Type, args []interface{}) ([]reflect.Value, error) {
	variadic := t.IsVariadic()
	numIn := t.NumIn()

	if (!variadic && len(args) != numIn) || (variadic && len(args) < numIn-1) {
		return nil, fmt.Errorf("invalid number of parameters")
	}

	in := make([]reflect.Value, len(args))
	var inType reflect.Type
	for i, arg := range args {
		if !variadic || i < numIn-1 {
			inType = t.In(i)
		} else if i == numIn-1 {
			inType = t.In(numIn - 1).Elem()
		}
		argVal := reflect.ValueOf(arg)
		if arg == nil || !argVal.Type().AssignableTo(inType) {
			return nil, fmt.Errorf("expected type %s for parameter %d but got %T",
				inType.String(), i, arg)
		}
		in[i] = argVal
	}
	return in, nil
}
