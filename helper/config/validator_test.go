package config

import (
	"fmt"
	"testing"

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
	testValid(v, c)

	// Valid + optional
	c = testConfig(t, map[string]interface{}{
		"foo": "bar",
		"bar": "baz",
	})
	testValid(v, c)

	// Missing required
	c = testConfig(t, map[string]interface{}{
		"bar": "baz",
	})
	testInvalid(v, c)

	// Unknown key
	c = testConfig(t, map[string]interface{}{
		"foo":  "bar",
		"what": "what",
	})
	testInvalid(v, c)
}

func TestValidator_array(t *testing.T) {
	v := &Validator{
		Required: []string{
			"foo",
			"nested.*",
		},
	}

	var c *terraform.ResourceConfig

	// Valid
	c = testConfig(t, map[string]interface{}{
		"foo":    "bar",
		"nested": []interface{}{"foo", "bar"},
	})
	testValid(v, c)

	// Not a nested structure
	c = testConfig(t, map[string]interface{}{
		"foo":    "bar",
		"nested": "baa",
	})
	testInvalid(v, c)
}

func TestValidator_complex(t *testing.T) {
	v := &Validator{
		Required: []string{
			"foo",
			"nested.*",
		},
	}

	var c *terraform.ResourceConfig

	// Valid
	c = testConfig(t, map[string]interface{}{
		"foo": "bar",
		"nested": []interface{}{
			map[string]interface{}{"foo": "bar"},
		},
	})
	testValid(v, c)

	// Not a nested structure
	c = testConfig(t, map[string]interface{}{
		"foo":    "bar",
		"nested": "baa",
	})
	testInvalid(v, c)
}

func TestValidator_complexNested(t *testing.T) {
	v := &Validator{
		Required: []string{
			"ingress.*",
			"ingress.*.from_port",
		},

		Optional: []string{
			"ingress.*.cidr_blocks.*",
		},
	}

	var c *terraform.ResourceConfig

	// Valid
	c = testConfig(t, map[string]interface{}{
		"ingress": []interface{}{
			map[string]interface{}{
				"from_port": "80",
			},
		},
	})
	testValid(v, c)

	// Valid
	c = testConfig(t, map[string]interface{}{
		"ingress": []interface{}{
			map[string]interface{}{
				"from_port":   "80",
				"cidr_blocks": []interface{}{"foo"},
			},
		},
	})
	testValid(v, c)
}

func TestValidator_complexDeepRequired(t *testing.T) {
	v := &Validator{
		Required: []string{
			"foo",
			"nested.*.foo",
		},
	}

	var c *terraform.ResourceConfig

	// Valid
	c = testConfig(t, map[string]interface{}{
		"foo": "bar",
		"nested": []interface{}{
			map[string]interface{}{"foo": "bar"},
		},
	})
	testValid(v, c)

	// Valid
	c = testConfig(t, map[string]interface{}{
		"foo": "bar",
	})
	testInvalid(v, c)

	// Not a nested structure
	c = testConfig(t, map[string]interface{}{
		"foo":    "bar",
		"nested": "baa",
	})
	testInvalid(v, c)
}

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	return terraform.NewResourceConfigRaw(c)
}

func testInvalid(v *Validator, c *terraform.ResourceConfig) {
	ws, es := v.Validate(c)
	if len(ws) > 0 {
		panic(fmt.Sprintf("bad: %#v", ws))
	}
	if len(es) == 0 {
		panic(fmt.Sprintf("bad: %#v", es))
	}
}

func testValid(v *Validator, c *terraform.ResourceConfig) {
	ws, es := v.Validate(c)
	if len(ws) > 0 {
		panic(fmt.Sprintf("bad: %#v", ws))
	}
	if len(es) > 0 {
		estrs := make([]string, len(es))
		for i, e := range es {
			estrs[i] = e.Error()
		}
		panic(fmt.Sprintf("bad: %#v", estrs))
	}
}
