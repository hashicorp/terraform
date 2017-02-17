package datadog

// Int is a helper routine that allocates a new int value
// to store v and returns a pointer to it.
func Int(v int) *int { return &v }

// GetInt is a helper routine that returns a boolean representing
// if a value was set, and if so, dereferences the pointer to it.
func GetInt(v *int) (int, bool) {
	if v != nil {
		return *v, true
	}

	return 0, false
}
