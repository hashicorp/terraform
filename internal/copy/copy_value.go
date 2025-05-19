// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package copy

import (
	"reflect"
)

// DeepCopyValue produces a deep copy of the given value, where the result
// ideally shares no mutable memory with the given value.
//
// There are some limitations on what's possible, however:
//   - This package can't write into an unexported field of a struct, so those
//     will be ignored entirely and thus left as their zero values in the
//     result.
//   - If the given structure contains function pointers then their closures
//     might refer to shared memory that this function cannot copy. If they
//     refer to memory that's also included in the data structure outside of
//     the function pointer then the two will be disconnected in the result.
//   - It isn't really meaningful to "copy" a channel since it's a
//     synchronization primitive rather than a data structure, so the result
//     will share the same channels as the input.
//   - Copying other library-based synchronization primitives like [sync.Mutex]
//     doesn't really make sense either, although this function doesn't
//     understand what they are and so the result is undefined. If your
//     synchronization primitive has a "ready to use" zero value then it _might_
//     be acceptable to store it in an unexported field and thus have it be
//     zeroed in the result, but at that point you're probably better off
//     writing a specialized deepcopy function so that you can actually use
//     the synchronization primitive to prevent data races during copying.
//   - The uintptr and [unsafe.Pointer] types might well refer to some shared
//     memory, but don't give any information about how to copy that memory
//     and so those are just preserved verbatim, making the result point to
//     the same memory as the input.
//   - Broadly, this function needs special handling for each different kind
//     of value in Go, and so if a later version of Go has introduced a new kind
//     of value then this function might not support it yet. That might cause
//     this function to panic, or to ignore part of the structure, or otherwise
//     misbehave.
//
// This is intended as a relatively simple utility for straightforward cases,
// primarily for use in contrived situations like unit tests.
//
// It intentionally does not offer any customization; if you need to do
// something special then it's better to just write your own simple direct code
// than to pull in all of this reflection trickery. Even if you don't need to do
// something special it's probably still better to just write some
// straightforward code that directly describes the behavior you're intending,
// so that the Go compiler can help you and so you don't force future
// maintainers to understand all of this metaprogramming if something goes
// wrong. Seriously... don't use this function.
func DeepCopyValue[T any](v T) T {
	// We use type parameters in the signature to make usage more convenient
	// for the caller (no type assertions required) but we actually do all
	// our internal work in the realm of package reflect.
	input := reflect.ValueOf(&v).Elem() // if T is an interface type then input is the interface value, not the value inside it
	ty := reflect.TypeFor[T]()          // likewise, if T is interface type then this is the static interface type, not the dynamic type
	result := deepCopyValue(input, ty)
	return result.Interface().(T)
}

func deepCopyValue(v reflect.Value, ty reflect.Type) reflect.Value {
	switch ty.Kind() {
	// A large subset of the kinds don't point to mutable sharable memory,
	// or don't refer to something we can possibly copy, and so we can just
	// return them directly without any extra work.
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String, reflect.UnsafePointer, reflect.Func, reflect.Chan:
		return v

	case reflect.Array:
		return deepCopyArray(v, ty)
	case reflect.Interface:
		return deepCopyInterface(v, ty)
	case reflect.Map:
		return deepCopyMap(v, ty)
	case reflect.Pointer:
		return deepCopyPointer(v, ty)
	case reflect.Slice:
		return deepCopySlice(v, ty)
	case reflect.Struct:
		return deepCopyStruct(v, ty)

	default:
		panic("unsupported type kind " + ty.Kind().String())
	}
}

func deepCopyArray(v reflect.Value, ty reflect.Type) reflect.Value {
	// Copying an array really means allocating a new array and then
	// copying each of the elements from the source.
	ret := reflect.New(ty).Elem()
	for i := range ret.Len() {
		newElemV := deepCopyValue(v.Index(i), ty.Elem())
		ret.Index(i).Set(newElemV)
	}
	return ret
}

func deepCopyInterface(v reflect.Value, ty reflect.Type) reflect.Value {
	if v.IsNil() {
		return v
	}
	// An interface value is not directly mutable itself, but the value
	// inside it might be and so we'll copy that and then wrap the result
	// in a new interface value of the same type.
	ret := reflect.New(ty).Elem()
	dynV := deepCopyValue(v.Elem(), v.Elem().Type())
	ret.Set(dynV)
	return ret
}

func deepCopyMap(v reflect.Value, ty reflect.Type) reflect.Value {
	if v.IsNil() {
		return v
	}
	ret := reflect.MakeMap(ty)
	for iter := v.MapRange(); iter.Next(); {
		// We don't copy the key because Go does not allow any mutably-aliasable
		// types as map keys. (That would make it very easy to corrupt the
		// internals of the map, after all!)
		k := iter.Key()
		v := deepCopyValue(iter.Value(), ty.Elem())
		ret.SetMapIndex(k, v)
	}
	return ret
}

func deepCopyPointer(v reflect.Value, ty reflect.Type) reflect.Value {
	if v.IsNil() {
		return v
	}
	// We copy a pointer by copying what it refers to and then returning
	// a pointer to that copy.
	newTarget := deepCopyValue(v.Elem(), ty.Elem())
	return newTarget.Addr()
}

func deepCopySlice(v reflect.Value, ty reflect.Type) reflect.Value {
	if v.IsNil() {
		return v
	}
	// Copying a slice really means copying the part of its backing array
	// that in could potentially observe. In particular, it's possible to
	// expand the view of the backing array up to the slice's capacity,
	// so we need to copy the entire capacity even if the length is
	// currently shorter to ensure that the result is truly equivalent.
	length := v.Len()
	capacity := v.Cap()

	// This exposes any elements that are between length and capacity.
	fullView := v.Slice3(0, capacity, capacity)
	// Making a slice also allocates a new backing array for it.
	ret := reflect.MakeSlice(ty, capacity, capacity)
	for i := range capacity {
		ret.Index(i).Set(fullView.Index(i))
	}

	// We must restore the original length before we return.
	return ret.Slice(0, length)
}

func deepCopyStruct(v reflect.Value, ty reflect.Type) reflect.Value {
	// To copy a struct we must copy each exported field one by one.
	// We can't assign to unexported fields and so we just leave those
	// unset in the new value.
	ret := reflect.New(ty).Elem()
	for i := range ty.NumField() {
		fieldRet := ret.Field(i)
		if !fieldRet.CanSet() {
			// Presumably it's an unexported field, so we can't do anything
			// with it and must leave it zeroed.
			continue
		}
		newVal := deepCopyValue(v.Field(i), ty.Field(i).Type)
		fieldRet.Set(newVal)
	}
	return ret
}
