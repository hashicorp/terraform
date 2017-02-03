package helper

import "regexp"

// validUUID is used to check if a given string looks like a UUID
var validUUID = regexp.MustCompile(`(?i)^[\da-f]{8}-[\da-f]{4}-[\da-f]{4}-[\da-f]{4}-[\da-f]{12}$`)

// IsUUID returns true if the given string is a valid UUID.
func IsUUID(str string) bool {
	const uuidLen = 36
	if len(str) != uuidLen {
		return false
	}

	return validUUID.MatchString(str)
}

// boolToPtr returns the pointer to a boolean
func BoolToPtr(b bool) *bool {
	return &b
}

// MapStringStringSliceValueSet returns the set of values in a map[string][]string
func MapStringStringSliceValueSet(m map[string][]string) []string {
	set := make(map[string]struct{})
	for _, slice := range m {
		for _, v := range slice {
			set[v] = struct{}{}
		}
	}

	flat := make([]string, 0, len(set))
	for k := range set {
		flat = append(flat, k)
	}
	return flat
}

func SliceStringToSet(s []string) map[string]struct{} {
	m := make(map[string]struct{}, (len(s)+1)/2)
	for _, k := range s {
		m[k] = struct{}{}
	}
	return m
}

// SliceStringIsSubset returns whether the smaller set of strings is a subset of
// the larger. If the smaller slice is not a subset, the offending elements are
// returned.
func SliceStringIsSubset(larger, smaller []string) (bool, []string) {
	largerSet := make(map[string]struct{}, len(larger))
	for _, l := range larger {
		largerSet[l] = struct{}{}
	}

	subset := true
	var offending []string
	for _, s := range smaller {
		if _, ok := largerSet[s]; !ok {
			subset = false
			offending = append(offending, s)
		}
	}

	return subset, offending
}

func SliceSetDisjoint(first, second []string) (bool, []string) {
	contained := make(map[string]struct{}, len(first))
	for _, k := range first {
		contained[k] = struct{}{}
	}

	offending := make(map[string]struct{})
	for _, k := range second {
		if _, ok := contained[k]; ok {
			offending[k] = struct{}{}
		}
	}

	if len(offending) == 0 {
		return true, nil
	}

	flattened := make([]string, 0, len(offending))
	for k := range offending {
		flattened = append(flattened, k)
	}
	return false, flattened
}

// Helpers for copying generic structures.
func CopyMapStringString(m map[string]string) map[string]string {
	l := len(m)
	if l == 0 {
		return nil
	}

	c := make(map[string]string, l)
	for k, v := range m {
		c[k] = v
	}
	return c
}

func CopyMapStringInt(m map[string]int) map[string]int {
	l := len(m)
	if l == 0 {
		return nil
	}

	c := make(map[string]int, l)
	for k, v := range m {
		c[k] = v
	}
	return c
}

func CopyMapStringFloat64(m map[string]float64) map[string]float64 {
	l := len(m)
	if l == 0 {
		return nil
	}

	c := make(map[string]float64, l)
	for k, v := range m {
		c[k] = v
	}
	return c
}

func CopySliceString(s []string) []string {
	l := len(s)
	if l == 0 {
		return nil
	}

	c := make([]string, l)
	for i, v := range s {
		c[i] = v
	}
	return c
}

func CopySliceInt(s []int) []int {
	l := len(s)
	if l == 0 {
		return nil
	}

	c := make([]int, l)
	for i, v := range s {
		c[i] = v
	}
	return c
}
