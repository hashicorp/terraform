package config

import (
	"reflect"
	"testing"
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

				unknownKeys: []string{"foo", "bar"},
			},

			false,
		},
	}

	for i, tc := range cases {
		actual, err := Merge(tc.c1, tc.c2)
		if (err != nil) != tc.err {
			t.Fatalf("%d: error fail", i)
		}

		if !reflect.DeepEqual(actual, tc.result) {
			t.Fatalf("%d: bad:\n\n%#v", i, actual)
		}
	}
}
