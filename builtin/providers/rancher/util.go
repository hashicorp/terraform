package rancher

import (
	"strings"

	"github.com/rancher/go-rancher/client"
)

const (
	stateRemoved = "removed"
	statePurged  = "purged"
)

// GetActiveOrchestration get the name of the active orchestration for a environment
func getActiveOrchestration(project *client.Project) string {
	orch := "cattle"

	switch {
	case project.Swarm:
		orch = "swarm"
	case project.Mesos:
		orch = "mesos"
	case project.Kubernetes:
		orch = "kubernetes"
	}

	return orch
}

func removed(state string) bool {
	return state == stateRemoved || state == statePurged
}

func splitID(id string) (envID, resourceID string) {
	if strings.Contains(id, "/") {
		return id[0:strings.Index(id, "/")], id[strings.Index(id, "/")+1:]
	}
	return "", id
}

// NewListOpts wraps around client.NewListOpts()
func NewListOpts() *client.ListOpts {
	return client.NewListOpts()
}
