package ast

import (
	"fmt"
	"reflect"
)

// Scope is the interface used to look up variables and functions while
// evaluating. How these functions/variables are defined are up to the caller.
type Scope interface {
	LookupFunc(string) (Function, bool)
	LookupVar(string) (Variable, bool)
}

// Variable is a variable value for execution given as input to the engine.
// It records the value of a variables along with their type.
type Variable struct {
	Value interface{}
	Type  Type
}

// NewVariable creates a new Variable for the given value. This will
// attempt to infer the correct type. If it can't, an error will be returned.
func NewVariable(v interface{}) (result Variable, err error) {
	switch v := reflect.ValueOf(v); v.Kind() {
	case reflect.String:
		result.Type = TypeString
	default:
		err = fmt.Errorf("Unknown type: %s", v.Kind())
	}

	result.Value = v
	return
}

// String implements Stringer on Variable, displaying the type and value
// of the Variable.
func (v Variable) String() string {
	return fmt.Sprintf("{Variable (%s): %+v}", v.Type, v.Value)
}

// Function defines a function that can be executed by the engine.
// The type checker will validate that the proper types will be called
// to the callback.
type Function struct {
	// ArgTypes is the list of types in argument order. These are the
	// required arguments.
	ArgTypes []Type

	// Either ReturnType *or* ReturnTypeFunc decide the type of the returned
	// value. The Callback MUST return this type. Setting both attributes
	// is invalid usage.
	ReturnType     Type
	ReturnTypeFunc ReturnTypeFunc

	// Variadic, if true, says that this function is variadic, meaning
	// it takes a variable number of arguments. In this case, the
	// VariadicType must be set.
	Variadic     bool
	VariadicType Type

	// Either Callback or CallbackTyped are called as the implementation of
	// the function. Both recieve a slice interface values of an appropriate
	// dynamic type for the call arguments, while CallbackTyped additionally
	// recieves the required result type, for easier implementation of
	// type-generic functions without duplicating the logic in ReturnTypeFunc.
	//
	// The argument types are guaranteed by the type checker to match what is
	// described by ArgTypes, ReturnTypeFunc and VariadicType.
	// The length of the args is strictly == len(ArgTypes) unless Varidiac
	// is true, in which case its >= len(ArgTypes).
	//
	// The value returned MUST confirm to the function's return type, whether
	// determined by ReturnType or ReturnTypeFunc.
	//
	// Setting both Callback and CallbackTyped is invalid usage.
	Callback      func([]interface{}) (interface{}, error)
	CallbackTyped func(args []interface{}, returnType Type) (interface{}, error)
}

// ReturnTypeFunc is a function type used to decide the return type of a
// function based on its argument types.
//
// The given argument types are those of the actual *call*, not the types
// declared in ArgTypes and VariadicType. This allows the definition of
// functions that work with TypeList and TypeMap in a generic way for all
// element types, and other similar interesting cases.
//
// Function must either return a concrete Type or an user-oriented error
// that explains why the given combination of argument types are not
// acceptable. If an error is not returned then the Function's Callback
// MUST be able to accept the given argument types without crashing,
// and produce a value of the given return type.
//
// ReturnTypeFunc is called only if the given ArgTypes and VariadicType
// match the given arguments, so it need only check additional
// unusual rules that cannot be expressed as static types. Use TypeAny
// (or TypeList{TypeAny}, etc) in ArgTypes to bypass the simple type
// checking for certain arguments where more complex rules are required.
type ReturnTypeFunc func(argTypes []Type) (Type, error)

// BasicScope is a simple scope that looks up variables and functions
// using a map.
type BasicScope struct {
	FuncMap map[string]Function
	VarMap  map[string]Variable
}

func (s *BasicScope) LookupFunc(n string) (Function, bool) {
	if s == nil {
		return Function{}, false
	}

	v, ok := s.FuncMap[n]
	return v, ok
}

func (s *BasicScope) LookupVar(n string) (Variable, bool) {
	if s == nil {
		return Variable{}, false
	}

	v, ok := s.VarMap[n]
	return v, ok
}
