package terraform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
)

func testProviderTransformerGraph(t *testing.T, cfg *configs.Config) *Graph {
	t.Helper()

	g := &Graph{Path: addrs.RootModuleInstance}
	ct := &ConfigTransformer{Config: cfg}
	if err := ct.Transform(g); err != nil {
		t.Fatal(err)
	}
	arct := &AttachResourceConfigTransformer{Config: cfg}
	if err := arct.Transform(g); err != nil {
		t.Fatal(err)
	}

	return g
}

func TestProviderTransformer(t *testing.T) {
	mod := testModule(t, "transform-provider-basic")

	g := testProviderTransformerGraph(t, mod)
	{
		transform := &MissingProviderTransformer{}
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	transform := &ProviderTransformer{}
	if err := transform.Transform(g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformProviderBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestProviderTransformer_ImportModuleChild(t *testing.T) {
	mod := testModule(t, "import-module")

	g := testProviderTransformerGraph(t, mod)

	{
		tf := &ImportStateTransformer{
			Config: mod,
			Targets: []*ImportTarget{
				&ImportTarget{
					Addr: addrs.RootModuleInstance.
						Child("child", addrs.NoKey).
						ResourceInstance(
							addrs.ManagedResourceMode,
							"aws_instance",
							"foo",
							addrs.NoKey,
						),
					ID: "bar",
				},
			},
		}

		if err := tf.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
		t.Logf("graph after ImportStateTransformer:\n%s", g.String())
	}

	{
		tf := &MissingProviderTransformer{}
		if err := tf.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
		t.Logf("graph after MissingProviderTransformer:\n%s", g.String())
	}

	{
		tf := &ProviderTransformer{}
		if err := tf.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
		t.Logf("graph after ProviderTransformer:\n%s", g.String())
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformImportModuleChildStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// Test providers with FQNs that do not match the typeName
func TestProviderTransformer_fqns(t *testing.T) {
	for _, mod := range []string{"fqns", "fqns-module"} {
		mod := testModule(t, fmt.Sprintf("transform-provider-%s", mod))

		g := testProviderTransformerGraph(t, mod)
		{
			transform := &MissingProviderTransformer{Config: mod}
			if err := transform.Transform(g); err != nil {
				t.Fatalf("err: %s", err)
			}
		}

		transform := &ProviderTransformer{Config: mod}
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}

		actual := strings.TrimSpace(g.String())
		expected := strings.TrimSpace(testTransformProviderBasicStr)
		if actual != expected {
			t.Fatalf("bad:\n\n%s", actual)
		}
	}
}

func TestCloseProviderTransformer(t *testing.T) {
	mod := testModule(t, "transform-provider-basic")
	g := testProviderTransformerGraph(t, mod)

	{
		transform := &MissingProviderTransformer{}
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &ProviderTransformer{}
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &CloseProviderTransformer{}
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformCloseProviderBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestCloseProviderTransformer_withTargets(t *testing.T) {
	mod := testModule(t, "transform-provider-basic")

	g := testProviderTransformerGraph(t, mod)
	transforms := []GraphTransformer{
		&MissingProviderTransformer{},
		&ProviderTransformer{},
		&CloseProviderTransformer{},
		&TargetsTransformer{
			Targets: []addrs.Targetable{
				addrs.RootModuleInstance.Resource(
					addrs.ManagedResourceMode, "something", "else",
				),
			},
		},
	}

	for _, tr := range transforms {
		if err := tr.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(``)
	if actual != expected {
		t.Fatalf("expected:%s\n\ngot:\n\n%s", expected, actual)
	}
}

func TestMissingProviderTransformer(t *testing.T) {
	mod := testModule(t, "transform-provider-missing")

	g := testProviderTransformerGraph(t, mod)
	{
		transform := &MissingProviderTransformer{}
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &ProviderTransformer{}
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &CloseProviderTransformer{}
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformMissingProviderBasicStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestMissingProviderTransformer_grandchildMissing(t *testing.T) {
	mod := testModule(t, "transform-provider-missing-grandchild")

	concrete := func(a *NodeAbstractProvider) dag.Vertex { return a }

	g := testProviderTransformerGraph(t, mod)
	{
		transform := transformProviders(concrete, mod)
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}
	{
		transform := &TransitiveReductionTransformer{}
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformMissingGrandchildProviderStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestPruneProviderTransformer(t *testing.T) {
	mod := testModule(t, "transform-provider-prune")

	g := testProviderTransformerGraph(t, mod)
	{
		transform := &MissingProviderTransformer{}
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &ProviderTransformer{}
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &CloseProviderTransformer{}
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &PruneProviderTransformer{}
		if err := transform.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformPruneProviderBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

// the child module resource is attached to the configured parent provider
func TestProviderConfigTransformer_parentProviders(t *testing.T) {
	mod := testModule(t, "transform-provider-inherit")
	concrete := func(a *NodeAbstractProvider) dag.Vertex { return a }

	g := testProviderTransformerGraph(t, mod)
	{
		tf := transformProviders(concrete, mod)
		if err := tf.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformModuleProviderConfigStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

// the child module resource is attached to the configured grand-parent provider
func TestProviderConfigTransformer_grandparentProviders(t *testing.T) {
	mod := testModule(t, "transform-provider-grandchild-inherit")
	concrete := func(a *NodeAbstractProvider) dag.Vertex { return a }

	g := testProviderTransformerGraph(t, mod)
	{
		tf := transformProviders(concrete, mod)
		if err := tf.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformModuleProviderGrandparentStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestProviderConfigTransformer_inheritOldSkool(t *testing.T) {
	mod := testModuleInline(t, map[string]string{
		"main.tf": `
provider "test" {
  test_string = "config"
}

module "moda" {
  source = "./moda"
}
`,

		"moda/main.tf": `
resource "test_object" "a" {
}
`,
	})
	concrete := func(a *NodeAbstractProvider) dag.Vertex { return a }

	g := testProviderTransformerGraph(t, mod)
	{
		tf := transformProviders(concrete, mod)
		if err := tf.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	expected := `module.moda.test_object.a
  provider["registry.terraform.io/hashicorp/test"]
provider["registry.terraform.io/hashicorp/test"]`

	actual := strings.TrimSpace(g.String())
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

// Verify that configurations which are not recommended yet supported still work
func TestProviderConfigTransformer_nestedModuleProviders(t *testing.T) {
	mod := testModuleInline(t, map[string]string{
		"main.tf": `
terraform {
  required_providers {
    test = {
      source = "registry.terraform.io/hashicorp/test"
	}
  }
}

provider "test" {
  alias = "z"
  test_string = "config"
}

module "moda" {
  source = "./moda"
  providers = {
    test.x = test.z
  }
}
`,

		"moda/main.tf": `
terraform {
  required_providers {
    test = {
      source = "registry.terraform.io/hashicorp/test"
      configuration_aliases = [ test.x ]
	}
  }
}

provider "test" {
  test_string = "config"
}

// this should connect to this module's provider
resource "test_object" "a" {
}

resource "test_object" "x" {
  provider = test.x
}

module "modb" {
  source = "./modb"
}
`,

		"moda/modb/main.tf": `
# this should end up with the provider from the parent module
resource "test_object" "a" {
}
`,
	})
	concrete := func(a *NodeAbstractProvider) dag.Vertex { return a }

	g := testProviderTransformerGraph(t, mod)
	{
		tf := transformProviders(concrete, mod)
		if err := tf.Transform(g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	expected := `module.moda.module.modb.test_object.a
  module.moda.provider["registry.terraform.io/hashicorp/test"]
module.moda.provider["registry.terraform.io/hashicorp/test"]
module.moda.test_object.a
  module.moda.provider["registry.terraform.io/hashicorp/test"]
module.moda.test_object.x
  provider["registry.terraform.io/hashicorp/test"].z
provider["registry.terraform.io/hashicorp/test"].z`

	actual := strings.TrimSpace(g.String())
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

const testTransformProviderBasicStr = `
aws_instance.web
  provider["registry.terraform.io/hashicorp/aws"]
provider["registry.terraform.io/hashicorp/aws"]
`

const testTransformCloseProviderBasicStr = `
aws_instance.web
  provider["registry.terraform.io/hashicorp/aws"]
provider["registry.terraform.io/hashicorp/aws"]
provider["registry.terraform.io/hashicorp/aws"] (close)
  aws_instance.web
  provider["registry.terraform.io/hashicorp/aws"]
`

const testTransformMissingProviderBasicStr = `
aws_instance.web
  provider["registry.terraform.io/hashicorp/aws"]
foo_instance.web
  provider["registry.terraform.io/hashicorp/foo"]
provider["registry.terraform.io/hashicorp/aws"]
provider["registry.terraform.io/hashicorp/aws"] (close)
  aws_instance.web
  provider["registry.terraform.io/hashicorp/aws"]
provider["registry.terraform.io/hashicorp/foo"]
provider["registry.terraform.io/hashicorp/foo"] (close)
  foo_instance.web
  provider["registry.terraform.io/hashicorp/foo"]
`

const testTransformMissingGrandchildProviderStr = `
module.sub.module.subsub.bar_instance.two
  provider["registry.terraform.io/hashicorp/bar"]
module.sub.module.subsub.foo_instance.one
  module.sub.provider["registry.terraform.io/hashicorp/foo"]
module.sub.provider["registry.terraform.io/hashicorp/foo"]
provider["registry.terraform.io/hashicorp/bar"]
`

const testTransformPruneProviderBasicStr = `
foo_instance.web
  provider["registry.terraform.io/hashicorp/foo"]
provider["registry.terraform.io/hashicorp/foo"]
provider["registry.terraform.io/hashicorp/foo"] (close)
  foo_instance.web
  provider["registry.terraform.io/hashicorp/foo"]
`

const testTransformModuleProviderConfigStr = `
module.child.aws_instance.thing
  provider["registry.terraform.io/hashicorp/aws"].foo
provider["registry.terraform.io/hashicorp/aws"].foo
`

const testTransformModuleProviderGrandparentStr = `
module.child.module.grandchild.aws_instance.baz
  provider["registry.terraform.io/hashicorp/aws"].foo
provider["registry.terraform.io/hashicorp/aws"].foo
`

const testTransformImportModuleChildStr = `        
module.child.aws_instance.foo
  provider["registry.terraform.io/hashicorp/aws"]
module.child.aws_instance.foo (import id "bar")
  provider["registry.terraform.io/hashicorp/aws"]
module.child.module.nested.aws_instance.foo
  provider["registry.terraform.io/hashicorp/aws"]
provider["registry.terraform.io/hashicorp/aws"]`
