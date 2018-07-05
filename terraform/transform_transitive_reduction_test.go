package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

func TestTransitiveReductionTransformer(t *testing.T) {
	mod := testModule(t, "transform-trans-reduce-basic")

	g := Graph{Path: addrs.RootModuleInstance}
	{
		tf := &ConfigTransformer{Config: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
		t.Logf("graph after ConfigTransformer:\n%s", g.String())
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &AttachSchemaTransformer{
			Schemas: &Schemas{
				providers: map[string]*ProviderSchema{
					"aws": {
						ResourceTypes: map[string]*configschema.Block{
							"aws_instance": &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"A": {
										Type:     cty.String,
										Optional: true,
									},
									"B": {
										Type:     cty.String,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
		}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &ReferenceTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
		t.Logf("graph after ReferenceTransformer:\n%s", g.String())
	}

	{
		transform := &TransitiveReductionTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
		t.Logf("graph after TransitiveReductionTransformer:\n%s", g.String())
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformTransReduceBasicStr)
	if actual != expected {
		t.Errorf("wrong result\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

const testTransformTransReduceBasicStr = `
aws_instance.A
aws_instance.B
  aws_instance.A
aws_instance.C
  aws_instance.B
`
