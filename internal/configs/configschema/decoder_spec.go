// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configschema

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
)

var mapLabelNames = []string{"key"}

// specCache is a global cache of all the generated hcldec.Spec values for
// Blocks. This cache is used by the Block.DecoderSpec method to memoize calls
// and prevent unnecessary regeneration of the spec, especially when they are
// large and deeply nested.
// Caching these externally rather than within the struct is required because
// Blocks are used by value and copied when working with NestedBlocks, and the
// copying of the value prevents any safe synchronisation of the struct itself.
//
// While we are using the *Block pointer as the cache key, and the Block
// contents are mutable, once a Block is created it is treated as immutable for
// the duration of its life. Because a Block is a representation of a logical
// schema, which cannot change while it's being used, any modifications to the
// schema during execution would be an error.
type specCache struct {
	sync.Mutex
	specs map[uintptr]hcldec.Spec
}

var decoderSpecCache = specCache{
	specs: map[uintptr]hcldec.Spec{},
}

// get returns the Spec associated with eth given Block, or nil if non is
// found.
func (s *specCache) get(b *Block) hcldec.Spec {
	s.Lock()
	defer s.Unlock()
	k := uintptr(unsafe.Pointer(b))
	return s.specs[k]
}

// set stores the given Spec as being the result of b.DecoderSpec().
func (s *specCache) set(b *Block, spec hcldec.Spec) {
	s.Lock()
	defer s.Unlock()

	// the uintptr value gets us a unique identifier for each block, without
	// tying this to the block value itself.
	k := uintptr(unsafe.Pointer(b))
	if _, ok := s.specs[k]; ok {
		return
	}

	s.specs[k] = spec

	// This must use a finalizer tied to the Block, otherwise we'll continue to
	// build up Spec values as the Blocks are recycled.
	runtime.SetFinalizer(b, s.delete)
}

// delete removes the spec associated with the given Block.
func (s *specCache) delete(b *Block) {
	s.Lock()
	defer s.Unlock()

	k := uintptr(unsafe.Pointer(b))
	delete(s.specs, k)
}

// DecoderSpec returns a hcldec.Spec that can be used to decode a HCL Body
// using the facilities in the hcldec package.
//
// The returned specification is guaranteed to return a value of the same type
// returned by method ImpliedType, but it may contain null values if any of the
// block attributes are defined as optional and/or computed respectively.
func (b *Block) DecoderSpec() hcldec.Spec {
	ret := hcldec.ObjectSpec{}
	if b == nil {
		return ret
	}

	if spec := decoderSpecCache.get(b); spec != nil {
		return spec
	}

	for name, attrS := range b.Attributes {
		ret[name] = attrS.decoderSpec(name)
	}

	for name, blockS := range b.BlockTypes {
		if _, exists := ret[name]; exists {
			// This indicates an invalid schema, since it's not valid to define
			// both an attribute and a block type of the same name. We assume
			// that the provider has already used something like
			// InternalValidate to validate their schema.
			continue
		}

		childSpec := blockS.Block.DecoderSpec()

		switch blockS.Nesting {
		case NestingSingle, NestingGroup:
			ret[name] = &hcldec.BlockSpec{
				TypeName: name,
				Nested:   childSpec,
				Required: blockS.MinItems == 1,
			}
			if blockS.Nesting == NestingGroup {
				ret[name] = &hcldec.DefaultSpec{
					Primary: ret[name],
					Default: &hcldec.LiteralSpec{
						Value: blockS.EmptyValue(),
					},
				}
			}
		case NestingList:
			// We prefer to use a list where possible, since it makes our
			// implied type more complete, but if there are any
			// dynamically-typed attributes inside we must use a tuple
			// instead, at the expense of our type then not being predictable.
			if blockS.Block.specType().HasDynamicTypes() {
				ret[name] = &hcldec.BlockTupleSpec{
					TypeName: name,
					Nested:   childSpec,
					MinItems: blockS.MinItems,
					MaxItems: blockS.MaxItems,
				}
			} else {
				ret[name] = &hcldec.BlockListSpec{
					TypeName: name,
					Nested:   childSpec,
					MinItems: blockS.MinItems,
					MaxItems: blockS.MaxItems,
				}
			}
		case NestingSet:
			// We forbid dynamically-typed attributes inside NestingSet in
			// InternalValidate, so we don't do anything special to handle that
			// here. (There is no set analog to tuple and object types, because
			// cty's set implementation depends on knowing the static type in
			// order to properly compute its internal hashes.)  We assume that
			// the provider has already used something like InternalValidate to
			// validate their schema.
			ret[name] = &hcldec.BlockSetSpec{
				TypeName: name,
				Nested:   childSpec,
				MinItems: blockS.MinItems,
				MaxItems: blockS.MaxItems,
			}
		case NestingMap:
			// We prefer to use a list where possible, since it makes our
			// implied type more complete, but if there are any
			// dynamically-typed attributes inside we must use a tuple
			// instead, at the expense of our type then not being predictable.
			if blockS.Block.specType().HasDynamicTypes() {
				ret[name] = &hcldec.BlockObjectSpec{
					TypeName:   name,
					Nested:     childSpec,
					LabelNames: mapLabelNames,
				}
			} else {
				ret[name] = &hcldec.BlockMapSpec{
					TypeName:   name,
					Nested:     childSpec,
					LabelNames: mapLabelNames,
				}
			}
		default:
			// Invalid nesting type is just ignored. It's checked by
			// InternalValidate.  We assume that the provider has already used
			// something like InternalValidate to validate their schema.
			continue
		}
	}

	decoderSpecCache.set(b, ret)
	return ret
}

func (a *Attribute) decoderSpec(name string) hcldec.Spec {
	ret := &hcldec.AttrSpec{Name: name}
	if a == nil {
		return ret
	}

	if a.NestedType != nil {
		if a.Type != cty.NilType {
			panic("Invalid attribute schema: NestedType and Type cannot both be set. This is a bug in the provider.")
		}

		ty := a.NestedType.specType()
		ret.Type = ty
		ret.Required = a.Required
		return ret
	}

	ret.Type = a.Type
	ret.Required = a.Required
	return ret
}

// listOptionalAttrsFromObject is a helper function which does *not* recurse
// into NestedType Attributes, because the optional types for each of those will
// belong to their own cty.Object definitions. It is used in other functions
// which themselves handle that recursion.
func listOptionalAttrsFromObject(o *Object) []string {
	ret := make([]string, 0)

	// This is unlikely to happen outside of tests.
	if o == nil {
		return ret
	}

	for name, attr := range o.Attributes {
		if attr.Optional || attr.Computed {
			ret = append(ret, name)
		}
	}
	return ret
}
