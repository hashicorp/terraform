package terraform

import (
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
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
					ProviderAddr: addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("aws"),
						Module:   addrs.RootModule,
					},
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

func TestDiffTransformer_noOpChange(t *testing.T) {
	// "No-op" changes are how we record explicitly in a plan that we did
	// indeed visit a particular resource instance during the planning phase
	// and concluded that no changes were needed, as opposed to the resource
	// instance not existing at all or having been excluded from planning
	// entirely.
	//
	// We must include nodes for resource instances with no-op changes in the
	// apply graph, even though they won't take any external actions, because
	// there are some secondary effects such as precondition/postcondition
	// checks that can refer to objects elsewhere and so might have their
	// results changed even if the resource instance they are attached to
	// didn't actually change directly itself.

	// aws_instance.foo has a precondition, so should be included in the final
	// graph. aws_instance.bar has no conditions, so there is nothing to
	// execute during apply and it should not be included in the graph.
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "aws_instance" "bar" {
}

resource "aws_instance" "foo" {
  test_string = "ok"

  lifecycle {
	precondition {
		condition     = self.test_string != ""
		error_message = "resource error"
	}
  }
}
`})

	g := Graph{Path: addrs.RootModuleInstance}

	beforeVal, err := plans.NewDynamicValue(cty.StringVal(""), cty.String)
	if err != nil {
		t.Fatal(err)
	}

	tf := &DiffTransformer{
		Config: m,
		Changes: &plans.Changes{
			Resources: []*plans.ResourceInstanceChangeSrc{
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "aws_instance",
						Name: "foo",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					ProviderAddr: addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("aws"),
						Module:   addrs.RootModule,
					},
					ChangeSrc: plans.ChangeSrc{
						// A "no-op" change has the no-op action and has the
						// same object as both Before and After.
						Action: plans.NoOp,
						Before: beforeVal,
						After:  beforeVal,
					},
				},
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "aws_instance",
						Name: "bar",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					ProviderAddr: addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("aws"),
						Module:   addrs.RootModule,
					},
					ChangeSrc: plans.ChangeSrc{
						// A "no-op" change has the no-op action and has the
						// same object as both Before and After.
						Action: plans.NoOp,
						Before: beforeVal,
						After:  beforeVal,
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
