package docker

import (
	"testing"

	"time"

	"github.com/hashicorp/terraform/terraform"
)

func TestProvisioner_connInfoDefaults(t *testing.T) {
	expectedHost := "unix:///var/run/docker.sock"
	expectedContainerId := "045c63979a"
	r := &terraform.InstanceState{
		ID: expectedContainerId,
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type": "docker",
				"host": expectedHost,
			},
		},
	}

	conf, err := parseConnectionInfo(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if conf.Host != expectedHost {
		t.Fatalf("bad: %v", conf)
	}
	if conf.ContainerId != expectedContainerId {
		t.Fatalf("bad: %v", conf)
	}
	if conf.ScriptPath != DefaultScriptPath {
		t.Fatalf("bad: %v", conf)
	}
	if conf.TimeoutVal != DefaultTimeout {
		t.Fatalf("bad: %v", conf)
	}
}
func TestProvisioner_connInfoCustomized(t *testing.T) {
	expectedHost := "tcp://mytest.com:2375"
	expectedContainerId := "045c63979a47"
	expectedScriptPath := "/tmp/terraform_test.sh"
	expectedTimeout := "42s"
	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":        "docker",
				"host":        expectedHost,
				"containerId": expectedContainerId,
				"script_path": expectedScriptPath,
				"timeout":     expectedTimeout,
			},
		},
	}

	conf, err := parseConnectionInfo(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if conf.Host != expectedHost {
		t.Fatalf("bad: %v", conf)
	}
	if conf.ContainerId != expectedContainerId {
		t.Fatalf("bad: %v", conf)
	}
	if conf.ScriptPath != expectedScriptPath {
		t.Fatalf("bad: %v", conf)
	}
	if duration, err := time.ParseDuration(expectedTimeout); err != nil || conf.TimeoutVal != duration {
		t.Fatalf("bad: %v", conf)
	}
}
