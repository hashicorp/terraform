package schema

import (
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestConfigFieldReader_impl(t *testing.T) {
	var _ FieldReader = new(ConfigFieldReader)
}

func TestConfigFieldReader(t *testing.T) {
	testFieldReader(t, func(s map[string]*Schema) FieldReader {
		return &ConfigFieldReader{
			Schema: s,

			Config: testConfig(t, map[string]interface{}{
				"bool":   true,
				"float":  3.1415,
				"int":    42,
				"string": "string",

				"list": []interface{}{"foo", "bar"},

				"listInt": []interface{}{21, 42},

				"map": map[string]interface{}{
					"foo": "bar",
					"bar": "baz",
				},

				"set": []interface{}{10, 50},
				"setDeep": []interface{}{
					map[string]interface{}{
						"index": 10,
						"value": "foo",
					},
					map[string]interface{}{
						"index": 50,
						"value": "bar",
					},
				},
			}),
		}
	})
}

func testConfig(
	t *testing.T, raw map[string]interface{}) *terraform.ResourceConfig {
	rc, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return terraform.NewResourceConfig(rc)
}
