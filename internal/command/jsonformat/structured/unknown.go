package structured

func (change Change) IsUnknown() bool {
	if unknown, ok := change.Unknown.(bool); ok {
		return unknown
	}
	return false
}
