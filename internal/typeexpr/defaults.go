package typeexpr

import (
	"github.com/zclconf/go-cty/cty"
)

// Defaults represents a type tree which may contain default values for
// optional object attributes at any level. This is used to apply nested
// defaults to an input value before converting it to the concrete type.
type Defaults struct {
	// Type of the node for which these defaults apply. This is necessary in
	// order to determine how to inspect the Defaults and Children collections.
	Type cty.Type

	// DefaultValues contains the default values for each object attribute,
	// indexed by attribute name.
	DefaultValues map[string]cty.Value

	// Children is a map of Defaults for elements contained in this type. This
	// only applies to structural and collection types.
	//
	// The map is indexed by string instead of cty.Value because cty.Number
	// instances are non-comparable, due to embedding a *big.Float.
	//
	// Collections have a single element type, which is stored at key "".
	Children map[string]*Defaults
}

// Apply walks the given value, applying specified defaults wherever optional
// attributes are missing. The input and output values may have different
// types, and the result may still require type conversion to the final desired
// type.
//
// This function is permissive and does not report errors, assuming that the
// caller will have better context to report useful type conversion failure
// diagnostics.
func (d *Defaults) Apply(val cty.Value) cty.Value {
	val, err := cty.TransformWithTransformer(val, &defaultsTransformer{defaults: d})

	// The transformer should never return an error.
	if err != nil {
		panic(err)
	}

	return val
}

// defaultsTransformer implements cty.Transformer, as a pre-order traversal,
// applying defaults as it goes. The pre-order traversal allows us to specify
// defaults more loosely for structural types, as the defaults for the types
// will be applied to the default value later in the walk.
type defaultsTransformer struct {
	defaults *Defaults
}

var _ cty.Transformer = (*defaultsTransformer)(nil)

func (t *defaultsTransformer) Enter(p cty.Path, v cty.Value) (cty.Value, error) {
	// Cannot apply defaults to an unknown value
	if !v.IsKnown() {
		return v, nil
	}

	// Look up the defaults for this path.
	defaults := t.defaults.traverse(p)

	// If we have no defaults, nothing to do.
	if len(defaults) == 0 {
		return v, nil
	}

	// Ensure we are working with an object or map.
	vt := v.Type()
	if !vt.IsObjectType() && !vt.IsMapType() {
		// Cannot apply defaults because the value type is incompatible.
		// We'll ignore this and let the later conversion stage display a
		// more useful diagnostic.
		return v, nil
	}

	// Unmark the value and reapply the marks later.
	v, valMarks := v.Unmark()

	// Convert the given value into an attribute map (if it's non-null and
	// non-empty).
	attrs := make(map[string]cty.Value)
	if !v.IsNull() && v.LengthInt() > 0 {
		attrs = v.AsValueMap()
	}

	// Apply defaults where attributes are missing, constructing a new
	// value with the same marks.
	for attr, defaultValue := range defaults {
		if _, ok := attrs[attr]; !ok {
			attrs[attr] = defaultValue
		}
	}

	// We construct an object even if the input value was a map, as the
	// type of an attribute's default value may be incompatible with the
	// map element type.
	return cty.ObjectVal(attrs).WithMarks(valMarks), nil
}

func (t *defaultsTransformer) Exit(p cty.Path, v cty.Value) (cty.Value, error) {
	return v, nil
}

// traverse walks the abstract defaults structure for a given path, returning
// a set of default values (if any are present) or nil (if not). This operation
// differs from applying a path to a value because we need to customize the
// traversal steps for collection types, where a single set of defaults can be
// applied to an arbitrary number of elements.
func (d *Defaults) traverse(path cty.Path) map[string]cty.Value {
	if len(path) == 0 {
		return d.DefaultValues
	}

	switch s := path[0].(type) {
	case cty.GetAttrStep:
		if d.Type.IsObjectType() {
			// Attribute path steps are normally applied to objects, where each
			// attribute may have different defaults.
			return d.traverseChild(s.Name, path)
		} else if d.Type.IsMapType() {
			// Literal values for maps can result in attribute path steps, in which
			// case we need to disregard the attribute name, as maps can have only
			// one child.
			return d.traverseChild("", path)
		}

		return nil
	case cty.IndexStep:
		if d.Type.IsTupleType() {
			// Tuples can have different types for each element, so we look
			// up the defaults based on the index key.
			return d.traverseChild(s.Key.AsBigFloat().String(), path)
		} else if d.Type.IsCollectionType() {
			// Defaults for collection element types are stored with a blank
			// key, so we disregard the index key.
			return d.traverseChild("", path)
		}
		return nil
	default:
		// At time of writing there are no other path step types.
		return nil
	}
}

// traverseChild continues the traversal for a given child key, and mutually
// recurses with traverse.
func (d *Defaults) traverseChild(name string, path cty.Path) map[string]cty.Value {
	if child, ok := d.Children[name]; ok {
		return child.traverse(path[1:])
	}
	return nil
}
