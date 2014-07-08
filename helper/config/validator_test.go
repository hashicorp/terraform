package config

import (
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestValidator(t *testing.T) {
	v := &Validator{
		Required: []string{"foo"},
		Optional: []string{"bar"},
	}

	var c *terraform.ResourceConfig

	// Valid
	c = testConfig(t, map[string]interface{}{
		"foo": "bar",
	})
	testValid(t, v, c)

	// Missing required
	c = testConfig(t, map[string]interface{}{
		"bar": "baz",
	})
	testInvalid(t, v, c)

	// Unknown key
	c = testConfig(t, map[string]interface{}{
		"foo":  "bar",
		"what": "what",
	})
	testInvalid(t, v, c)
}

func testConfig(
	t *testing.T,
	c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	return terraform.NewResourceConfig(r)
}

func testInvalid(t *testing.T, v *Validator, c *terraform.ResourceConfig) {
	ws, es := v.Validate(c)
	if len(ws) > 0 {
		t.Fatalf("bad: %#v", ws)
	}
	if len(es) == 0 {
		t.Fatalf("bad: %#v", es)
	}
}

func testValid(t *testing.T, v *Validator, c *terraform.ResourceConfig) {
	ws, es := v.Validate(c)
	if len(ws) > 0 {
		t.Fatalf("bad: %#v", ws)
	}
	if len(es) > 0 {
		t.Fatalf("bad: %#v", es)
	}
}
