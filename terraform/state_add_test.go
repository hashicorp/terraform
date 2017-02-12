package terraform

import (
	"fmt"
	"testing"
)

func TestStateAdd(t *testing.T) {
	cases := []struct {
		Name     string
		Err      bool
		From, To string
		Value    interface{}
		One, Two *State
	}{
		{
			"ModuleState => Module Addr (new)",
			false,
			"",
			"module.foo",
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"test_instance.foo": &ResourceState{
						Type: "test_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"test_instance.bar": &ResourceState{
						Type: "test_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root", "foo"},
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},

							"test_instance.bar": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
		},

		{
			"ModuleState => Nested Module Addr (new)",
			false,
			"",
			"module.foo.module.bar",
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"test_instance.foo": &ResourceState{
						Type: "test_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"test_instance.bar": &ResourceState{
						Type: "test_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root", "foo", "bar"},
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},

							"test_instance.bar": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
		},

		{
			"ModuleState w/ outputs and deps => Module Addr (new)",
			false,
			"",
			"module.foo",
			&ModuleState{
				Path: rootModulePath,
				Outputs: map[string]*OutputState{
					"foo": &OutputState{
						Type:      "string",
						Sensitive: false,
						Value:     "bar",
					},
				},
				Dependencies: []string{"foo"},
				Resources: map[string]*ResourceState{
					"test_instance.foo": &ResourceState{
						Type: "test_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"test_instance.bar": &ResourceState{
						Type: "test_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root", "foo"},
						Outputs: map[string]*OutputState{
							"foo": &OutputState{
								Type:      "string",
								Sensitive: false,
								Value:     "bar",
							},
						},
						Dependencies: []string{"foo"},
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},

							"test_instance.bar": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
		},

		{
			"ModuleState => Module Addr (existing)",
			true,
			"",
			"module.foo",
			&ModuleState{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root", "foo"},
						Resources: map[string]*ResourceState{
							"test_instance.baz": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
			nil,
		},

		{
			"ModuleState with children => Module Addr (new)",
			false,
			"module.foo",
			"module.bar",

			[]*ModuleState{
				&ModuleState{
					Path:      []string{"root", "foo"},
					Resources: map[string]*ResourceState{},
				},

				&ModuleState{
					Path: []string{"root", "foo", "child1"},
					Resources: map[string]*ResourceState{
						"test_instance.foo": &ResourceState{
							Type: "test_instance",
							Primary: &InstanceState{
								ID: "foo",
							},
						},
					},
				},

				&ModuleState{
					Path: []string{"root", "foo", "child2"},
					Resources: map[string]*ResourceState{
						"test_instance.foo": &ResourceState{
							Type: "test_instance",
							Primary: &InstanceState{
								ID: "foo",
							},
						},
					},
				},

				// Should be ignored
				&ModuleState{
					Path: []string{"root", "baz", "child2"},
					Resources: map[string]*ResourceState{
						"test_instance.foo": &ResourceState{
							Type: "test_instance",
							Primary: &InstanceState{
								ID: "foo",
							},
						},
					},
				},
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path:      []string{"root", "bar"},
						Resources: map[string]*ResourceState{},
					},

					&ModuleState{
						Path: []string{"root", "bar", "child1"},
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},

					&ModuleState{
						Path: []string{"root", "bar", "child2"},
						Resources: map[string]*ResourceState{
							"test_instance.foo": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
		},

		{
			"ResourceState => Resource Addr (new)",
			false,
			"aws_instance.bar",
			"aws_instance.foo",
			&ResourceState{
				Type: "test_instance",
				Primary: &InstanceState{
					ID: "foo",
				},
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"aws_instance.foo": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
		},

		{
			"ResourceState w/ deps, provider => Resource Addr (new)",
			false,
			"aws_instance.bar",
			"aws_instance.foo",
			&ResourceState{
				Type:         "test_instance",
				Provider:     "foo",
				Dependencies: []string{"bar"},
				Primary: &InstanceState{
					ID: "foo",
				},
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"aws_instance.foo": &ResourceState{
								Type:         "test_instance",
								Provider:     "foo",
								Dependencies: []string{"bar"},
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
		},

		{
			"ResourceState tainted => Resource Addr (new)",
			false,
			"aws_instance.bar",
			"aws_instance.foo",
			&ResourceState{
				Type: "test_instance",
				Primary: &InstanceState{
					ID:      "foo",
					Tainted: true,
				},
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"aws_instance.foo": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID:      "foo",
									Tainted: true,
								},
							},
						},
					},
				},
			},
		},

		{
			"ResourceState with count unspecified => Resource Addr (new)",
			false,
			"aws_instance.bar",
			"aws_instance.foo",
			[]*ResourceState{
				&ResourceState{
					Type: "test_instance",
					Primary: &InstanceState{
						ID: "foo",
					},
				},

				&ResourceState{
					Type: "test_instance",
					Primary: &InstanceState{
						ID: "bar",
					},
				},
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"aws_instance.foo.0": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},

							"aws_instance.foo.1": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "bar",
								},
							},
						},
					},
				},
			},
		},

		{
			"ResourceState with count unspecified => Resource Addr (new with count)",
			true,
			"aws_instance.bar",
			"aws_instance.foo[0]",
			[]*ResourceState{
				&ResourceState{
					Type: "test_instance",
					Primary: &InstanceState{
						ID: "foo",
					},
				},

				&ResourceState{
					Type: "test_instance",
					Primary: &InstanceState{
						ID: "bar",
					},
				},
			},

			&State{},
			nil,
		},

		{
			"ResourceState with single count unspecified => Resource Addr (new with count)",
			false,
			"aws_instance.bar",
			"aws_instance.foo[0]",
			[]*ResourceState{
				&ResourceState{
					Type: "test_instance",
					Primary: &InstanceState{
						ID: "foo",
					},
				},
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"aws_instance.foo.0": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
		},

		{
			"ResourceState => Resource Addr (new with count)",
			false,
			"aws_instance.bar",
			"aws_instance.foo[0]",
			&ResourceState{
				Type: "test_instance",
				Primary: &InstanceState{
					ID: "foo",
				},
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"aws_instance.foo.0": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
		},

		{
			"ResourceState => Resource Addr (existing)",
			true,
			"aws_instance.bar",
			"aws_instance.foo",
			&ResourceState{
				Type: "test_instance",
				Primary: &InstanceState{
					ID: "foo",
				},
			},

			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"aws_instance.foo": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
			nil,
		},

		{
			"ResourceState => Module (new)",
			false,
			"aws_instance.bar",
			"module.foo",
			&ResourceState{
				Type: "test_instance",
				Primary: &InstanceState{
					ID: "foo",
				},
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root", "foo"},
						Resources: map[string]*ResourceState{
							"aws_instance.bar": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
		},

		{
			"InstanceState => Resource (new)",
			false,
			"aws_instance.bar.primary",
			"aws_instance.baz",
			&InstanceState{
				ID: "foo",
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root"},
						Resources: map[string]*ResourceState{
							"aws_instance.baz": &ResourceState{
								Type: "aws_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
		},

		{
			"InstanceState => Module (new)",
			false,
			"aws_instance.bar.primary",
			"module.foo",
			&InstanceState{
				ID: "foo",
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root", "foo"},
						Resources: map[string]*ResourceState{
							"aws_instance.bar": &ResourceState{
								Type: "aws_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
		},

		{
			"ModuleState => Module Addr (new with data source)",
			false,
			"",
			"module.foo",
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"data.test_instance.foo": &ResourceState{
						Type: "test_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
			},

			&State{},
			&State{
				Modules: []*ModuleState{
					&ModuleState{
						Path: []string{"root", "foo"},
						Resources: map[string]*ResourceState{
							"data.test_instance.foo": &ResourceState{
								Type: "test_instance",
								Primary: &InstanceState{
									ID: "foo",
								},
							},
						},
					},
				},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			// Make sure they're both initialized as normal
			tc.One.init()
			if tc.Two != nil {
				tc.Two.init()
			}

			// Add the value
			err := tc.One.Add(tc.From, tc.To, tc.Value)
			if (err != nil) != tc.Err {
				t.Fatal(err)
			}
			if tc.Err {
				return
			}

			// Prune them both to be sure
			tc.One.prune()
			tc.Two.prune()

			// Verify equality
			if !tc.One.Equal(tc.Two) {
				//t.Fatalf("Bad: %s\n\n%#v\n\n%#v", k, tc.One, tc.Two)
				t.Fatalf("Bad: \n\n%s\n\n%s", tc.One.String(), tc.Two.String())
			}
		})
	}
}
