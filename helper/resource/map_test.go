package resource

import (
	"reflect"
	"testing"

	tfconfig "github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/config"
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

func TestMapValidate(t *testing.T) {
	m := &Map{
		Mapping: map[string]Resource{
			"aws_elb": Resource{
				ConfigValidator: &config.Validator{
					Required: []string{"foo"},
				},
			},
		},
	}

	var c *terraform.ResourceConfig
	var ws []string
	var es []error

	// Valid
	c = testConfig(t, map[string]interface{}{"foo": "bar"})
	ws, es = m.Validate("aws_elb", c)
	if len(ws) > 0 {
		t.Fatalf("bad: %#v", ws)
	}
	if len(es) > 0 {
		t.Fatalf("bad: %#v", es)
	}

	// Invalid
	c = testConfig(t, map[string]interface{}{})
	ws, es = m.Validate("aws_elb", c)
	if len(ws) > 0 {
		t.Fatalf("bad: %#v", ws)
	}
	if len(es) == 0 {
		t.Fatalf("bad: %#v", es)
	}
}

func testConfig(
	t *testing.T,
	c map[string]interface{}) *terraform.ResourceConfig {
	r, err := tfconfig.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	return terraform.NewResourceConfig(r)
}
