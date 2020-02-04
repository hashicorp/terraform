package config

//go:generate go run golang.org/x/tools/cmd/stringer -type=ResourceMode -output=resource_mode_string.go resource_mode.go
type ResourceMode int

const (
	ManagedResourceMode ResourceMode = iota
	DataResourceMode
)
