package variables

// Merge merges raw variable values b into a.
//
// The parameters given here should be the full map of set variables, such
// as those created by Flag and FlagFile.
//
// The merge behavior is to override the top-level key except for map
// types. Map types are merged together by key. Any other types are overwritten:
// primitives and lists.
//
// This returns the resulting map. This merges into a but if a is nil a new
// map will be allocated. A non-nil "a" value is returned regardless.
func Merge(a, b map[string]interface{}) map[string]interface{} {
	if a == nil {
		a = map[string]interface{}{}
	}

	for k, raw := range b {
		switch v := raw.(type) {
		case map[string]interface{}:
			// For maps, we do a deep merge. If the value in the original
			// map (a) is not a map, we just overwrite. For invalid types
			// they're caught later in the validation step in Terraform.

			// If there is no value set, just set it
			rawA, ok := a[k]
			if !ok {
				a[k] = v
				continue
			}

			// If the value is not a map, just set it
			mapA, ok := rawA.(map[string]interface{})
			if !ok {
				a[k] = v
				continue
			}

			// Go over the values in the map. If we're setting a raw value,
			// then override. If we're setting a nested map, then recurse.
			for k, v := range v {
				// If the value isn't a map, then there is nothing to merge
				// further so we just set it.
				mv, ok := v.(map[string]interface{})
				if !ok {
					mapA[k] = v
					continue
				}

				switch av := mapA[k].(type) {
				case map[string]interface{}:
					mapA[k] = Merge(av, mv)
				default:
					// Unset or non-map, just set it
					mapA[k] = mv
				}
			}
		default:
			// Any other type we just set directly
			a[k] = v
		}
	}

	return a
}
