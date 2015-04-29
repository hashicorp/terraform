package terraform

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

func TestGraphNodeConfigModule_impl(t *testing.T) {
	var _ dag.Vertex = new(GraphNodeConfigModule)
	var _ dag.NamedVertex = new(GraphNodeConfigModule)
	var _ graphNodeConfig = new(GraphNodeConfigModule)
	var _ GraphNodeExpandable = new(GraphNodeConfigModule)
}

func TestGraphNodeConfigModuleExpand(t *testing.T) {
	mod := testModule(t, "graph-node-module-expand")

	node := &GraphNodeConfigModule{
		Path:   []string{RootModuleName, "child"},
		Module: &config.Module{},
		Tree:   nil,
	}

	g, err := node.Expand(&BasicGraphBuilder{
		Steps: []GraphTransformer{
			&ConfigTransformer{Module: mod},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.Subgraph().String())
	expected := strings.TrimSpace(testGraphNodeModuleExpandStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestGraphNodeConfigOutput_impl(t *testing.T) {
	var _ dag.Vertex = new(GraphNodeConfigOutput)
	var _ dag.NamedVertex = new(GraphNodeConfigOutput)
	var _ graphNodeConfig = new(GraphNodeConfigOutput)
	var _ GraphNodeOutput = new(GraphNodeConfigOutput)
}

func TestGraphNodeConfigProvider_impl(t *testing.T) {
	var _ dag.Vertex = new(GraphNodeConfigProvider)
	var _ dag.NamedVertex = new(GraphNodeConfigProvider)
	var _ graphNodeConfig = new(GraphNodeConfigProvider)
	var _ GraphNodeProvider = new(GraphNodeConfigProvider)
}

func TestGraphNodeConfigProvider_ProviderName(t *testing.T) {
	n := &GraphNodeConfigProvider{
		Provider: &config.ProviderConfig{Name: "foo"},
	}

	if v := n.ProviderName(); v != "foo" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestGraphNodeConfigProvider_ProviderName_alias(t *testing.T) {
	n := &GraphNodeConfigProvider{
		Provider: &config.ProviderConfig{Name: "foo", Alias: "bar"},
	}

	if v := n.ProviderName(); v != "foo.bar" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestGraphNodeConfigProvider_Name(t *testing.T) {
	n := &GraphNodeConfigProvider{
		Provider: &config.ProviderConfig{Name: "foo"},
	}

	if v := n.Name(); v != "provider.foo" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestGraphNodeConfigProvider_Name_alias(t *testing.T) {
	n := &GraphNodeConfigProvider{
		Provider: &config.ProviderConfig{Name: "foo", Alias: "bar"},
	}

	if v := n.Name(); v != "provider.foo.bar" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestGraphNodeConfigResource_impl(t *testing.T) {
	var _ dag.Vertex = new(GraphNodeConfigResource)
	var _ dag.NamedVertex = new(GraphNodeConfigResource)
	var _ graphNodeConfig = new(GraphNodeConfigResource)
	var _ GraphNodeProviderConsumer = new(GraphNodeConfigResource)
	var _ GraphNodeProvisionerConsumer = new(GraphNodeConfigResource)
}

func TestGraphNodeConfigResource_ProvidedBy(t *testing.T) {
	n := &GraphNodeConfigResource{
		Resource: &config.Resource{Type: "aws_instance"},
	}

	if v := n.ProvidedBy(); v[0] != "aws" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestGraphNodeConfigResource_ProvidedBy_alias(t *testing.T) {
	n := &GraphNodeConfigResource{
		Resource: &config.Resource{Type: "aws_instance", Provider: "aws.bar"},
	}

	if v := n.ProvidedBy(); v[0] != "aws.bar" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestGraphNodeConfigResource_ProvisionedBy(t *testing.T) {
	n := &GraphNodeConfigResource{
		Resource: &config.Resource{
			Type: "aws_instance",
			Provisioners: []*config.Provisioner{
				&config.Provisioner{Type: "foo"},
				&config.Provisioner{Type: "bar"},
			},
		},
	}

	expected := []string{"foo", "bar"}
	actual := n.ProvisionedBy()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

const testGraphNodeModuleExpandStr = `
aws_instance.bar
  aws_instance.foo
aws_instance.foo
  module inputs
module inputs
`
