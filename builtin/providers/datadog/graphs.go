package datadog

import (
	"github.com/zorkian/go-datadog-api"
)

// createPlaceholderGraph returns a slice with one mandatory graph. Useful to create new dashboard, as the API
// mandates one or more graphs.
func createPlaceholderGraph() []datadog.Graph {
	// Return a dummy placeholder graph.
	// This should be used when creating new dashboards, or removing the last
	// in a board.
	// Background; An API call to create or update dashboards (Timeboards) will
	// fail if it contains zero graphs. This is a bug in the Datadog API,
	// as dashboards *can* exist without any graphs.

	graphDefinition := datadog.Graph{}.Definition
	graphDefinition.Viz = "timeseries"
	r := datadog.Graph{}.Definition.Requests
	graphDefinition.Requests = append(r, graphDefintionRequests{Query: "avg:system.mem.free{*}", Stacked: false})
	graph := datadog.Graph{Title: "Mandatory placeholder graph", Definition: graphDefinition}
	graphs := []datadog.Graph{}
	graphs = append(graphs, graph)
	return graphs
}
