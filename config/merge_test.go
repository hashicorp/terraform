package config

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestMerge(t *testing.T) {
	cases := []struct {
		c1, c2, result *Config
		err            bool
	}{
		// Normal good case.
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

		// Test that when merging duplicates, it merges into the
		// first, but keeps the duplicates so that errors still
		// happen.
		{
			&Config{
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
					{Name: "foo", Default: "foo"},
					{Name: "foo"},
				},
				Locals: []*Local{
					{Name: "foo"},
				},

				unknownKeys: []string{"foo"},
			},

			&Config{
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
					{Name: "foo", Default: "bar"},
					{Name: "bar"},
				},
				Locals: []*Local{
					{Name: "foo"},
				},

				unknownKeys: []string{"bar"},
			},

			&Config{
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
					{Name: "foo", Default: "bar"},
					{Name: "foo"},
					{Name: "bar"},
				},
				Locals: []*Local{
					{Name: "foo"},
					{Name: "foo"},
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

		// Provider alias
		{
			&Config{
				ProviderConfigs: []*ProviderConfig{
					{Alias: "foo"},
				},
			},
			&Config{},
			&Config{
				ProviderConfigs: []*ProviderConfig{
					{Alias: "foo"},
				},
			},
			false,
		},

		{
			&Config{},
			&Config{
				ProviderConfigs: []*ProviderConfig{
					{Alias: "foo"},
				},
			},
			&Config{
				ProviderConfigs: []*ProviderConfig{
					{Alias: "foo"},
				},
			},
			false,
		},

		{
			&Config{
				ProviderConfigs: []*ProviderConfig{
					{Alias: "bar"},
				},
			},
			&Config{
				ProviderConfigs: []*ProviderConfig{
					{Alias: "foo"},
				},
			},
			&Config{
				ProviderConfigs: []*ProviderConfig{
					{Alias: "foo"},
				},
			},
			false,
		},

		// Variable type
		{
			&Config{
				Variables: []*Variable{
					{DeclaredType: "foo"},
				},
			},
			&Config{},
			&Config{
				Variables: []*Variable{
					{DeclaredType: "foo"},
				},
			},
			false,
		},

		{
			&Config{},
			&Config{
				Variables: []*Variable{
					{DeclaredType: "foo"},
				},
			},
			&Config{
				Variables: []*Variable{
					{DeclaredType: "foo"},
				},
			},
			false,
		},

		{
			&Config{
				Variables: []*Variable{
					{DeclaredType: "bar"},
				},
			},
			&Config{
				Variables: []*Variable{
					{DeclaredType: "foo"},
				},
			},
			&Config{
				Variables: []*Variable{
					{DeclaredType: "foo"},
				},
			},
			false,
		},

		// Output description
		{
			&Config{
				Outputs: []*Output{
					{Description: "foo"},
				},
			},
			&Config{},
			&Config{
				Outputs: []*Output{
					{Description: "foo"},
				},
			},
			false,
		},

		{
			&Config{},
			&Config{
				Outputs: []*Output{
					{Description: "foo"},
				},
			},
			&Config{
				Outputs: []*Output{
					{Description: "foo"},
				},
			},
			false,
		},

		{
			&Config{
				Outputs: []*Output{
					{Description: "bar"},
				},
			},
			&Config{
				Outputs: []*Output{
					{Description: "foo"},
				},
			},
			&Config{
				Outputs: []*Output{
					{Description: "foo"},
				},
			},
			false,
		},

		// Output depends_on
		{
			&Config{
				Outputs: []*Output{
					{DependsOn: []string{"foo"}},
				},
			},
			&Config{},
			&Config{
				Outputs: []*Output{
					{DependsOn: []string{"foo"}},
				},
			},
			false,
		},

		{
			&Config{},
			&Config{
				Outputs: []*Output{
					{DependsOn: []string{"foo"}},
				},
			},
			&Config{
				Outputs: []*Output{
					{DependsOn: []string{"foo"}},
				},
			},
			false,
		},

		{
			&Config{
				Outputs: []*Output{
					{DependsOn: []string{"bar"}},
				},
			},
			&Config{
				Outputs: []*Output{
					{DependsOn: []string{"foo"}},
				},
			},
			&Config{
				Outputs: []*Output{
					{DependsOn: []string{"foo"}},
				},
			},
			false,
		},

		// Output sensitive
		{
			&Config{
				Outputs: []*Output{
					{Sensitive: true},
				},
			},
			&Config{},
			&Config{
				Outputs: []*Output{
					{Sensitive: true},
				},
			},
			false,
		},

		{
			&Config{},
			&Config{
				Outputs: []*Output{
					{Sensitive: true},
				},
			},
			&Config{
				Outputs: []*Output{
					{Sensitive: true},
				},
			},
			false,
		},

		{
			&Config{
				Outputs: []*Output{
					{Sensitive: false},
				},
			},
			&Config{
				Outputs: []*Output{
					{Sensitive: true},
				},
			},
			&Config{
				Outputs: []*Output{
					{Sensitive: true},
				},
			},
			false,
		},

		// terraform blocks are merged, not overwritten
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
			actual, err := Merge(tc.c1, tc.c2)
			if err != nil != tc.err {
				t.Errorf("unexpected error: %s", err)
			}

			if !reflect.DeepEqual(actual, tc.result) {
				t.Errorf("wrong result\ngot: %swant: %s", spew.Sdump(actual), spew.Sdump(tc.result))
			}
		})
	}
}
