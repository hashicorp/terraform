package schema

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestProviderConfigure(t *testing.T) {
	cases := []struct {
		P      *Provider
		Config map[string]interface{}
		Err    bool
	}{
		{
			P:      &Provider{},
			Config: nil,
			Err:    false,
		},

		{
			P: &Provider{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},

				ConfigureFunc: func(d *ResourceData) error {
					if d.Get("foo").(int) == 42 {
						return nil
					}

					return fmt.Errorf("nope")
				},
			},
			Config: map[string]interface{}{
				"foo": 42,
			},
			Err: false,
		},

		{
			P: &Provider{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},

				ConfigureFunc: func(d *ResourceData) error {
					if d.Get("foo").(int) == 42 {
						return nil
					}

					return fmt.Errorf("nope")
				},
			},
			Config: map[string]interface{}{
				"foo": 52,
			},
			Err: true,
		},
	}

	for i, tc := range cases {
		c, err := config.NewRawConfig(tc.Config)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		err = tc.P.Configure(terraform.NewResourceConfig(c))
		if (err != nil) != tc.Err {
			t.Fatalf("%d: %s", i, err)
		}
	}
}

func TestProviderValidateResource(t *testing.T) {
	cases := []struct {
		P      *Provider
		Type   string
		Config map[string]interface{}
		Err    bool
	}{
		{
			P:      &Provider{},
			Type:   "foo",
			Config: nil,
			Err:    true,
		},

		{
			P: &Provider{
				Resources: map[string]*Resource{
					"foo": &Resource{},
				},
			},
			Type:   "foo",
			Config: nil,
			Err:    false,
		},
	}

	for i, tc := range cases {
		c, err := config.NewRawConfig(tc.Config)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		_, es := tc.P.ValidateResource(tc.Type, terraform.NewResourceConfig(c))
		if (len(es) > 0) != tc.Err {
			t.Fatalf("%d: %#v", i, es)
		}
	}
}
