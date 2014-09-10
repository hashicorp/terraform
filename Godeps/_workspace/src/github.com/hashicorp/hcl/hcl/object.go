package hcl

import (
	"fmt"
	"strings"
)

// ValueType is an enum represnting the type of a value in
// a LiteralNode.
type ValueType byte

const (
	ValueTypeUnknown ValueType = iota
	ValueTypeFloat
	ValueTypeInt
	ValueTypeString
	ValueTypeBool
	ValueTypeNil
	ValueTypeList
	ValueTypeObject
)

// Object represents any element of HCL: an object itself, a list,
// a literal, etc.
type Object struct {
	Key   string
	Type  ValueType
	Value interface{}
	Next  *Object
}

// GoStrig is an implementation of the GoStringer interface.
func (o *Object) GoString() string {
	return fmt.Sprintf("*%#v", *o)
}

// Get gets all the objects that match the given key.
//
// It returns the resulting objects as a single Object structure with
// the linked list populated.
func (o *Object) Get(k string, insensitive bool) *Object {
	if o.Type != ValueTypeObject {
		return nil
	}

	for _, o := range o.Elem(true) {
		if o.Key != k {
			if !insensitive || !strings.EqualFold(o.Key, k) {
				continue
			}
		}

		return o
	}

	return nil
}

// Elem returns all the elements that are part of this object.
func (o *Object) Elem(expand bool) []*Object {
	if !expand {
		result := make([]*Object, 0, 1)
		current := o
		for current != nil {
			obj := *current
			obj.Next = nil
			result = append(result, &obj)

			current = current.Next
		}

		return result
	}

	if o.Value == nil {
		return nil
	}

	switch o.Type {
	case ValueTypeList:
		return o.Value.([]*Object)
	case ValueTypeObject:
		result := make([]*Object, 0, 5)
		for _, obj := range o.Elem(false) {
			result = append(result, obj.Value.([]*Object)...)
		}
		return result
	default:
		return []*Object{o}
	}
}

// Len returns the number of objects in this object structure.
func (o *Object) Len() (i int) {
	current := o
	for current != nil {
		i += 1
		current = current.Next
	}

	return
}

// ObjectList is a list of objects.
type ObjectList []*Object

// Flat returns a flattened list structure of the objects.
func (l ObjectList) Flat() []*Object {
	m := make(map[string]*Object)
	result := make([]*Object, 0, len(l))
	for _, obj := range l {
		prev, ok := m[obj.Key]
		if !ok {
			m[obj.Key] = obj
			result = append(result, obj)
			continue
		}

		for prev.Next != nil {
			prev = prev.Next
		}
		prev.Next = obj
	}

	return result
}
