package terraform

import (
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
)

func TestDiffTransformer_nilDiff(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	tf := &DiffTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(g.Vertices()) > 0 {
		t.Fatal("graph should be empty")
	}
}

func TestDiffTransformer(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}

	beforeVal, err := plans.NewDynamicValue(cty.StringVal(""), cty.String)
	if err != nil {
		t.Fatal(err)
	}
	afterVal, err := plans.NewDynamicValue(cty.StringVal(""), cty.String)
	if err != nil {
		t.Fatal(err)
	}

	tf := &DiffTransformer{
		Changes: &plans.Changes{
			Resources: []*plans.ResourceInstanceChangeSrc{
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "aws_instance",
						Name: "foo",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					ProviderAddr: addrs.ProviderConfig{
						Type: "aws",
					}.Absolute(addrs.RootModuleInstance),
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Update,
						Before: beforeVal,
						After:  afterVal,
					},
				},
			},
		},
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDiffBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformDiffBasicStr = `
aws_instance.foo
`
