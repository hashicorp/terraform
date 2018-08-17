package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/provisioners"
)

func TestEvalInitProvisioner_impl(t *testing.T) {
	var _ EvalNode = new(EvalInitProvisioner)
}

func TestEvalInitProvisioner(t *testing.T) {
	n := &EvalInitProvisioner{Name: "foo"}
	provisioner := &MockProvisioner{}
	ctx := &MockEvalContext{InitProvisionerProvisioner: provisioner}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !ctx.InitProvisionerCalled {
		t.Fatal("should be called")
	}
	if ctx.InitProvisionerName != "foo" {
		t.Fatalf("bad: %#v", ctx.InitProvisionerName)
	}
}

func TestEvalCloseProvisioner(t *testing.T) {
	n := &EvalCloseProvisioner{Name: "foo"}
	provisioner := &MockProvisioner{}
	ctx := &MockEvalContext{CloseProvisionerProvisioner: provisioner}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !ctx.CloseProvisionerCalled {
		t.Fatal("should be called")
	}
	if ctx.CloseProvisionerName != "foo" {
		t.Fatalf("bad: %#v", ctx.CloseProvisionerName)
	}
}

func TestEvalGetProvisioner_impl(t *testing.T) {
	var _ EvalNode = new(EvalGetProvisioner)
}

func TestEvalGetProvisioner(t *testing.T) {
	var actual provisioners.Interface
	n := &EvalGetProvisioner{Name: "foo", Output: &actual}
	provisioner := &MockProvisioner{}
	ctx := &MockEvalContext{ProvisionerProvisioner: provisioner}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != provisioner {
		t.Fatalf("bad: %#v", actual)
	}

	if !ctx.ProvisionerCalled {
		t.Fatal("should be called")
	}
	if ctx.ProvisionerName != "foo" {
		t.Fatalf("bad: %#v", ctx.ProvisionerName)
	}
}
