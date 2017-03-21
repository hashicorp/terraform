package schema

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestBackendValidate(t *testing.T) {
	cases := []struct {
		Name   string
		B      *Backend
		Config map[string]interface{}
		Err    bool
	}{
		{
			"Basic required field",
			&Backend{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Required: true,
						Type:     TypeString,
					},
				},
			},
			nil,
			true,
		},

		{
			"Basic required field set",
			&Backend{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Required: true,
						Type:     TypeString,
					},
				},
			},
			map[string]interface{}{
				"foo": "bar",
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			c, err := config.NewRawConfig(tc.Config)
			if err != nil {
				t.Fatalf("err: %s", err)
			}

			_, es := tc.B.Validate(terraform.NewResourceConfig(c))
			if len(es) > 0 != tc.Err {
				t.Fatalf("%d: %#v", i, es)
			}
		})
	}
}

func TestBackendConfigure(t *testing.T) {
	cases := []struct {
		Name   string
		B      *Backend
		Config map[string]interface{}
		Err    bool
	}{
		{
			"Basic config",
			&Backend{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},

				ConfigureFunc: func(ctx context.Context) error {
					d := FromContextBackendConfig(ctx)
					if d.Get("foo").(int) != 42 {
						return fmt.Errorf("bad config data")
					}

					return nil
				},
			},
			map[string]interface{}{
				"foo": 42,
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			c, err := config.NewRawConfig(tc.Config)
			if err != nil {
				t.Fatalf("err: %s", err)
			}

			err = tc.B.Configure(terraform.NewResourceConfig(c))
			if err != nil != tc.Err {
				t.Fatalf("%d: %s", i, err)
			}
		})
	}
}
