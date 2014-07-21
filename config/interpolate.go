package config

import (
	"fmt"
	"strconv"
	"strings"
)

// Interpolation is something that can be contained in a "${}" in a
// configuration value.
//
// Interpolations might be simple variable references, or it might be
// function calls, or even nested function calls.
type Interpolation interface {
	FullString() string
	Interpolate(map[string]string) (string, error)
	Variables() map[string]InterpolatedVariable
}

// An InterpolatedVariable is a variable reference within an interpolation.
//
// Implementations of this interface represents various sources where
// variables can come from: user variables, resources, etc.
type InterpolatedVariable interface {
	FullKey() string
}

// VariableInterpolation implements Interpolation for simple variable
// interpolation. Ex: "${var.foo}" or "${aws_instance.foo.bar}"
type VariableInterpolation struct {
	Variable InterpolatedVariable

	key string
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

	key string
}

// A UserMapVariable is a variable that is referencing a user
// variable that is a map. This looks like "${var.amis.us-east-1}"
type UserMapVariable struct {
	Name string
	Elem string

	key string
}

// NewInterpolation takes some string and returns the valid
// Interpolation associated with it, or error if a valid
// interpolation could not be found or the interpolation itself
// is invalid.
func NewInterpolation(v string) (Interpolation, error) {
	if idx := strings.Index(v, "."); idx >= 0 {
		v, err := NewInterpolatedVariable(v)
		if err != nil {
			return nil, err
		}

		return &VariableInterpolation{
			Variable: v,
			key:      v.FullKey(),
		}, nil
	}

	return nil, fmt.Errorf(
		"Interpolation '%s' is not a valid interpolation. " +
			"Please check your syntax and try again.")
}

func NewInterpolatedVariable(v string) (InterpolatedVariable, error) {
	if !strings.HasPrefix(v, "var.") {
		return NewResourceVariable(v)
	}

	varKey := v[len("var."):]
	if strings.Index(varKey, ".") == -1 {
		return NewUserVariable(v)
	} else {
		return NewUserMapVariable(v)
	}
}

func (i *VariableInterpolation) FullString() string {
	return i.key
}

func (i *VariableInterpolation) Interpolate(
	vs map[string]string) (string, error) {
	return vs[i.key], nil
}

func (i *VariableInterpolation) Variables() map[string]InterpolatedVariable {
	return map[string]InterpolatedVariable{i.key: i.Variable}
}

func NewResourceVariable(key string) (*ResourceVariable, error) {
	parts := strings.SplitN(key, ".", 3)
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
	return &UserVariable{
		key:  key,
		Name: name,
	}, nil
}

func (v *UserVariable) FullKey() string {
	return v.key
}

func NewUserMapVariable(key string) (*UserMapVariable, error) {
	name := key[len("var."):]
	idx := strings.Index(name, ".")
	if idx == -1 {
		return nil, fmt.Errorf("not a user map variable: %s", key)
	}

	elem := name[idx+1:]
	name = name[:idx]
	return &UserMapVariable{
		Name: name,
		Elem: elem,

		key: key,
	}, nil
}

func (v *UserMapVariable) FullKey() string {
	return v.key
}

func (v *UserMapVariable) GoString() string {
	return fmt.Sprintf("%#v", *v)
}
