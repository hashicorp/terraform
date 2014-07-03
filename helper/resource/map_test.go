package resource

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestMapResources(t *testing.T) {
	m := &Map{
		Mapping: map[string]Resource{
			"aws_elb":      Resource{},
			"aws_instance": Resource{},
		},
	}

	rts := m.Resources()

	expected := []terraform.ResourceType{
		terraform.ResourceType{
			Name: "aws_elb",
		},
		terraform.ResourceType{
			Name: "aws_instance",
		},
	}

	if !reflect.DeepEqual(rts, expected) {
		t.Fatalf("bad: %#v", rts)
	}
}
