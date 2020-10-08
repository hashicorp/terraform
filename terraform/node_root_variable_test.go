package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
)

func TestNodeRootVariableExecute(t *testing.T) {
	ctx := new(MockEvalContext)

	n := &NodeRootVariable{
		Addr: addrs.InputVariable{Name: "foo"},
		Config: &configs.Variable{
			Name: "foo",
		},
	}

	err := n.Execute(ctx, walkApply)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

}
