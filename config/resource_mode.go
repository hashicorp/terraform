package config

//go:generate stringer -type=ResourceMode -output=resource_mode_string.go resource_mode.go
type ResourceMode int

const (
	ManagedResourceMode ResourceMode = iota
	DataResourceMode
)

func (m ResourceMode) MarshalJSON() ([]byte, error) {
	switch m {
	case ManagedResourceMode:
		return []byte(`"managed"`), nil
	case DataResourceMode:
		return []byte(`"data"`), nil
	default:
		return []byte(`"invalid"`), nil
	}
}
