package terraform

import (
	"testing"
)

func TestProvisionerUIOutput_impl(t *testing.T) {
	var _ UIOutput = new(ProvisionerUIOutput)
}

func TestProvisionerUIOutputOutput(t *testing.T) {
	hook := new(MockHook)
	output := &ProvisionerUIOutput{
		Info:  nil,
		Type:  "foo",
		Hooks: []Hook{hook},
	}

	output.Output("bar")

	if !hook.ProvisionOutputCalled {
		t.Fatal("should be called")
	}
	if hook.ProvisionOutputProvisionerId != "foo" {
		t.Fatalf("bad: %#v", hook.ProvisionOutputProvisionerId)
	}
	if hook.ProvisionOutputMessage != "bar" {
		t.Fatalf("bad: %#v", hook.ProvisionOutputMessage)
	}
}
