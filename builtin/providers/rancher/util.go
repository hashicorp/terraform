package rancher

import "github.com/rancher/go-rancher/client"

const (
	REMOVED = "removed"
	PURGED  = "purged"
)

// GetActiveOrchestration get the name of the active orchestration for a environment
func GetActiveOrchestration(project *client.Project) string {
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
	return state == REMOVED || state == PURGED
}
