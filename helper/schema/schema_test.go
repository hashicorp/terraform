package schema

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestSchemaMap_Diff(t *testing.T) {
	cases := []struct {
		Schema map[string]*Schema
		State  *terraform.ResourceState
		Config map[string]interface{}
		Diff   *terraform.ResourceDiff
		Err    bool
	}{
		/*
		 * String decode
		 */

		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},

			Diff: &terraform.ResourceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			Diff: &terraform.ResourceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "",
						NewComputed: true,
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Required: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			Diff: nil,

			Err: true,
		},

		/*
		 * Int decode
		 */

		{
			Schema: map[string]*Schema{
				"port": &Schema{
					Type:     TypeInt,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"port": 27,
			},

			Diff: &terraform.ResourceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"port": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "27",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		/*
		 * Bool decode
		 */

		{
			Schema: map[string]*Schema{
				"port": &Schema{
					Type:     TypeBool,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"port": false,
			},

			Diff: &terraform.ResourceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"port": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "0",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		/*
		 * List decode
		 */

		{
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, 2, 5},
			},

			Diff: &terraform.ResourceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "",
						New: "3",
					},
					"ports.0": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "",
						New: "2",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: &terraform.ResourceState{
				Attributes: map[string]string{
					"ports.#": "3",
					"ports.0": "1",
					"ports.1": "2",
					"ports.2": "5",
				},
			},

			Config: map[string]interface{}{
				"ports": []interface{}{1, 2, 5},
			},

			Diff: nil,

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: &terraform.ResourceState{
				Attributes: map[string]string{
					"ports.#": "2",
					"ports.0": "1",
					"ports.1": "2",
				},
			},

			Config: map[string]interface{}{
				"ports": []interface{}{1, 2, 5},
			},

			Diff: &terraform.ResourceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "3",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, 2, 5},
			},

			Diff: &terraform.ResourceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "3",
						RequiresNew: true,
					},
					"ports.0": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "1",
						RequiresNew: true,
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "2",
						RequiresNew: true,
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "5",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		/*
		 * List of structure decode
		 */

		{
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:     TypeList,
					Required: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"from": &Schema{
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ingress": []interface{}{
					map[string]interface{}{
						"from": 8080,
					},
				},
			},

			Diff: &terraform.ResourceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ingress.#": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"ingress.0.from": &terraform.ResourceAttrDiff{
						Old: "",
						New: "8080",
					},
				},
			},

			Err: false,
		},
	}

	for i, tc := range cases {
		c, err := config.NewRawConfig(tc.Config)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		d, err := schemaMap(tc.Schema).Diff(
			tc.State, terraform.NewResourceConfig(c))
		if (err != nil) != tc.Err {
			t.Fatalf("err: %s", err)
		}

		if !reflect.DeepEqual(tc.Diff, d) {
			t.Fatalf("#%d: bad:\n\n%#v", i, d)
		}
	}
}
