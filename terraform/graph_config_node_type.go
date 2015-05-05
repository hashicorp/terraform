package terraform

//go:generate stringer -type=GraphNodeConfigType graph_config_node_type.go

// GraphNodeConfigType is an enum for the type of thing that a graph
// node represents from the configuration.
type GraphNodeConfigType int

const (
	GraphNodeConfigTypeInvalid  GraphNodeConfigType = 0
	GraphNodeConfigTypeResource GraphNodeConfigType = iota
	GraphNodeConfigTypeProvider
	GraphNodeConfigTypeModule
	GraphNodeConfigTypeOutput
	GraphNodeConfigTypeVariable
)
