package format

import (
	"sort"

	"github.com/zclconf/go-cty/cty"
)

// Calculate the distance (inverse of similarity) between two lists,
// as a floating-point number between 0 (identical) and 1 (completely dissimilar).
func listDistance(a, b cty.Value) float32 {
	// Weighted Wagner-Fischer, O(n*m) space and time
	// (not including weight calculation).
	// Calculating weight (distance) for a pair should only be done once.

	m := a.LengthInt()
	n := b.LengthInt()

	d := make([][]float32, m+1)
	for i := 0; i < m+1; i++ {
		d[i] = make([]float32, n+1)
	}

	for i := 1; i <= m; i++ {
		d[i][0] = float32(i)
	}

	for j := 1; j <= n; j++ {
		d[0][j] = float32(j)
	}

	for j := 1; j <= n; j++ {
		for i := 1; i <= m; i++ {
			cost_modify := valueDistance(
				a.Index(cty.NumberIntVal(int64(i-1))),
				b.Index(cty.NumberIntVal(int64(j-1))),
			)

			total_cost_delete := d[i-1][j] + 1
			total_cost_insert := d[i][j-1] + 1
			total_cost_modify := d[i-1][j-1] + cost_modify

			best_cost := total_cost_delete
			if total_cost_insert < best_cost {
				best_cost = total_cost_insert
			}
			if total_cost_modify < best_cost {
				best_cost = total_cost_modify
			}
			d[i][j] = best_cost
		}
	}

	return d[m][n]
}

// Calculate the distance (inverse of similarity) between two unordered sets,
// as a floating-point number between 0 (identical) and 1 (completely dissimilar).
func setDistance(a, b cty.Value) float32 {
	// Simple O(n*m) greedy algorithm which sorts pairs by similarity.
	// Calculating weight (distance) for a pair should only be done once.

	m := a.LengthInt()
	n := b.LengthInt()

	type pair struct {
		distance float32
		x        int
		y        int
	}
	pairs := make([]pair, (m+1)*(n+1)-1)
	pos := 0
	for y := 0; y <= n; y++ {
		for x := 0; x <= m; x++ {
			if x == m && y == n {
				continue
			}
			aVal := cty.DynamicVal
			if x < m {
				aVal = a.Index(cty.NumberIntVal(int64(x)))
			}
			bVal := cty.DynamicVal
			if y < n {
				bVal = b.Index(cty.NumberIntVal(int64(y)))
			}
			pairs[pos] = pair{valueDistance(aVal, bVal), x, y}
			pos++
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].distance < pairs[j].distance
	})

	aUsed := make([]bool, m)
	bUsed := make([]bool, n)

	total := float32(0)
	count := 0

	for _, pair := range pairs {
		if pair.x < m && aUsed[pair.x] {
			continue
		}
		if pair.y < n && bUsed[pair.y] {
			continue
		}
		total += pair.distance
		count += 1
		if pair.x < m {
			aUsed[pair.x] = true
		}
		if pair.y < n {
			bUsed[pair.y] = true
		}

	}

	return total / float32(count)
}

// Calculate the distance (inverse of similarity) between two maps,
// as a floating-point number between 0 (identical) and 1 (completely dissimilar).
func mapDistance(a, b cty.Value) float32 {
	allKeys := make([]cty.Value, 0, a.LengthInt()+b.LengthInt())
	for _, m := range []cty.Value{a, b} {
		for it := m.ElementIterator(); it.Next(); {
			key, _ := it.Element()
			allKeys = append(allKeys, key)
		}
	}
	allKeySet := cty.SetVal(allKeys)

	total := float32(0)

	for it := allKeySet.ElementIterator(); it.Next(); {
		val, _ := it.Element()
		switch {
		case !a.HasIndex(val).True():
			total += 1
		case !b.HasIndex(val).True():
			total += 1
		default:
			total += valueDistance(
				a.Index(val),
				b.Index(val),
			)
		}
	}

	return total / float32(allKeySet.LengthInt())
}

// Calculate the distance (inverse of similarity) between two objects,
// as a floating-point number between 0 (identical) and 1 (completely dissimilar).
func objectDistance(a, b cty.Value) float32 {
	allKeys := make([]cty.Value, 0, a.LengthInt()+b.LengthInt())
	for _, m := range []cty.Value{a, b} {
		for it := m.ElementIterator(); it.Next(); {
			key, _ := it.Element()
			allKeys = append(allKeys, key)
		}
	}
	allKeySet := cty.SetVal(allKeys)

	total := float32(0)

	for it := allKeySet.ElementIterator(); it.Next(); {
		val, _ := it.Element()
		switch {
		case !a.Type().HasAttribute(val.AsString()):
			total += 1
		case !b.Type().HasAttribute(val.AsString()):
			total += 1
		default:
			total += valueDistance(
				a.GetAttr(val.AsString()),
				b.GetAttr(val.AsString()),
			)
		}
	}

	return total / float32(allKeySet.LengthInt())
}

// Calculate the distance (inverse of similarity) between two values,
// as a floating-point number between 0 (identical) and 1 (completely dissimilar).
func valueDistance(a, b cty.Value) float32 {
	ty := a.Type()

	switch {
	case !ctyTypesEqual(ty, b.Type()):
		return 1
	case !a.IsKnown() || !b.IsKnown():
		return 1
	case a.Equals(b) == cty.True:
		return 0
	case ty.IsPrimitiveType():
		return 1
	case ty.IsListType() || ty.IsTupleType():
		return listDistance(a, b)
	case ty.IsSetType():
		return setDistance(a, b)
	case ty.IsMapType():
		return mapDistance(a, b)
	case ty.IsObjectType():
		return objectDistance(a, b)
	default:
		return 1
	}
}
