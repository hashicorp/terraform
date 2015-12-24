package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/config/lang/ast"
)

// An InterpolatedVariable is a variable reference within an interpolation.
//
// Implementations of this interface represents various sources where
// variables can come from: user variables, resources, etc.
type InterpolatedVariable interface {
	FullKey() string
}

// CountVariable is a variable for referencing information about
// the count.
type CountVariable struct {
	Type CountValueType
	key  string
}

// CountValueType is the type of the count variable that is referenced.
type CountValueType byte

const (
	CountValueInvalid CountValueType = iota
	CountValueIndex
)

// A ModuleVariable is a variable that is referencing the output
// of a module, such as "${module.foo.bar}"
type ModuleVariable struct {
	Name  string
	Field string
	key   string
}

// A PathVariable is a variable that references path information about the
// module.
type PathVariable struct {
	Type PathValueType
	key  string
}

type PathValueType byte

const (
	PathValueInvalid PathValueType = iota
	PathValueCwd
	PathValueModule
	PathValueRoot
)

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

// SelfVariable is a variable that is referencing the same resource
// it is running on: "${self.address}"
type SelfVariable struct {
	Field string

	key string
}

// SimpleVariable is an unprefixed variable, which can show up when users have
// strings they are passing down to resources that use interpolation
// internally. The template_file resource is an example of this.
type SimpleVariable struct {
	Key string
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
	if strings.HasPrefix(v, "count.") {
		return NewCountVariable(v)
	} else if strings.HasPrefix(v, "path.") {
		return NewPathVariable(v)
	} else if strings.HasPrefix(v, "self.") {
		return NewSelfVariable(v)
	} else if strings.HasPrefix(v, "var.") {
		return NewUserVariable(v)
	} else if strings.HasPrefix(v, "module.") {
		return NewModuleVariable(v)
	} else if !strings.ContainsRune(v, '.') {
		return NewSimpleVariable(v)
	} else {
		return NewResourceVariable(v)
	}
}

func NewCountVariable(key string) (*CountVariable, error) {
	var fieldType CountValueType
	parts := strings.SplitN(key, ".", 2)
	switch parts[1] {
	case "index":
		fieldType = CountValueIndex
	}

	return &CountVariable{
		Type: fieldType,
		key:  key,
	}, nil
}

func (c *CountVariable) FullKey() string {
	return c.key
}

func NewModuleVariable(key string) (*ModuleVariable, error) {
	parts := strings.SplitN(key, ".", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf(
			"%s: module variables must be three parts: module.name.attr",
			key)
	}

	return &ModuleVariable{
		Name:  parts[1],
		Field: parts[2],
		key:   key,
	}, nil
}

func (v *ModuleVariable) FullKey() string {
	return v.key
}

func NewPathVariable(key string) (*PathVariable, error) {
	var fieldType PathValueType
	parts := strings.SplitN(key, ".", 2)
	switch parts[1] {
	case "cwd":
		fieldType = PathValueCwd
	case "module":
		fieldType = PathValueModule
	case "root":
		fieldType = PathValueRoot
	}

	return &PathVariable{
		Type: fieldType,
		key:  key,
	}, nil
}

func (v *PathVariable) FullKey() string {
	return v.key
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

func NewSelfVariable(key string) (*SelfVariable, error) {
	field := key[len("self."):]

	return &SelfVariable{
		Field: field,

		key: key,
	}, nil
}

func (v *SelfVariable) FullKey() string {
	return v.key
}

func (v *SelfVariable) GoString() string {
	return fmt.Sprintf("*%#v", *v)
}

func NewSimpleVariable(key string) (*SimpleVariable, error) {
	return &SimpleVariable{key}, nil
}

func (v *SimpleVariable) FullKey() string {
	return v.Key
}

func (v *SimpleVariable) GoString() string {
	return fmt.Sprintf("*%#v", *v)
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

// DetectVariables takes an AST root and returns all the interpolated
// variables that are detected in the AST tree.
func DetectVariables(root ast.Node) ([]InterpolatedVariable, error) {
	var result []InterpolatedVariable
	var resultErr error

	// Visitor callback
	fn := func(n ast.Node) ast.Node {
		if resultErr != nil {
			return n
		}

		vn, ok := n.(*ast.VariableAccess)
		if !ok {
			return n
		}

		v, err := NewInterpolatedVariable(vn.Name)
		if err != nil {
			resultErr = err
			return n
		}

		result = append(result, v)
		return n
	}

	// Visitor pattern
	root.Accept(fn)

	if resultErr != nil {
		return nil, resultErr
	}

	return result, nil
}
