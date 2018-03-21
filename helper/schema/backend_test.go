package schema

import (
	"context"
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestBackendValidate(t *testing.T) {
	cases := []struct {
		Name   string
		B      *Backend
		Config map[string]cty.Value
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
			map[string]cty.Value{},
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
			map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			diags := tc.B.ValidateConfig(cty.ObjectVal(tc.Config))
			if diags.HasErrors() != tc.Err {
				t.Errorf("wrong number of diagnostics")
			}
		})
	}
}

func TestBackendConfigure(t *testing.T) {
	cases := []struct {
		Name   string
		B      *Backend
		Config map[string]cty.Value
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
			map[string]cty.Value{
				"foo": cty.NumberIntVal(42),
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			diags := tc.B.Configure(cty.ObjectVal(tc.Config))
			if diags.HasErrors() != tc.Err {
				t.Errorf("wrong number of diagnostics")
			}
		})
	}
}
