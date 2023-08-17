// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mnptu

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/mnptu/internal/addrs"
	"github.com/hashicorp/mnptu/internal/configs"
	"github.com/hashicorp/mnptu/internal/dag"
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
  provider["registry.mnptu.io/hashicorp/test"]
provider["registry.mnptu.io/hashicorp/test"]`

	actual := strings.TrimSpace(g.String())
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

// Verify that configurations which are not recommended yet supported still work
func TestProviderConfigTransformer_nestedModuleProviders(t *testing.T) {
	mod := testModuleInline(t, map[string]string{
		"main.tf": `
mnptu {
  required_providers {
    test = {
      source = "registry.mnptu.io/hashicorp/test"
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
mnptu {
  required_providers {
    test = {
      source = "registry.mnptu.io/hashicorp/test"
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
  module.moda.provider["registry.mnptu.io/hashicorp/test"]
module.moda.provider["registry.mnptu.io/hashicorp/test"]
module.moda.test_object.a
  module.moda.provider["registry.mnptu.io/hashicorp/test"]
module.moda.test_object.x
  provider["registry.mnptu.io/hashicorp/test"].z
provider["registry.mnptu.io/hashicorp/test"].z`

	actual := strings.TrimSpace(g.String())
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestProviderConfigTransformer_duplicateLocalName(t *testing.T) {
	mod := testModuleInline(t, map[string]string{
		"main.tf": `
mnptu {
  required_providers {
	# We have to allow this since it wasn't previously prevented. If the
	# default config is equivalent to the provider config, the user may never
	# see an error.
    dupe = {
      source = "registry.mnptu.io/hashicorp/test"
    }
  }
}

provider "test" {
}
`})
	concrete := func(a *NodeAbstractProvider) dag.Vertex { return a }

	g := testProviderTransformerGraph(t, mod)
	tf := ProviderConfigTransformer{
		Config:   mod,
		Concrete: concrete,
	}
	if err := tf.Transform(g); err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := `provider["registry.mnptu.io/hashicorp/test"]`

	actual := strings.TrimSpace(g.String())
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

const testTransformProviderBasicStr = `
aws_instance.web
  provider["registry.mnptu.io/hashicorp/aws"]
provider["registry.mnptu.io/hashicorp/aws"]
`

const testTransformCloseProviderBasicStr = `
aws_instance.web
  provider["registry.mnptu.io/hashicorp/aws"]
provider["registry.mnptu.io/hashicorp/aws"]
provider["registry.mnptu.io/hashicorp/aws"] (close)
  aws_instance.web
  provider["registry.mnptu.io/hashicorp/aws"]
`

const testTransformMissingProviderBasicStr = `
aws_instance.web
  provider["registry.mnptu.io/hashicorp/aws"]
foo_instance.web
  provider["registry.mnptu.io/hashicorp/foo"]
provider["registry.mnptu.io/hashicorp/aws"]
provider["registry.mnptu.io/hashicorp/aws"] (close)
  aws_instance.web
  provider["registry.mnptu.io/hashicorp/aws"]
provider["registry.mnptu.io/hashicorp/foo"]
provider["registry.mnptu.io/hashicorp/foo"] (close)
  foo_instance.web
  provider["registry.mnptu.io/hashicorp/foo"]
`

const testTransformMissingGrandchildProviderStr = `
module.sub.module.subsub.bar_instance.two
  provider["registry.mnptu.io/hashicorp/bar"]
module.sub.module.subsub.foo_instance.one
  module.sub.provider["registry.mnptu.io/hashicorp/foo"]
module.sub.provider["registry.mnptu.io/hashicorp/foo"]
provider["registry.mnptu.io/hashicorp/bar"]
`

const testTransformPruneProviderBasicStr = `
foo_instance.web
  provider["registry.mnptu.io/hashicorp/foo"]
provider["registry.mnptu.io/hashicorp/foo"]
provider["registry.mnptu.io/hashicorp/foo"] (close)
  foo_instance.web
  provider["registry.mnptu.io/hashicorp/foo"]
`

const testTransformModuleProviderConfigStr = `
module.child.aws_instance.thing
  provider["registry.mnptu.io/hashicorp/aws"].foo
provider["registry.mnptu.io/hashicorp/aws"].foo
`

const testTransformModuleProviderGrandparentStr = `
module.child.module.grandchild.aws_instance.baz
  provider["registry.mnptu.io/hashicorp/aws"].foo
provider["registry.mnptu.io/hashicorp/aws"].foo
`
