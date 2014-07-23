package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// We really need to replace this with a real parser.
var funcRegexp *regexp.Regexp = regexp.MustCompile(
	`(?i)([a-z0-9_]+)\(\s*(?:([.a-z0-9_]+)\s*,\s*)*([.a-z0-9_]+)\s*\)`)

// Interpolation is something that can be contained in a "${}" in a
// configuration value.
//
// Interpolations might be simple variable references, or it might be
// function calls, or even nested function calls.
type Interpolation interface {
	Interpolate(map[string]string) (string, error)
	Variables() map[string]InterpolatedVariable
}

// InterpolationFunc is the function signature for implementing
// callable functions in Terraform configurations.
type InterpolationFunc func(map[string]string, ...string) (string, error)

// An InterpolatedVariable is a variable reference within an interpolation.
//
// Implementations of this interface represents various sources where
// variables can come from: user variables, resources, etc.
type InterpolatedVariable interface {
	FullKey() string
}

// FunctionInterpolation is an Interpolation that executes a function
// with some variable number of arguments to generate a value.
type FunctionInterpolation struct {
	Func InterpolationFunc
	Args []Interpolation
}

// LiteralInterpolation implements Interpolation for literals. Ex:
// ${"foo"} will equal "foo".
type LiteralInterpolation struct {
	Literal string
}

// VariableInterpolation implements Interpolation for simple variable
// interpolation. Ex: "${var.foo}" or "${aws_instance.foo.bar}"
type VariableInterpolation struct {
	Variable InterpolatedVariable
}

// A ResourceVariable is a variable that is referencing the field
// of a resource, such as "${aws_instance.foo.ami}"
type ResourceVariable struct {
	Type  string // Resource type, i.e. "aws_instance"
	Name  string // Resource name
	Field string // Resource field

	Multi bool // True if multi-variable: aws_instance.foo.*.id
	Index int  // Index for multi-variable: aws_instance.foo.1.id == 1

	key string
}

// A UserVariable is a variable that is referencing a user variable
// that is inputted from outside the configuration. This looks like
// "${var.foo}"
type UserVariable struct {
	Name string
	Elem string

	key string
}

func NewInterpolatedVariable(v string) (InterpolatedVariable, error) {
	if !strings.HasPrefix(v, "var.") {
		return NewResourceVariable(v)
	}

	return NewUserVariable(v)
}

func (i *FunctionInterpolation) Interpolate(
	vs map[string]string) (string, error) {
	args := make([]string, len(i.Args))
	for idx, a := range i.Args {
		v, err := a.Interpolate(vs)
		if err != nil {
			return "", err
		}

		args[idx] = v
	}

	return i.Func(vs, args...)
}

func (i *FunctionInterpolation) GoString() string {
	return fmt.Sprintf("*%#v", *i)
}

func (i *FunctionInterpolation) Variables() map[string]InterpolatedVariable {
	result := make(map[string]InterpolatedVariable)
	for _, a := range i.Args {
		for k, v := range a.Variables() {
			result[k] = v
		}
	}

	return result
}

func (i *LiteralInterpolation) Interpolate(
	map[string]string) (string, error) {
	return i.Literal, nil
}

func (i *LiteralInterpolation) Variables() map[string]InterpolatedVariable {
	return nil
}

func (i *VariableInterpolation) Interpolate(
	vs map[string]string) (string, error) {
	v, ok := vs[i.Variable.FullKey()]
	if !ok {
		return "", fmt.Errorf(
			"%s: value for variable not found",
			i.Variable.FullKey())
	}

	return v, nil
}

func (i *VariableInterpolation) GoString() string {
	return fmt.Sprintf("*%#v", *i)
}

func (i *VariableInterpolation) Variables() map[string]InterpolatedVariable {
	return map[string]InterpolatedVariable{i.Variable.FullKey(): i.Variable}
}

func NewResourceVariable(key string) (*ResourceVariable, error) {
	parts := strings.SplitN(key, ".", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf(
			"%s: resource variables must be three parts: type.name.attr",
			key)
	}

	field := parts[2]
	multi := false
	var index int

	if idx := strings.Index(field, "."); idx != -1 {
		indexStr := field[:idx]
		multi = indexStr == "*"
		index = -1

		if !multi {
			indexInt, err := strconv.ParseInt(indexStr, 0, 0)
			if err == nil {
				multi = true
				index = int(indexInt)
			}
		}

		if multi {
			field = field[idx+1:]
		}
	}

	return &ResourceVariable{
		Type:  parts[0],
		Name:  parts[1],
		Field: field,
		Multi: multi,
		Index: index,
		key:   key,
	}, nil
}

func (v *ResourceVariable) ResourceId() string {
	return fmt.Sprintf("%s.%s", v.Type, v.Name)
}

func (v *ResourceVariable) FullKey() string {
	return v.key
}

func NewUserVariable(key string) (*UserVariable, error) {
	name := key[len("var."):]
	elem := ""
	if idx := strings.Index(name, "."); idx > -1 {
		elem = name[idx+1:]
		name = name[:idx]
	}

	return &UserVariable{
		key: key,

		Name: name,
		Elem: elem,
	}, nil
}

func (v *UserVariable) FullKey() string {
	return v.key
}

func (v *UserVariable) GoString() string {
	return fmt.Sprintf("*%#v", *v)
}
