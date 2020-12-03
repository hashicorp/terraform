package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestProvisionerUIOutput_impl(t *testing.T) {
	var _ UIOutput = new(ProvisionerUIOutput)
}

func TestProvisionerUIOutputOutput(t *testing.T) {
	hook := new(MockHook)
	output := &ProvisionerUIOutput{
		InstanceAddr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "test",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
		ProvisionerType: "foo",
		Hooks:           []Hook{hook},
	}

	output.Output("bar")

	if !hook.ProvisionOutputCalled {
		t.Fatal("hook.ProvisionOutput was not called, and should've been")
	}
	if got, want := hook.ProvisionOutputProvisionerType, "foo"; got != want {
		t.Fatalf("wrong provisioner type\ngot:  %q\nwant: %q", got, want)
	}
	if got, want := hook.ProvisionOutputMessage, "bar"; got != want {
		t.Fatalf("wrong output message\ngot:  %q\nwant: %q", got, want)
	}
}
