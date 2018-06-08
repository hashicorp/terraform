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
					&Module{Name: "foo"},
				},
				Outputs: []*Output{
					&Output{Name: "foo"},
				},
				ProviderConfigs: []*ProviderConfig{
					&ProviderConfig{Name: "foo"},
				},
				Resources: []*Resource{
					&Resource{Name: "foo"},
				},
				Variables: []*Variable{
					&Variable{Name: "foo"},
				},
				Locals: []*Local{
					&Local{Name: "foo"},
				},

				unknownKeys: []string{"foo"},
			},

			&Config{
				Atlas: &AtlasConfig{
					Name: "bar",
				},
				Modules: []*Module{
					&Module{Name: "bar"},
				},
				Outputs: []*Output{
					&Output{Name: "bar"},
				},
				ProviderConfigs: []*ProviderConfig{
					&ProviderConfig{Name: "bar"},
				},
				Resources: []*Resource{
					&Resource{Name: "bar"},
				},
				Variables: []*Variable{
					&Variable{Name: "bar"},
				},
				Locals: []*Local{
					&Local{Name: "bar"},
				},

				unknownKeys: []string{"bar"},
			},

			&Config{
				Atlas: &AtlasConfig{
					Name: "bar",
				},
				Modules: []*Module{
					&Module{Name: "foo"},
					&Module{Name: "bar"},
				},
				Outputs: []*Output{
					&Output{Name: "foo"},
					&Output{Name: "bar"},
				},
				ProviderConfigs: []*ProviderConfig{
					&ProviderConfig{Name: "foo"},
					&ProviderConfig{Name: "bar"},
				},
				Resources: []*Resource{
					&Resource{Name: "foo"},
					&Resource{Name: "bar"},
				},
				Variables: []*Variable{
					&Variable{Name: "foo"},
					&Variable{Name: "bar"},
				},
				Locals: []*Local{
					&Local{Name: "foo"},
					&Local{Name: "bar"},
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
					&Output{Name: "foo"},
				},
				ProviderConfigs: []*ProviderConfig{
					&ProviderConfig{Name: "foo"},
				},
				Resources: []*Resource{
					&Resource{Name: "foo"},
				},
				Variables: []*Variable{
					&Variable{Name: "foo", Default: "foo"},
					&Variable{Name: "foo"},
				},
				Locals: []*Local{
					&Local{Name: "foo"},
				},

				unknownKeys: []string{"foo"},
			},

			&Config{
				Outputs: []*Output{
					&Output{Name: "bar"},
				},
				ProviderConfigs: []*ProviderConfig{
					&ProviderConfig{Name: "bar"},
				},
				Resources: []*Resource{
					&Resource{Name: "bar"},
				},
				Variables: []*Variable{
					&Variable{Name: "foo", Default: "bar"},
					&Variable{Name: "bar"},
				},
				Locals: []*Local{
					&Local{Name: "foo"},
				},

				unknownKeys: []string{"bar"},
			},

			&Config{
				Outputs: []*Output{
					&Output{Name: "foo"},
					&Output{Name: "bar"},
				},
				ProviderConfigs: []*ProviderConfig{
					&ProviderConfig{Name: "foo"},
					&ProviderConfig{Name: "bar"},
				},
				Resources: []*Resource{
					&Resource{Name: "foo"},
					&Resource{Name: "bar"},
				},
				Variables: []*Variable{
					&Variable{Name: "foo", Default: "bar"},
					&Variable{Name: "foo"},
					&Variable{Name: "bar"},
				},
				Locals: []*Local{
					&Local{Name: "foo"},
					&Local{Name: "foo"},
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
					&ProviderConfig{Alias: "foo"},
				},
			},
			&Config{},
			&Config{
				ProviderConfigs: []*ProviderConfig{
					&ProviderConfig{Alias: "foo"},
				},
			},
			false,
		},

		{
			&Config{},
			&Config{
				ProviderConfigs: []*ProviderConfig{
					&ProviderConfig{Alias: "foo"},
				},
			},
			&Config{
				ProviderConfigs: []*ProviderConfig{
					&ProviderConfig{Alias: "foo"},
				},
			},
			false,
		},

		{
			&Config{
				ProviderConfigs: []*ProviderConfig{
					&ProviderConfig{Alias: "bar"},
				},
			},
			&Config{
				ProviderConfigs: []*ProviderConfig{
					&ProviderConfig{Alias: "foo"},
				},
			},
			&Config{
				ProviderConfigs: []*ProviderConfig{
					&ProviderConfig{Alias: "foo"},
				},
			},
			false,
		},

		// Variable type
		{
			&Config{
				Variables: []*Variable{
					&Variable{DeclaredType: "foo"},
				},
			},
			&Config{},
			&Config{
				Variables: []*Variable{
					&Variable{DeclaredType: "foo"},
				},
			},
			false,
		},

		{
			&Config{},
			&Config{
				Variables: []*Variable{
					&Variable{DeclaredType: "foo"},
				},
			},
			&Config{
				Variables: []*Variable{
					&Variable{DeclaredType: "foo"},
				},
			},
			false,
		},

		{
			&Config{
				Variables: []*Variable{
					&Variable{DeclaredType: "bar"},
				},
			},
			&Config{
				Variables: []*Variable{
					&Variable{DeclaredType: "foo"},
				},
			},
			&Config{
				Variables: []*Variable{
					&Variable{DeclaredType: "foo"},
				},
			},
			false,
		},

		// Output description
		{
			&Config{
				Outputs: []*Output{
					&Output{Description: "foo"},
				},
			},
			&Config{},
			&Config{
				Outputs: []*Output{
					&Output{Description: "foo"},
				},
			},
			false,
		},

		{
			&Config{},
			&Config{
				Outputs: []*Output{
					&Output{Description: "foo"},
				},
			},
			&Config{
				Outputs: []*Output{
					&Output{Description: "foo"},
				},
			},
			false,
		},

		{
			&Config{
				Outputs: []*Output{
					&Output{Description: "bar"},
				},
			},
			&Config{
				Outputs: []*Output{
					&Output{Description: "foo"},
				},
			},
			&Config{
				Outputs: []*Output{
					&Output{Description: "foo"},
				},
			},
			false,
		},

		// Output depends_on
		{
			&Config{
				Outputs: []*Output{
					&Output{DependsOn: []string{"foo"}},
				},
			},
			&Config{},
			&Config{
				Outputs: []*Output{
					&Output{DependsOn: []string{"foo"}},
				},
			},
			false,
		},

		{
			&Config{},
			&Config{
				Outputs: []*Output{
					&Output{DependsOn: []string{"foo"}},
				},
			},
			&Config{
				Outputs: []*Output{
					&Output{DependsOn: []string{"foo"}},
				},
			},
			false,
		},

		{
			&Config{
				Outputs: []*Output{
					&Output{DependsOn: []string{"bar"}},
				},
			},
			&Config{
				Outputs: []*Output{
					&Output{DependsOn: []string{"foo"}},
				},
			},
			&Config{
				Outputs: []*Output{
					&Output{DependsOn: []string{"foo"}},
				},
			},
			false,
		},

		// Output sensitive
		{
			&Config{
				Outputs: []*Output{
					&Output{Sensitive: true},
				},
			},
			&Config{},
			&Config{
				Outputs: []*Output{
					&Output{Sensitive: true},
				},
			},
			false,
		},

		{
			&Config{},
			&Config{
				Outputs: []*Output{
					&Output{Sensitive: true},
				},
			},
			&Config{
				Outputs: []*Output{
					&Output{Sensitive: true},
				},
			},
			false,
		},

		{
			&Config{
				Outputs: []*Output{
					&Output{Sensitive: false},
				},
			},
			&Config{
				Outputs: []*Output{
					&Output{Sensitive: true},
				},
			},
			&Config{
				Outputs: []*Output{
					&Output{Sensitive: true},
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
