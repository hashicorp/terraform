package cmd

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/zclconf/go-cty/cty"
)

func TestCmdClient_impl(t *testing.T) {
	var _ remote.Client = new(CmdClient)
}

func TestCmdFactory(t *testing.T) {
	// This test just instantiates the client. Shouldn't make any actual
	// requests nor incur any costs.

	config := make(map[string]cty.Value)

	config["base_command"] = cty.StringVal("/usr/bin/base_command")
	config["state_transfer_file"] = cty.StringVal("terraform_states_file")
	config["lock_transfer_file"] = cty.StringVal("terraform_lock_file")

	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", config))

	state, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("Error for valid config: %s", err)
	}

	cmdClient := state.(*remote.State).Client.(*CmdClient)

	if cmdClient.baseCmd != "/usr/bin/base_command" {
		t.Fatalf("Incorrect base_command was populated")
	}
	if cmdClient.statesTransferFile != "terraform_states_file" {
		t.Fatalf("Incorrect state_transfer_file was populated")
	}
	if cmdClient.lockTransferFile != "terraform_lock_file" {
		t.Fatalf("Incorrect lock_transfer_file was populated")
	}
}
