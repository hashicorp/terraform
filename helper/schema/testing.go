package schema

import (
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

// TestResourceDataRaw creates a ResourceData from a raw configuration map.
func TestResourceDataRaw(
	t *testing.T, schema map[string]*Schema, raw map[string]interface{}) *ResourceData {
	return TestResourceDataStateRaw(t, schema, nil, raw)
}

// TestResourceDataStateRaw creates a ResourceData from an instance state map
// and a raw configuration map.
func TestResourceDataStateRaw(t *testing.T, schema map[string]*Schema,
	state map[string]string, raw map[string]interface{}) *ResourceData {
	t.Helper()

	c, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	sm := schemaMap(schema)
	diff, err := sm.Diff(nil, terraform.NewResourceConfig(c), nil, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var instanceState *terraform.InstanceState = nil
	if state != nil {
		instanceState = &terraform.InstanceState{
			Attributes: state,
		}
	}
	result, err := sm.Data(instanceState, diff)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return result
}
