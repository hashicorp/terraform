// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package collections

import (
	"github.com/google/go-cmp/cmp"
)

// CmpOptions is a set of options for use with the "go-cmp" module when
// comparing data structures that contain collection types from this package.
//
// Specifically, these options arrange for [Set] and [Map] values to be
// transformed into map[any]any to allow for element-based comparisons.
//
// [Set] of T values transform into a map whose keys have dynamic type
// UniqueKey[T] and whose values have dynamic type T.
//
// [Map] of K, V values transform into a map whose keys have dynamic type
// UniqueKey[K] and whose values have dynamic type MapElem[K, V].
var CmpOptions cmp.Option

func init() {
	CmpOptions = cmp.Options([]cmp.Option{
		cmp.Transformer("collectionElementsRaw", func(v transformerForCmp) any {
			return v.transformForCmp()
		}),
	})
}

// transformerForCmp is a helper interface implemented by both `Set` and `Map`
// types, to work around the fact that go-cmp does all its work with reflection
// and thus cannot rely on the static type information from the type
// parameters.
type transformerForCmp interface {
	transformForCmp() any
}

func (s Set[T]) transformForCmp() any {
	ret := make(map[any]any, s.Len())
	// It's okay to access the keys here because this package is allowed to
	// depend on its own implementation details.
	for k, v := range s.members {
		ret[k] = v
	}
	return ret
}

func (m Map[K, V]) transformForCmp() any {
	ret := make(map[any]any, m.Len())
	// It's okay to access the keys here because this package is allowed to
	// depend on its own implementation details.
	for k, v := range m.elems {
		ret[k] = v
	}
	return ret
}
