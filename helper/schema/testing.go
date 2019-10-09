package schema

import (
	"github.com/hashicorp/terraform/terraform"
	testing "github.com/mitchellh/go-testing-interface"
)

// TestResourceDataRaw creates a ResourceData from a raw configuration map.
func TestResourceDataRaw(
	t testing.T, schema map[string]*Schema, raw map[string]interface{}) *ResourceData {
	t.Helper()

	c := terraform.NewResourceConfigRaw(raw)

	sm := schemaMap(schema)
	diff, err := sm.Diff(nil, c, nil, nil, true)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	result, err := sm.Data(nil, diff)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return result
}
