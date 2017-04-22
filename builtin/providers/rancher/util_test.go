package rancher

import (
	"testing"

	"github.com/rancher/go-rancher/v2"
)

var idTests = []struct {
	id         string
	envID      string
	resourceID string
}{
	{"1a05", "", "1a05"},
	{"1a05/1s234", "1a05", "1s234"},
}

func TestSplitId(t *testing.T) {
	for _, tt := range idTests {
		envID, resourceID := splitID(tt.id)
		if envID != tt.envID || resourceID != tt.resourceID {
			t.Errorf("splitId(%s) => [%s, %s]) want [%s, %s]", tt.id, envID, resourceID, tt.envID, tt.resourceID)
		}
	}
}

var stateTests = []struct {
	state   string
	removed bool
}{
	{"removed", true},
	{"purged", true},
	{"active", false},
}

func TestRemovedState(t *testing.T) {
	for _, tt := range stateTests {
		removed := removed(tt.state)
		if removed != tt.removed {
			t.Errorf("removed(%s) => %t, wants %t", tt.state, removed, tt.removed)
		}
	}
}

var orchestrationTests = []struct {
	project       *client.Project
	orchestration string
}{
	{&client.Project{Orchestration: "cattle"}, "cattle"},
	{&client.Project{Orchestration: "swarm"}, "swarm"},
	{&client.Project{Orchestration: "mesos"}, "mesos"},
	{&client.Project{Orchestration: "kubernetes"}, "kubernetes"},
}

func TestActiveOrchestration(t *testing.T) {
	for _, tt := range orchestrationTests {
		orchestration := getActiveOrchestration(tt.project)
		if orchestration != tt.orchestration {
			t.Errorf("getActiveOrchestration(%+v) => %s, wants %s", tt.project, orchestration, tt.orchestration)
		}
	}
}
