package schema

import (
	"context"
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestBackendPrepare(t *testing.T) {
	cases := []struct {
		Name   string
		B      *Backend
		Config map[string]cty.Value
		Expect map[string]cty.Value
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
			map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			},
			false,
		},

		{
			"unused default",
			&Backend{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Optional: true,
						Type:     TypeString,
						Default:  "baz",
					},
				},
			},
			map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			},
			map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			},
			false,
		},

		{
			"default",
			&Backend{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type:     TypeString,
						Optional: true,
						Default:  "baz",
					},
				},
			},
			map[string]cty.Value{},
			map[string]cty.Value{
				"foo": cty.StringVal("baz"),
			},
			false,
		},

		{
			"default func",
			&Backend{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type:     TypeString,
						Optional: true,
						DefaultFunc: func() (interface{}, error) {
							return "baz", nil
						},
					},
				},
			},
			map[string]cty.Value{},
			map[string]cty.Value{
				"foo": cty.StringVal("baz"),
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			configVal, diags := tc.B.PrepareConfig(cty.ObjectVal(tc.Config))
			if diags.HasErrors() != tc.Err {
				for _, d := range diags {
					t.Error(d.Description())
				}
			}

			if tc.Err {
				return
			}

			expect := cty.ObjectVal(tc.Expect)
			if !expect.RawEquals(configVal) {
				t.Fatalf("\nexpected: %#v\ngot:     %#v\n", expect, configVal)
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
