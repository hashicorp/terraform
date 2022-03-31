package views

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/zclconf/go-cty/cty"
)

// Ensure that the correct view type and in-automation settings propagate to the
// Operation view.
func TestPlanHuman_operation(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	defer done(t)
	v := NewPlan(arguments.ViewHuman, NewView(streams).SetRunningInAutomation(true)).Operation()
	if hv, ok := v.(*OperationHuman); !ok {
		t.Fatalf("unexpected return type %t", v)
	} else if hv.inAutomation != true {
		t.Fatalf("unexpected inAutomation value on Operation view")
	}
}

// Verify that Hooks includes a UI hook
func TestPlanHuman_hooks(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	defer done(t)
	v := NewPlan(arguments.ViewHuman, NewView(streams).SetRunningInAutomation((true)))
	hooks := v.Hooks()

	var uiHook *UiHook
	for _, hook := range hooks {
		if ch, ok := hook.(*UiHook); ok {
			uiHook = ch
		}
	}
	if uiHook == nil {
		t.Fatalf("expected Hooks to include a UiHook: %#v", hooks)
	}
}

// Helper functions to build a trivial test plan, to exercise the plan
// renderer.
func testPlan(t *testing.T) *plans.Plan {
	t.Helper()

	plannedVal := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.UnknownVal(cty.String),
		"foo": cty.StringVal("bar"),
	})
	priorValRaw, err := plans.NewDynamicValue(cty.NullVal(plannedVal.Type()), plannedVal.Type())
	if err != nil {
		t.Fatal(err)
	}
	plannedValRaw, err := plans.NewDynamicValue(plannedVal, plannedVal.Type())
	if err != nil {
		t.Fatal(err)
	}

	changes := plans.NewChanges()
	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_resource",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	changes.SyncWrapper().AppendResourceInstanceChange(&plans.ResourceInstanceChangeSrc{
		Addr:        addr,
		PrevRunAddr: addr,
		ProviderAddr: addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
		ChangeSrc: plans.ChangeSrc{
			Action: plans.Create,
			Before: priorValRaw,
			After:  plannedValRaw,
		},
	})

	return &plans.Plan{
		Changes: changes,
	}
}

func testSchemas() *terraform.Schemas {
	provider := testProvider()
	return &terraform.Schemas{
		Providers: map[addrs.Provider]*terraform.ProviderSchema{
			addrs.NewDefaultProvider("test"): provider.ProviderSchema(),
		},
	}
}

func testProvider() *terraform.MockProvider {
	p := new(terraform.MockProvider)
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		return providers.ReadResourceResponse{NewState: req.PriorState}
	}

	p.GetProviderSchemaResponse = testProviderSchema()

	return p
}

func testProviderSchema() *providers.GetProviderSchemaResponse {
	return &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_resource": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Computed: true},
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
}

func TestFilterRefreshChange(t *testing.T) {
	tests := map[string]struct {
		paths                   []cty.Path
		before, after, expected cty.Value
	}{
		"attr was null": {
			// nested attr was null
			paths: []cty.Path{
				cty.GetAttrPath("attr").GetAttr("attr_null_before").GetAttr("b"),
			},
			before: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"attr_null_before": cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("old"),
						"b": cty.NullVal(cty.String),
					}),
				}),
			}),
			after: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"attr_null_before": cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("new"),
						"b": cty.StringVal("new"),
					}),
				}),
			}),
			expected: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"attr_null_before": cty.ObjectVal(map[string]cty.Value{
						// we old picked the change in b
						"a": cty.StringVal("old"),
						"b": cty.StringVal("new"),
					}),
				}),
			}),
		},
		"object was null": {
			// nested object attrs were null
			paths: []cty.Path{
				cty.GetAttrPath("attr").GetAttr("obj_null_before").GetAttr("b"),
			},
			before: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"obj_null_before": cty.NullVal(cty.Object(map[string]cty.Type{
						"a": cty.String,
						"b": cty.String,
					})),
					"other": cty.ObjectVal(map[string]cty.Value{
						"o": cty.StringVal("old"),
					}),
				}),
			}),
			after: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"obj_null_before": cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("new"),
						"b": cty.StringVal("new"),
					}),
					"other": cty.ObjectVal(map[string]cty.Value{
						"o": cty.StringVal("new"),
					}),
				}),
			}),
			expected: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"obj_null_before": cty.ObjectVal(map[string]cty.Value{
						// optimally "a" would be null, but we need to take the
						// entire object since it was null before.
						"a": cty.StringVal("new"),
						"b": cty.StringVal("new"),
					}),
					"other": cty.ObjectVal(map[string]cty.Value{
						"o": cty.StringVal("old"),
					}),
				}),
			}),
		},
		"object becomes null": {
			// nested object attr becoming null
			paths: []cty.Path{
				cty.GetAttrPath("attr").GetAttr("obj_null_after").GetAttr("a"),
			},
			before: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"obj_null_after": cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("old"),
						"b": cty.StringVal("old"),
					}),
					"other": cty.ObjectVal(map[string]cty.Value{
						"o": cty.StringVal("old"),
					}),
				}),
			}),
			after: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"obj_null_after": cty.NullVal(cty.Object(map[string]cty.Type{
						"a": cty.String,
						"b": cty.String,
					})),
					"other": cty.ObjectVal(map[string]cty.Value{
						"o": cty.StringVal("new"),
					}),
				}),
			}),
			expected: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"obj_null_after": cty.ObjectVal(map[string]cty.Value{
						"a": cty.NullVal(cty.String),
						"b": cty.StringVal("old"),
					}),
					"other": cty.ObjectVal(map[string]cty.Value{
						"o": cty.StringVal("old"),
					}),
				}),
			}),
		},
		"dynamic adding values": {
			// dynamic gaining values
			paths: []cty.Path{
				cty.GetAttrPath("attr").GetAttr("after").GetAttr("a"),
			},
			before: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.DynamicVal,
			}),
			after: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					// the entire attr object is taken here because there is
					// nothing to compare within the before value
					"after": cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("new"),
						"b": cty.StringVal("new"),
					}),
					"other": cty.ObjectVal(map[string]cty.Value{
						"o": cty.StringVal("new"),
					}),
				}),
			}),
			expected: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"after": cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("new"),
						"b": cty.StringVal("new"),
					}),
					// "other" is picked up here too this time, because we need
					// to take the entire dynamic "attr" value
					"other": cty.ObjectVal(map[string]cty.Value{
						"o": cty.StringVal("new"),
					}),
				}),
			}),
		},
		"whole object becomes null": {
			// whole object becomes null
			paths: []cty.Path{
				cty.GetAttrPath("attr").GetAttr("after").GetAttr("a"),
			},
			before: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"after": cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("old"),
						"b": cty.StringVal("old"),
					}),
				}),
			}),
			after: cty.NullVal(cty.Object(map[string]cty.Type{
				"attr": cty.DynamicPseudoType,
			})),
			// since we have a dynamic type we have to take the entire object
			// because the paths may not apply between versions.
			expected: cty.NullVal(cty.Object(map[string]cty.Type{
				"attr": cty.DynamicPseudoType,
			})),
		},
		"whole object was null": {
			// whole object was null
			paths: []cty.Path{
				cty.GetAttrPath("attr").GetAttr("after").GetAttr("a"),
			},
			before: cty.NullVal(cty.Object(map[string]cty.Type{
				"attr": cty.DynamicPseudoType,
			})),
			after: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"after": cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("new"),
						"b": cty.StringVal("new"),
					}),
				}),
			}),
			expected: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"after": cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("new"),
						"b": cty.StringVal("new"),
					}),
				}),
			}),
		},
		"restructured dynamic": {
			// dynamic value changing structure significantly
			paths: []cty.Path{
				cty.GetAttrPath("attr").GetAttr("list").IndexInt(1).GetAttr("a"),
			},
			before: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"list": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"a": cty.StringVal("old"),
						}),
					}),
				}),
			}),
			after: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"after": cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("new"),
						"b": cty.StringVal("new"),
					}),
				}),
			}),
			// the path does not apply at all to the new object, so we must
			// take all the changes
			expected: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.ObjectVal(map[string]cty.Value{
					"after": cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("new"),
						"b": cty.StringVal("new"),
					}),
				}),
			}),
		},
	}

	for k, tc := range tests {
		t.Run(k, func(t *testing.T) {
			addr, diags := addrs.ParseAbsResourceInstanceStr("test_resource.a")
			if diags != nil {
				t.Fatal(diags.ErrWithWarnings())
			}

			change := &plans.ResourceInstanceChange{
				Addr: addr,
				Change: plans.Change{
					Before: tc.before,
					After:  tc.after,
					Action: plans.Update,
				},
			}

			var contributing []globalref.ResourceAttr
			for _, p := range tc.paths {
				contributing = append(contributing, globalref.ResourceAttr{
					Resource: addr,
					Attr:     p,
				})
			}

			res := filterRefreshChange(change, contributing)
			if !res.After.RawEquals(tc.expected) {
				t.Errorf("\nexpected: %#v\ngot:      %#v\n", tc.expected, res.After)
			}
		})
	}
}
