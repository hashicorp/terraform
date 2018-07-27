package hcl2shim

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zclconf/go-cty/cty"
)

// PathFromFlatmapKey takes a key from a flatmap along with the cty.Type
// describing the structure, and returns the cty.Path that would be used to
// reference the nested value in the data structure.
func PathFromFlatmapKey(k string, ty cty.Type) (cty.Path, error) {
	if k == "" {
		return nil, nil
	}
	if !ty.IsObjectType() {
		panic(fmt.Sprintf("PathFromFlatmapKey called on %#v", ty))
	}

	path, err := pathFromFlatmapKeyObject(k, ty.AttributeTypes())
	if err != nil {
		return path, fmt.Errorf("[%s] %s", k, err)
	}
	return path, nil
}

func pathSplit(p string) (string, string) {
	parts := strings.SplitN(p, ".", 2)
	head := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	}
	return head, rest
}

func pathFromFlatmapKeyObject(key string, atys map[string]cty.Type) (cty.Path, error) {
	k, rest := pathSplit(key)

	path := cty.Path{cty.GetAttrStep{Name: k}}

	ty, ok := atys[k]
	if !ok {
		return path, fmt.Errorf("attribute %q not found", key)
	}

	if rest == "" {
		return path, nil
	}

	p, err := pathFromFlatmapKeyValue(rest, ty)
	if err != nil {
		return path, err
	}

	return append(path, p...), nil
}

func pathFromFlatmapKeyValue(key string, ty cty.Type) (cty.Path, error) {
	var path cty.Path
	var err error

	switch {
	case ty.IsPrimitiveType():
		err = fmt.Errorf("invalid step %q with type %#v", key, ty)
	case ty.IsObjectType():
		path, err = pathFromFlatmapKeyObject(key, ty.AttributeTypes())
	case ty.IsTupleType():
		path, err = pathFromFlatmapKeyTuple(key, ty.TupleElementTypes())
	case ty.IsMapType():
		path, err = pathFromFlatmapKeyMap(key, ty)
	case ty.IsListType():
		path, err = pathFromFlatmapKeyList(key, ty)
	case ty.IsSetType():
		path, err = pathFromFlatmapKeySet(key, ty)
	default:
		err = fmt.Errorf("unrecognized type: %s", ty.FriendlyName())
	}

	if err != nil {
		return path, err
	}

	return path, nil
}

func pathFromFlatmapKeyTuple(key string, etys []cty.Type) (cty.Path, error) {
	var path cty.Path
	var err error

	k, rest := pathSplit(key)

	// we don't need to convert the index keys to paths
	if k == "#" {
		return path, nil
	}

	idx, err := strconv.Atoi(k)
	if err != nil {
		return path, err
	}

	path = cty.Path{cty.IndexStep{Key: cty.NumberIntVal(int64(idx))}}

	if idx >= len(etys) {
		return path, fmt.Errorf("index %s out of range in %#v", key, etys)
	}

	if rest == "" {
		return path, nil
	}

	ty := etys[idx]

	p, err := pathFromFlatmapKeyValue(rest, ty.ElementType())
	if err != nil {
		return path, err
	}

	return append(path, p...), nil
}

func pathFromFlatmapKeyMap(key string, ty cty.Type) (cty.Path, error) {
	var path cty.Path
	var err error

	k, rest := pathSplit(key)

	// we don't need to convert the index keys to paths
	if k == "%" {
		return path, nil
	}

	path = cty.Path{cty.IndexStep{Key: cty.StringVal(k)}}

	if rest == "" {
		return path, nil
	}

	p, err := pathFromFlatmapKeyValue(rest, ty.ElementType())
	if err != nil {
		return path, err
	}

	return append(path, p...), nil
}

func pathFromFlatmapKeyList(key string, ty cty.Type) (cty.Path, error) {
	var path cty.Path
	var err error

	k, rest := pathSplit(key)

	// we don't need to convert the index keys to paths
	if key == "#" {
		return path, nil
	}

	idx, err := strconv.Atoi(k)
	if err != nil {
		return path, err
	}

	path = cty.Path{cty.IndexStep{Key: cty.NumberIntVal(int64(idx))}}

	if rest == "" {
		return path, nil
	}

	p, err := pathFromFlatmapKeyValue(rest, ty.ElementType())
	if err != nil {
		return path, err
	}

	return append(path, p...), nil
}

func pathFromFlatmapKeySet(key string, ty cty.Type) (cty.Path, error) {
	var path cty.Path
	var err error

	k, rest := pathSplit(key)

	// we don't need to convert the index keys to paths
	if k == "#" {
		return path, nil
	}

	idx, err := strconv.Atoi(k)
	if err != nil {
		return path, err
	}

	path = cty.Path{cty.IndexStep{Key: cty.NumberIntVal(int64(idx))}}

	if rest == "" {
		return path, nil
	}

	p, err := pathFromFlatmapKeyValue(rest, ty.ElementType())
	if err != nil {
		return path, err
	}

	return append(path, p...), nil
}
