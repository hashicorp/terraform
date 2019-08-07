package terraform

//go:generate stringer -type=ResourceMode -output=resource_mode_string.go resource_mode.go
type ResourceMode int

const (
	ManagedResourceMode ResourceMode = iota
	DataResourceMode
)
