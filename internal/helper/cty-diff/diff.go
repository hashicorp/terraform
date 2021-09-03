package cty_diff

import (
	"sort"

	"github.com/zclconf/go-cty/cty"
)

const (
	EditNoOp   byte = '='
	EditInsert byte = '+'
	EditDelete byte = '-'
	EditModify byte = '~'
)

type EditStep struct {
	Operation          byte // EditNoOp etc.
	OldKey, NewKey     cty.PathStep
	OldValue, NewValue cty.Value
}

// ListDiff calculates the edit distance and path between two lists.
// The distance is a floating-point number between 0 (identical) and 1 (completely dissimilar).
func ListDiff(a, b cty.Value, withPath bool) (float32, []EditStep) {
	// Weighted Wagner-Fischer, O(n*m) space and time
	// (not including weight calculation).
	// Calculating weight (distance) for a pair should only be done once.

	m := a.LengthInt()
	n := b.LengthInt()

	// TODO: use a single array
	d := make([][]float32, m+1)
	for i := 0; i < m+1; i++ {
		d[i] = make([]float32, n+1)
	}

	var bestPath [][]byte
	if withPath {
		bestPath = make([][]byte, m+1)
		for i := 0; i < m+1; i++ {
			bestPath[i] = make([]byte, n+1)
		}
	}

	for i := 1; i <= m; i++ {
		d[i][0] = float32(i)
		if withPath {
			bestPath[i][0] = EditDelete
		}
	}

	for j := 1; j <= n; j++ {
		d[0][j] = float32(j)
		if withPath {
			bestPath[0][j] = EditInsert
		}
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			// Traverse lists in reverse order,
			// so that the path is in forward order.
			aVal := a.Index(cty.NumberIntVal(int64(m - i)))
			bVal := b.Index(cty.NumberIntVal(int64(n - j)))
			costModify := ValueDiff(aVal, bVal)

			totalCostDelete := d[i-1][j] + 1
			totalCostInsert := d[i][j-1] + 1
			totalCostModify := d[i-1][j-1] + costModify

			bestCost := totalCostDelete
			bestOp := EditDelete
			if totalCostInsert < bestCost {
				bestCost = totalCostInsert
				bestOp = EditInsert
			}
			if totalCostModify < bestCost {
				bestCost = totalCostModify
				if costModify == 0 {
					bestOp = EditNoOp
				} else {
					bestOp = EditModify
				}
			}

			d[i][j] = bestCost
			if withPath {
				bestPath[i][j] = bestOp
			}
		}
	}

	pathLength := m
	if pathLength < n {
		pathLength = n
	}
	distance := d[m][n] / float32(pathLength)

	var editPath []EditStep
	if withPath {
		editPath = make([]EditStep, 0, pathLength)
		i := m
		j := n
		for i > 0 || j > 0 {
			var aKey, bKey cty.PathStep
			var aVal, bVal cty.Value
			op := bestPath[i][j]
			if op != EditInsert {
				aIndex := cty.NumberIntVal(int64(m - i))
				aKey = cty.IndexStep{Key: aIndex}
				aVal = a.Index(aIndex)
			}
			if op != EditDelete {
				bIndex := cty.NumberIntVal(int64(n - j))
				bKey = cty.IndexStep{Key: bIndex}
				bVal = b.Index(bIndex)
			}
			switch {
			case op == EditDelete:
				i--
			case op == EditInsert:
				j--
			default: // EditModify or EditNoOp
				i--
				j--
			}
			step := EditStep{op, aKey, bKey, aVal, bVal}
			editPath = append(editPath, step)
		}
	}

	return distance, editPath
}

// SetDiff calculates the edit distance and path between two unordered sets.
// The distance is a floating-point number between 0 (identical) and 1 (completely dissimilar).
func SetDiff(a, b cty.Value, withPath bool) (float32, []EditStep) {
	// Simple O(n*m) greedy algorithm which sorts pairs by similarity.
	// Calculating weight (distance) for a pair should only be done once.

	m := a.LengthInt()
	n := b.LengthInt()

	aValues := ctyCollectionValues(a)
	bValues := ctyCollectionValues(b)

	// Construct an (m+1)*(n+1) matrix which holds the distance for editing element (i-1) to (j-1).
	// For i==0 the operation is insertion, and for j==0 it is deletion.
	type pair struct {
		distance float32
		x        int
		y        int
	}
	pairs := make([]pair, (m+1)*(n+1)-1)
	pos := 0
	for y := 0; y <= n; y++ {
		for x := 0; x <= m; x++ {
			if x == 0 && y == 0 {
				continue
			}
			var aVal, bVal cty.Value
			if x > 0 {
				aVal = aValues[x-1]
			}
			if y > 0 {
				bVal = bValues[y-1]
			}
			pairs[pos] = pair{ValueDiff(aVal, bVal), x, y}
			pos++
		}
	}

	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].distance < pairs[j].distance
	})

	// We use the first (lowest-distance) edit operation
	// per either source or target element.
	aUsed := make([]bool, m)
	bUsed := make([]bool, n)

	totalDiff := float32(0)
	numSteps := 0

	var editPath []EditStep
	if withPath {
		pathLength := m
		if pathLength < n {
			pathLength = n
		}
		editPath = make([]EditStep, 0, pathLength)
	}

	for _, pair := range pairs {
		if pair.x > 0 && aUsed[pair.x-1] {
			continue
		}
		if pair.y > 0 && bUsed[pair.y-1] {
			continue
		}
		var op byte
		var aKey, bKey cty.PathStep
		var aVal, bVal cty.Value
		if pair.x > 0 {
			aUsed[pair.x-1] = true
			aVal = aValues[pair.x-1]
			aKey = cty.IndexStep{Key: aVal}
		}
		if pair.y > 0 {
			bUsed[pair.y-1] = true
			bVal = bValues[pair.y-1]
			bKey = cty.IndexStep{Key: bVal}
		}
		switch {
		case pair.x == 0:
			op = EditInsert
		case pair.y == 0:
			op = EditDelete
		case pair.distance == 0:
			op = EditNoOp
		default:
			op = EditModify
		}

		totalDiff += pair.distance
		numSteps++

		if withPath {
			editPath = append(editPath, EditStep{
				op,
				aKey,
				bKey,
				aVal,
				bVal,
			})
		}

	}

	distance := totalDiff / float32(numSteps)

	return distance, editPath
}

// MapDiff calculates the edit distance and path between two maps.
// The distance is a floating-point number between 0 (identical) and 1 (completely dissimilar).
func MapDiff(a, b cty.Value, withPath bool) (float32, []EditStep) {
	allKeys := make([]cty.Value, 0, a.LengthInt()+b.LengthInt())
	for _, m := range []cty.Value{a, b} {
		for it := m.ElementIterator(); it.Next(); {
			key, _ := it.Element()
			allKeys = append(allKeys, key)
		}
	}
	allKeySet := cty.SetVal(allKeys)

	totalDiff := float32(0)
	var editPath []EditStep
	if withPath {
		editPath = make([]EditStep, 0, allKeySet.LengthInt())
	}

	for it := allKeySet.ElementIterator(); it.Next(); {
		key, _ := it.Element()
		aPresent := a.HasIndex(key).True()
		bPresent := b.HasIndex(key).True()
		var aKey, bKey cty.PathStep
		var aVal, bVal cty.Value
		if aPresent {
			aKey = cty.IndexStep{Key: key}
			aVal = a.Index(key)
		}
		if bPresent {
			bKey = cty.IndexStep{Key: key}
			bVal = b.Index(key)
		}
		var distance float32
		var op byte
		switch {
		case !aPresent:
			distance = 1
			op = EditInsert
		case !bPresent:
			distance = 1
			op = EditDelete
		default:
			distance = ValueDiff(
				aVal,
				bVal,
			)
			if distance == 0 {
				op = EditNoOp
			} else {
				op = EditModify
			}
		}
		if withPath {
			editPath = append(editPath, EditStep{
				op, aKey, bKey, aVal, bVal,
			})
		}
		totalDiff += distance
	}

	distance := totalDiff / float32(allKeySet.LengthInt())
	return distance, editPath
}

// ObjectDiff calculates the distance (inverse of similarity) between two objects,
// as a floating-point number between 0 (identical) and 1 (completely dissimilar).
func ObjectDiff(a, b cty.Value) float32 {
	allKeys := make([]cty.Value, 0, a.LengthInt()+b.LengthInt())
	for _, m := range []cty.Value{a, b} {
		for it := m.ElementIterator(); it.Next(); {
			key, _ := it.Element()
			allKeys = append(allKeys, key)
		}
	}
	allKeySet := cty.SetVal(allKeys)

	totalDiff := float32(0)

	for it := allKeySet.ElementIterator(); it.Next(); {
		val, _ := it.Element()
		switch {
		case !a.Type().HasAttribute(val.AsString()):
			totalDiff += 1
		case !b.Type().HasAttribute(val.AsString()):
			totalDiff += 1
		default:
			totalDiff += ValueDiff(
				a.GetAttr(val.AsString()),
				b.GetAttr(val.AsString()),
			)
		}
	}

	return totalDiff / float32(allKeySet.LengthInt())
}

// ValueDiff calculates the distance (inverse of similarity) between two values,
// as a floating-point number between 0 (identical) and 1 (completely dissimilar).
func ValueDiff(a, b cty.Value) float32 {
	ty := a.Type()

	switch {
	case !ty.Equals(b.Type()):
		return 1
	case !a.IsKnown() || !b.IsKnown():
		return 1
	case a.Equals(b) == cty.True:
		return 0
	case ty.IsPrimitiveType():
		return 1
	case a.IsNull() || b.IsNull():
		return 1
	case ty.IsListType() || ty.IsTupleType():
		distance, _ := ListDiff(a, b, false)
		return distance
	case ty.IsSetType():
		distance, _ := SetDiff(a, b, false)
		return distance
	case ty.IsMapType():
		distance, _ := MapDiff(a, b, false)
		return distance
	case ty.IsObjectType():
		return ObjectDiff(a, b)
	default:
		return 1
	}
}

func ctyCollectionValues(val cty.Value) []cty.Value {
	if !val.IsKnown() || val.IsNull() {
		return nil
	}

	ret := make([]cty.Value, 0, val.LengthInt())
	for it := val.ElementIterator(); it.Next(); {
		_, value := it.Element()
		ret = append(ret, value)
	}
	return ret
}
