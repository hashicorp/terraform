package diffs

import (
	"fmt"
	"hash/crc64"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/set"
)

// PathSet represents a set of cty.Path objects.
type PathSet struct {
	set set.Set
}

// NewPathSet creates and returns an empty PathSet.
func NewPathSet() PathSet {
	return PathSet{
		set: set.NewSet(pathSetRules{}),
	}
}

func (s PathSet) Add(path cty.Path) {
	s.set.Add(path)
}

// AddAllSteps is like Add but it also adds all of the steps leading to
// the given path.
func (s PathSet) AddAllSteps(path cty.Path) {
	for i := 1; i <= len(path); i++ {
		s.Add(path[:i])
	}
}

func (s PathSet) Has(path cty.Path) bool {
	return s.set.Has(path)
}

func (s PathSet) Remove(path cty.Path) {
	s.set.Remove(path)
}

func (s PathSet) Empty() bool {
	return s.set.Length() == 0
}

func (s PathSet) Union(other PathSet) PathSet {
	return PathSet{
		set: s.set.Union(other.set),
	}
}

func (s PathSet) Intersection(other PathSet) PathSet {
	return PathSet{
		set: s.set.Intersection(other.set),
	}
}

func (s PathSet) Subtract(other PathSet) PathSet {
	return PathSet{
		set: s.set.Subtract(other.set),
	}
}

func (s PathSet) SymmetricDifference(other PathSet) PathSet {
	return PathSet{
		set: s.set.SymmetricDifference(other.set),
	}
}

var crc64Table = crc64.MakeTable(crc64.ISO)

var indexStepPlaceholder = []byte("#")

// pathSetRules is an implementation of set.Rules from cty's set package,
// used internally within PathSet.
type pathSetRules struct {
}

func (r pathSetRules) Hash(v interface{}) int {
	path := v.(cty.Path)
	hash := crc64.New(crc64Table)

	for _, rawStep := range path {
		switch step := rawStep.(type) {
		case cty.GetAttrStep:
			// (this creates some garbage converting the string name to a
			// []byte, but we don't care too much since Terraform is a
			// short-lived program.)
			hash.Write([]byte(step.Name))
		default:
			// For any other step type we just append a predefined value,
			// which means that e.g. all indexes into a given collection will
			// hash to the same value but we assume that collections are
			// small and thus this won't hurt too much.
			hash.Write(indexStepPlaceholder)
		}
	}

	// We discard half of the hash on 32-bit platforms; collisions just make
	// our lookups take marginally longer, so not a big deal.
	return int(hash.Sum64())
}

func (r pathSetRules) Equivalent(a, b interface{}) bool {
	aPath := a.(cty.Path)
	bPath := b.(cty.Path)

	if len(aPath) != len(bPath) {
		return false
	}

	for i := range aPath {
		switch aStep := aPath[i].(type) {
		case cty.GetAttrStep:
			bStep, ok := bPath[i].(cty.GetAttrStep)
			if !ok {
				return false
			}

			if aStep.Name != bStep.Name {
				return false
			}
		case cty.IndexStep:
			bStep, ok := bPath[i].(cty.IndexStep)
			if !ok {
				return false
			}

			eq := aStep.Key.Equals(bStep.Key)
			if !eq.IsKnown() || eq.False() {
				return false
			}
		default:
			// Should never happen, since cty documents this as a closed type.
			panic(fmt.Errorf("unsupported step type %T", aStep))
		}
	}

	return true
}
