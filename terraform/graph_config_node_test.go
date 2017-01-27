package terraform

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

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
