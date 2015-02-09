package terraform

import (
	"testing"
)

func TestEvalInitProvisioner_impl(t *testing.T) {
	var _ EvalNode = new(EvalInitProvisioner)
}

func TestEvalInitProvisioner(t *testing.T) {
	n := &EvalInitProvisioner{Name: "foo"}
	provisioner := &MockResourceProvisioner{}
	ctx := &MockEvalContext{InitProvisionerProvisioner: provisioner}
	if actual, err := n.Eval(ctx, nil); err != nil {
		t.Fatalf("err: %s", err)
	} else if actual != provisioner {
		t.Fatalf("bad: %#v", actual)
	}

	if !ctx.InitProvisionerCalled {
		t.Fatal("should be called")
	}
	if ctx.InitProvisionerName != "foo" {
		t.Fatalf("bad: %#v", ctx.InitProvisionerName)
	}
}

func TestEvalGetProvisioner_impl(t *testing.T) {
	var _ EvalNode = new(EvalGetProvisioner)
}

func TestEvalGetProvisioner(t *testing.T) {
	n := &EvalGetProvisioner{Name: "foo"}
	provisioner := &MockResourceProvisioner{}
	ctx := &MockEvalContext{ProvisionerProvisioner: provisioner}
	if actual, err := n.Eval(ctx, nil); err != nil {
		t.Fatalf("err: %s", err)
	} else if actual != provisioner {
		t.Fatalf("bad: %#v", actual)
	}

	if !ctx.ProvisionerCalled {
		t.Fatal("should be called")
	}
	if ctx.ProvisionerName != "foo" {
		t.Fatalf("bad: %#v", ctx.ProvisionerName)
	}
}
