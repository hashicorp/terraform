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

			Config: map[string]interface{}{},

			Diff: nil,

			Err: true,
		},

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
				"ingress": []interface{}{},
			},

			Diff: nil,

			Err: true,
		},

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
					map[string]interface{}{},
				},
			},

			Diff: nil,

			Err: true,
		},

		/*
		 * ComputedWhen
		 */

		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:         TypeString,
					Computed:     true,
					ComputedWhen: []string{"port"},
				},

				"port": &Schema{
					Type:     TypeInt,
					Optional: true,
				},
			},

			State: &terraform.ResourceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
					"port":              "80",
				},
			},

			Config: map[string]interface{}{
				"port": 80,
			},

			Diff: nil,

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:         TypeString,
					Computed:     true,
					ComputedWhen: []string{"port"},
				},

				"port": &Schema{
					Type:     TypeInt,
					Optional: true,
				},
			},

			State: &terraform.ResourceState{
				Attributes: map[string]string{
					"port": "80",
				},
			},

			Config: map[string]interface{}{
				"port": 80,
			},

			Diff: &terraform.ResourceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		/* TODO
		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:         TypeString,
					Computed:     true,
					ComputedWhen: []string{"port"},
				},

				"port": &Schema{
					Type:     TypeInt,
					Optional: true,
				},
			},

			State: &terraform.ResourceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
					"port":              "80",
				},
			},

			Config: map[string]interface{}{
				"port": 8080,
			},

			Diff: &terraform.ResourceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "foo",
						NewComputed: true,
					},
					"port": &terraform.ResourceAttrDiff{
						Old: "80",
						New: "8080",
					},
				},
			},

			Err: false,
		},
		*/
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

func TestSchemaMap_Validate(t *testing.T) {
	cases := []struct {
		Schema map[string]*Schema
		Config map[string]interface{}
		Warn   bool
		Err    bool
	}{
		// Good
		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},
		},

		// Required field not set
		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Required: true,
				},
			},

			Config: map[string]interface{}{},

			Err: true,
		},

		// Invalid type
		{
			Schema: map[string]*Schema{
				"port": &Schema{
					Type:     TypeInt,
					Required: true,
				},
			},

			Config: map[string]interface{}{
				"port": "I am invalid",
			},

			Err: true,
		},

		// Optional sub-resource
		{
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type: TypeList,
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

			Config: map[string]interface{}{},

			Err: false,
		},

		// Not a list
		{
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type: TypeList,
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

			Config: map[string]interface{}{
				"ingress": "foo",
			},

			Err: true,
		},

		// Required sub-resource field
		{
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type: TypeList,
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

			Config: map[string]interface{}{
				"ingress": []interface{}{
					map[string]interface{}{},
				},
			},

			Err: true,
		},

		// Good sub-resource
		{
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:     TypeList,
					Optional: true,
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

			Config: map[string]interface{}{
				"ingress": []interface{}{
					map[string]interface{}{
						"from": 80,
					},
				},
			},

			Err: false,
		},

		// Invalid/unknown field
		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			Config: map[string]interface{}{
				"foo": "bar",
			},

			Err: true,
		},

		// Computed field set
		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Computed: true,
				},
			},

			Config: map[string]interface{}{
				"availability_zone": "bar",
			},

			Err: true,
		},
	}

	for i, tc := range cases {
		c, err := config.NewRawConfig(tc.Config)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		ws, es := schemaMap(tc.Schema).Validate(terraform.NewResourceConfig(c))
		if (len(es) > 0) != tc.Err {
			if len(es) == 0 {
				t.Errorf("%d: no errors", i)
			}

			for _, e := range es {
				t.Errorf("%d: err: %s", i, e)
			}

			t.FailNow()
		}

		if (len(ws) > 0) != tc.Warn {
			t.Fatalf("%d: ws: %#v", i, ws)
		}
	}
}
