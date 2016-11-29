package schema

import (
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
	"reflect"
	"testing"
)

func TestProvisioner_init(t *testing.T) {
	var _ terraform.ResourceProvisioner = new(Provisioner)
}

func TestProvisioner_Validate(t *testing.T) {
	cases := []struct {
		P      *Provisioner
		Config map[string]interface{}
		Warns  []string
		Err    bool
	}{
		{
			// Incorrect schema
			P: &Provisioner{
				Schema: map[string]*Schema{
					"foo": {},
				},
			},
			Config: nil,
			Err:    true,
		},
		{
			P: &Provisioner{
				Schema: map[string]*Schema{
					"foo": {
						Type:     TypeString,
						Optional: true,
						ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
							ws = append(ws, "Simple warning from property validation")
							return
						},
					},
				},
			},
			Config: map[string]interface{}{
				"foo": "",
			},
			Err:   false,
			Warns: []string{"Simple warning from property validation"},
		},
		{
			P: &Provisioner{
				Schema: nil,
			},
			Config: nil,
			Err:    false,
		},
		{
			P: &Provisioner{
				Schema: nil,
				ValidateFunc: func(*terraform.ResourceConfig) (ws []string, errors []error) {
					ws = append(ws, "Simple warning from provisioner ValidateFunc")
					return
				},
			},
			Config: nil,
			Err:    false,
			Warns:  []string{"Simple warning from provisioner ValidateFunc"},
		},
	}

	for i, tc := range cases {
		c, err := config.NewRawConfig(tc.Config)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		ws, es := tc.P.Validate(terraform.NewResourceConfig(c))
		if len(es) > 0 != tc.Err {
			t.Fatalf("%d: %#v %s", i, es, es)
		}
		if (tc.Warns != nil || len(ws) != 0) && !reflect.DeepEqual(ws, tc.Warns) {
			t.Fatalf("%d: warnings mismatch, actual: %#v", i, ws)
		}
	}
}
