package terraform

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/moduledeps"
	"github.com/hashicorp/terraform/plugin/discovery"
)

func TestModuleTreeDependencies(t *testing.T) {
	tests := map[string]struct {
		ConfigDir string // directory name from test-fixtures dir
		State     *State
		Want      *moduledeps.Module
	}{
		"no config or state": {
			"",
			nil,
			&moduledeps.Module{
				Name:      "root",
				Providers: moduledeps.Providers{},
				Children:  nil,
			},
		},
		"empty config no state": {
			"empty",
			nil,
			&moduledeps.Module{
				Name:      "root",
				Providers: moduledeps.Providers{},
				Children:  nil,
			},
		},
		"explicit provider": {
			"module-deps-explicit-provider",
			nil,
			&moduledeps.Module{
				Name: "root",
				Providers: moduledeps.Providers{
					"foo": moduledeps.ProviderDependency{
						Constraints: discovery.ConstraintStr(">=1.0.0").MustParse(),
						Reason:      moduledeps.ProviderDependencyExplicit,
					},
					"foo.bar": moduledeps.ProviderDependency{
						Constraints: discovery.ConstraintStr(">=2.0.0").MustParse(),
						Reason:      moduledeps.ProviderDependencyExplicit,
					},
				},
				Children: nil,
			},
		},
		"explicit provider unconstrained": {
			"module-deps-explicit-provider-unconstrained",
			nil,
			&moduledeps.Module{
				Name: "root",
				Providers: moduledeps.Providers{
					"foo": moduledeps.ProviderDependency{
						Constraints: discovery.AllVersions,
						Reason:      moduledeps.ProviderDependencyExplicit,
					},
				},
				Children: nil,
			},
		},
		"implicit provider": {
			"module-deps-implicit-provider",
			nil,
			&moduledeps.Module{
				Name: "root",
				Providers: moduledeps.Providers{
					"foo": moduledeps.ProviderDependency{
						Constraints: discovery.AllVersions,
						Reason:      moduledeps.ProviderDependencyImplicit,
					},
					"foo.baz": moduledeps.ProviderDependency{
						Constraints: discovery.AllVersions,
						Reason:      moduledeps.ProviderDependencyImplicit,
					},
				},
				Children: nil,
			},
		},
		"explicit provider with resource": {
			"module-deps-explicit-provider-resource",
			nil,
			&moduledeps.Module{
				Name: "root",
				Providers: moduledeps.Providers{
					"foo": moduledeps.ProviderDependency{
						Constraints: discovery.ConstraintStr(">=1.0.0").MustParse(),
						Reason:      moduledeps.ProviderDependencyExplicit,
					},
				},
				Children: nil,
			},
		},
		"inheritance of providers": {
			"module-deps-inherit-provider",
			nil,
			&moduledeps.Module{
				Name: "root",
				Providers: moduledeps.Providers{
					"foo": moduledeps.ProviderDependency{
						Constraints: discovery.AllVersions,
						Reason:      moduledeps.ProviderDependencyExplicit,
					},
					"bar": moduledeps.ProviderDependency{
						Constraints: discovery.AllVersions,
						Reason:      moduledeps.ProviderDependencyExplicit,
					},
				},
				Children: []*moduledeps.Module{
					{
						Name: "child",
						Providers: moduledeps.Providers{
							"foo": moduledeps.ProviderDependency{
								Constraints: discovery.AllVersions,
								Reason:      moduledeps.ProviderDependencyInherited,
							},
							"baz": moduledeps.ProviderDependency{
								Constraints: discovery.AllVersions,
								Reason:      moduledeps.ProviderDependencyImplicit,
							},
						},
						Children: []*moduledeps.Module{
							{
								Name: "grandchild",
								Providers: moduledeps.Providers{
									"bar": moduledeps.ProviderDependency{
										Constraints: discovery.AllVersions,
										Reason:      moduledeps.ProviderDependencyInherited,
									},
									"foo": moduledeps.ProviderDependency{
										Constraints: discovery.AllVersions,
										Reason:      moduledeps.ProviderDependencyExplicit,
									},
								},
							},
						},
					},
				},
			},
		},
		"provider from state": {
			"empty",
			&State{
				Modules: []*ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"foo_bar.baz": {
								Type:     "foo_bar",
								Provider: "",
							},
						},
					},
				},
			},
			&moduledeps.Module{
				Name: "root",
				Providers: moduledeps.Providers{
					"foo": moduledeps.ProviderDependency{
						Constraints: discovery.AllVersions,
						Reason:      moduledeps.ProviderDependencyFromState,
					},
				},
				Children: nil,
			},
		},
		"providers in both config and state": {
			"module-deps-explicit-provider",
			&State{
				Modules: []*ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"foo_bar.test1": {
								Type:     "foo_bar",
								Provider: "",
							},
							"foo_bar.test2": {
								Type:     "foo_bar",
								Provider: "foo.bar",
							},
							"baz_bar.test": {
								Type:     "baz_bar",
								Provider: "",
							},
						},
					},
					// note that we've skipped root.child intentionally here,
					// to verify that we'll infer it based on the following
					// module rather than crashing.
					{
						Path: []string{"root", "child", "grandchild"},
						Resources: map[string]*ResourceState{
							"banana_skin.test": {
								Type:     "banana_skin",
								Provider: "",
							},
						},
					},
				},
			},
			&moduledeps.Module{
				Name: "root",
				Providers: moduledeps.Providers{
					"foo": moduledeps.ProviderDependency{
						Constraints: discovery.ConstraintStr(">=1.0.0").MustParse(),
						Reason:      moduledeps.ProviderDependencyExplicit,
					},
					"foo.bar": moduledeps.ProviderDependency{
						Constraints: discovery.ConstraintStr(">=2.0.0").MustParse(),
						Reason:      moduledeps.ProviderDependencyExplicit,
					},
					"baz": moduledeps.ProviderDependency{
						Constraints: discovery.AllVersions,
						Reason:      moduledeps.ProviderDependencyFromState,
					},
				},
				Children: []*moduledeps.Module{
					{
						Name: "child",
						Children: []*moduledeps.Module{
							{
								Name: "grandchild",
								Providers: moduledeps.Providers{
									"banana": moduledeps.ProviderDependency{
										Constraints: discovery.AllVersions,
										Reason:      moduledeps.ProviderDependencyFromState,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var root *module.Tree
			if test.ConfigDir != "" {
				root = testModule(t, test.ConfigDir)
			}

			got := ModuleTreeDependencies(root, test.State)
			if !got.Equal(test.Want) {
				t.Errorf(
					"wrong dependency tree\ngot:  %s\nwant: %s",
					spew.Sdump(got),
					spew.Sdump(test.Want),
				)
			}
		})
	}
}
