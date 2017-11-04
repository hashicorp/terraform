package config

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestAppend(t *testing.T) {
	cases := []struct {
		c1, c2, result *Config
		err            bool
	}{
		{
			&Config{
				Atlas: &AtlasConfig{
					Name: "foo",
				},
				Modules: []*Module{
					{Name: "foo"},
				},
				Outputs: []*Output{
					{Name: "foo"},
				},
				ProviderConfigs: []*ProviderConfig{
					{Name: "foo"},
				},
				Resources: []*Resource{
					{Name: "foo"},
				},
				Variables: []*Variable{
					{Name: "foo"},
				},
				Locals: []*Local{
					{Name: "foo"},
				},

				unknownKeys: []string{"foo"},
			},

			&Config{
				Atlas: &AtlasConfig{
					Name: "bar",
				},
				Modules: []*Module{
					{Name: "bar"},
				},
				Outputs: []*Output{
					{Name: "bar"},
				},
				ProviderConfigs: []*ProviderConfig{
					{Name: "bar"},
				},
				Resources: []*Resource{
					{Name: "bar"},
				},
				Variables: []*Variable{
					{Name: "bar"},
				},
				Locals: []*Local{
					{Name: "bar"},
				},

				unknownKeys: []string{"bar"},
			},

			&Config{
				Atlas: &AtlasConfig{
					Name: "bar",
				},
				Modules: []*Module{
					{Name: "foo"},
					{Name: "bar"},
				},
				Outputs: []*Output{
					{Name: "foo"},
					{Name: "bar"},
				},
				ProviderConfigs: []*ProviderConfig{
					{Name: "foo"},
					{Name: "bar"},
				},
				Resources: []*Resource{
					{Name: "foo"},
					{Name: "bar"},
				},
				Variables: []*Variable{
					{Name: "foo"},
					{Name: "bar"},
				},
				Locals: []*Local{
					{Name: "foo"},
					{Name: "bar"},
				},

				unknownKeys: []string{"foo", "bar"},
			},

			false,
		},

		// Terraform block
		{
			&Config{
				Terraform: &Terraform{
					RequiredVersion: "A",
				},
			},
			&Config{},
			&Config{
				Terraform: &Terraform{
					RequiredVersion: "A",
				},
			},
			false,
		},

		{
			&Config{},
			&Config{
				Terraform: &Terraform{
					RequiredVersion: "A",
				},
			},
			&Config{
				Terraform: &Terraform{
					RequiredVersion: "A",
				},
			},
			false,
		},

		// appending configs merges terraform blocks
		{
			&Config{
				Terraform: &Terraform{
					RequiredVersion: "A",
				},
			},
			&Config{
				Terraform: &Terraform{
					Backend: &Backend{
						Type: "test",
					},
				},
			},
			&Config{
				Terraform: &Terraform{
					RequiredVersion: "A",
					Backend: &Backend{
						Type: "test",
					},
				},
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			actual, err := Append(tc.c1, tc.c2)
			if err != nil != tc.err {
				t.Errorf("unexpected error: %s", err)
			}

			if !reflect.DeepEqual(actual, tc.result) {
				t.Errorf("wrong result\ngot: %swant: %s", spew.Sdump(actual), spew.Sdump(tc.result))
			}
		})
	}
}
