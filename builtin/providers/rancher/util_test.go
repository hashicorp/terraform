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

type LabelTestCase struct {
	Labels          map[string]interface{}
	Command         string
	ExpectedCommand string
}

var (
	HostLabelTestCases = []LabelTestCase{
		LabelTestCase{
			Labels: map[string]interface{}{
				"orch": "true",
				"etcd": "true",
			},
			Command:         "sudo docker run --rm --privileged -v /var/run/docker.sock:/var/run/docker.sock -v /var/lib/rancher:/var/lib/rancher rancher/agent:v1.2.2 http://192.168.122.158:8080/v1/scripts/71FF294EA7A2B6865708:1483142400000:8OVFmSEUlS2VXvVGbYCXTFaMC8w",
			ExpectedCommand: "sudo docker run -e CATTLE_HOST_LABELS='etcd=true&orch=true' --rm --privileged -v /var/run/docker.sock:/var/run/docker.sock -v /var/lib/rancher:/var/lib/rancher rancher/agent:v1.2.2 http://192.168.122.158:8080/v1/scripts/71FF294EA7A2B6865708:1483142400000:8OVFmSEUlS2VXvVGbYCXTFaMC8w",
		},
		LabelTestCase{
			Labels:          map[string]interface{}{},
			Command:         "sudo docker run --rm --privileged -v /var/run/docker.sock:/var/run/docker.sock -v /var/lib/rancher:/var/lib/rancher rancher/agent:v1.2.2 http://192.168.122.158:8080/v1/scripts/71FF294EA7A2B6865708:1483142400000:8OVFmSEUlS2VXvVGbYCXTFaMC8w",
			ExpectedCommand: "sudo docker run --rm --privileged -v /var/run/docker.sock:/var/run/docker.sock -v /var/lib/rancher:/var/lib/rancher rancher/agent:v1.2.2 http://192.168.122.158:8080/v1/scripts/71FF294EA7A2B6865708:1483142400000:8OVFmSEUlS2VXvVGbYCXTFaMC8w",
		},
	}
)

func TestAddHostLabels(t *testing.T) {
	for _, tCase := range HostLabelTestCases {
		cmd := addHostLabels(tCase.Command, tCase.Labels)
		if cmd != tCase.ExpectedCommand {
			t.Errorf("Command:\n%s\nDoes not match\n%s", cmd, tCase.ExpectedCommand)
		}
	}
}
