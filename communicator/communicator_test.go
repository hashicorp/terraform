package communicator

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestCommunicator_new(t *testing.T) {
	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type": "telnet",
			},
		},
	}
	if _, err := New(r); err == nil {
		t.Fatalf("expected error with telnet")
	}

	r.Ephemeral.ConnInfo["type"] = "ssh"
	if _, err := New(r); err != nil {
		t.Fatalf("err: %v", err)
	}

	r.Ephemeral.ConnInfo["type"] = "winrm"
	if _, err := New(r); err != nil {
		t.Fatalf("err: %v", err)
	}
}
