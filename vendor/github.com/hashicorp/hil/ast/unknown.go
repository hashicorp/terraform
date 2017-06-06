package ast

// IsUnknown reports whether a variable is unknown or contains any value
// that is unknown. This will recurse into lists and maps and so on.
func IsUnknown(v Variable) bool {
	// If it is unknown itself, return true
	if v.Type == TypeUnknown {
		return true
	}

	// If it is a container type, check the values
	switch v.Type {
	case TypeList:
		for _, el := range v.Value.([]Variable) {
			if IsUnknown(el) {
				return true
			}
		}
	case TypeMap:
		for _, el := range v.Value.(map[string]Variable) {
			if IsUnknown(el) {
				return true
			}
		}
	default:
	}

	// Not a container type or survive the above checks
	return false
}
