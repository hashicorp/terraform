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
			// This indicates an invalid schema, since it's not valid to
			// define both an attribute and a block type of the same name.
			// However, we don't raise this here since it's checked by
			// InternalValidate.
			continue
		}

		childSpec := blockS.Block.DecoderSpec()

		// We can only validate 0 or 1 for MinItems, because a dynamic block
		// may satisfy any number of min items while only having a single
		// block in the config. We cannot validate MaxItems because a
		// configuration may have any number of dynamic blocks
		minItems := 0
		if blockS.MinItems > 1 {
			minItems = 1
		}

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
			if blockS.Block.ImpliedType().HasDynamicTypes() {
				ret[name] = &hcldec.BlockTupleSpec{
					TypeName: name,
					Nested:   childSpec,
					MinItems: minItems,
				}
			} else {
				ret[name] = &hcldec.BlockListSpec{
					TypeName: name,
					Nested:   childSpec,
					MinItems: minItems,
				}
			}
		case NestingSet:
			// We forbid dynamically-typed attributes inside NestingSet in
			// InternalValidate, so we don't do anything special to handle
			// that here. (There is no set analog to tuple and object types,
			// because cty's set implementation depends on knowing the static
			// type in order to properly compute its internal hashes.)
			ret[name] = &hcldec.BlockSetSpec{
				TypeName: name,
				Nested:   childSpec,
				MinItems: minItems,
			}
		case NestingMap:
			// We prefer to use a list where possible, since it makes our
			// implied type more complete, but if there are any
			// dynamically-typed attributes inside we must use a tuple
			// instead, at the expense of our type then not being predictable.
			if blockS.Block.ImpliedType().HasDynamicTypes() {
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
			// InternalValidate.
			continue
		}
	}

	decoderSpecCache.set(b, ret)
	return ret
}

func (a *Attribute) decoderSpec(name string) hcldec.Spec {
	ret := &hcldec.AttrSpec{Name: name}

	if a.NestedType != nil {
		var optAttrs []string
		optAttrs = listOptionalAttrsFromBlock(a.NestedType.Block, optAttrs)
		ty := a.NestedType.Block.ImpliedType()
		if !ty.IsObjectType() {
			panic("NestedType must be an Object")
		}

		switch a.NestedType.Nesting {
		case NestingList:
			ret.Type = cty.List(cty.ObjectWithOptionalAttrs(ty.AttributeTypes(), optAttrs))
		case NestingSet:
			ret.Type = cty.Set(cty.ObjectWithOptionalAttrs(ty.AttributeTypes(), optAttrs))
		case NestingMap:
			ret.Type = cty.Map(cty.ObjectWithOptionalAttrs(ty.AttributeTypes(), optAttrs))
		default: // NestingSingle or no nesting
			ret.Type = cty.ObjectWithOptionalAttrs(ty.AttributeTypes(), optAttrs)
		}
		ret.Required = a.NestedType.MinItems > 0

		return ret
	}

	ret.Type = a.Type
	ret.Required = a.Required
	return ret
}

func listOptionalAttrsFromBlock(b Block, optAttrs []string) []string {
	for name, attr := range b.Attributes {
		if attr.Optional == true {
			optAttrs = append(optAttrs, name)
		}
	}

	for _, block := range b.BlockTypes {
		listOptionalAttrsFromBlock(block.Block, optAttrs)
	}

	return optAttrs
}
