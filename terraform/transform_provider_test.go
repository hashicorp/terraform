package terraform

import (
	"strings"
	"testing"
)

func TestProviderTransformer(t *testing.T) {
	mod := testModule(t, "transform-provider-basic")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	transform := &ProviderTransformer{}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformProviderBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformProviderBasicStr = `
aws_instance.web
  provider.aws
provider.aws
`
