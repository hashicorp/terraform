// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Plan_QueryContext(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					test = {
						source = "hashicorp/test"
						version = "1.0.0"
					}
				}
			}
	`,
		"main.tfquery.hcl": `
			provider "test" {}

			variable "input" {
				type = string
				default = "test"
			}

			list "test_resource" "test" {
				provider = test

				filter = {
					attr = var.input
				}
			}
	`,
	})

	p := simpleMockProvider()
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"attr": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
		ListResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"filter": {
						Required: true,
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"attr": {
									Type:     cty.String,
									Required: true,
								},
							},
						},
					},
				},
			},
		},
	})
	p.ListResourceFn = func(request providers.ListResourceRequest) error {
		filter := request.Config.GetAttr("filter")
		str := filter.GetAttr("attr").AsString()
		if str != "inputed" {
			return fmt.Errorf("Expected filter attr to be 'inputed', got '%s'", str)
		}
		for _, attr := range []string{"attr1", "attr2"} {
			request.ResourceEmitter(providers.ListResult{
				ResourceObject: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal(attr),
				}),
			})
		}
		request.DoneCh <- struct{}{}
		return nil
	}
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	qv := &MockQueryViews{
		ResourceAddrs: addrs.MakeMap[addrs.AbsResourceInstance, []*states.ResourceInstanceObjectSrc](),
	}
	_, _, diags := ctx.PlanAndEval(m, states.NewState(), &PlanOpts{
		QueryViews: qv,
		SetVariables: InputValues{
			"input": &InputValue{
				Value: cty.StringVal("inputed"),
			},
		},
		Mode: plans.NormalMode,
	})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if !qv.ResourceCalled {
		t.Fatal("Resource was not called")
	}
	root := addrs.RootModuleInstance
	objs := qv.ResourceAddrs.Get(root.ResourceInstance(addrs.ListResourceMode, "test_resource", "test", addrs.NoKey))
	if len(objs) != 2 {
		t.Fatalf("Expected 2 resource objects, got %d", len(qv.ResourceAddrs.Elements()))
	}

	obj, err := objs[0].Decode(p.GetProviderSchemaResponse.ListResourceTypes["test_resource"])
	if err != nil {
		t.Fatalf("Failed to decode resource object: %s", err)
	}

	if obj.Value.GetAttr("attr").AsString() != "attr1" {
		t.Fatalf("Expected attr to be 'attr1', got '%s'", obj.Value.GetAttr("attr").AsString())
	}
}

func TestContext2Plan_QueryContextCount(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					test = {
						source = "hashicorp/test"
						version = "1.0.0"
					}
				}
			}
	`,
		"main.tfquery.hcl": `
			provider "test" {}

			variable "input" {
				type = string
				default = "test"
			}

			list "test_resource" "test" {
				provider = test

				filter = {
					attr = var.input
				}
			}

			list "test_child_resource" "test_child" {
				count = 2
				provider = test

				filter = {
					attr = count.index
				}
			}
	`,
	})

	testSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"filter": {
				Required: true,
				NestedType: &configschema.Object{
					Nesting: configschema.NestingSingle,
					Attributes: map[string]*configschema.Attribute{
						"attr": {
							Type:     cty.String,
							Required: true,
						},
					},
				},
			},
		},
	}

	p := simpleMockProvider()
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"attr": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
			"test_child_resource": {
				Attributes: map[string]*configschema.Attribute{
					"attr": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
		ListResourceTypes: map[string]*configschema.Block{
			"test_resource":       testSchema,
			"test_child_resource": testSchema,
		},
	})
	p.ListResourceFn = func(request providers.ListResourceRequest) error {
		filter := request.Config.GetAttr("filter")
		str := filter.GetAttr("attr").AsString()
		if str != "inputed" && request.TypeName == "test_resource" {
			return fmt.Errorf("Expected filter attr to be 'inputed' for test_resource, got '%s'", str)
		}
		if request.TypeName == "test_child_resource" {
			request.ResourceEmitter(providers.ListResult{
				ResourceObject: cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("child_attr"),
				}),
			})
		} else {
			for _, attr := range []string{"attr1", "attr2"} {
				request.ResourceEmitter(providers.ListResult{
					ResourceObject: cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal(attr),
					}),
				})
			}
		}
		request.DoneCh <- struct{}{}
		return nil
	}
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
		Parallelism: 1,
	})

	qv := &MockQueryViews{
		ResourceAddrs: addrs.MakeMap[addrs.AbsResourceInstance, []*states.ResourceInstanceObjectSrc](),
	}
	_, _, diags := ctx.PlanAndEval(m, states.NewState(), &PlanOpts{
		QueryViews: qv,
		SetVariables: InputValues{
			"input": &InputValue{
				Value: cty.StringVal("inputed"),
			},
		},
		Mode: plans.NormalMode,
	})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if !qv.ResourceCalled {
		t.Fatal("Resource was not called")
	}

	root := addrs.RootModuleInstance
	objs := qv.ResourceAddrs.Get(root.ResourceInstance(addrs.ListResourceMode, "test_resource", "test", addrs.NoKey))
	if len(objs) != 2 {
		t.Fatalf("Expected 2 resource objects, got %d", len(qv.ResourceAddrs.Elements()))
	}

	obj, err := objs[0].Decode(p.GetProviderSchemaResponse.ListResourceTypes["test_resource"])
	if err != nil {
		t.Fatalf("Failed to decode resource object: %s", err)
	}

	if obj.Value.GetAttr("attr").AsString() != "attr1" {
		t.Fatalf("Expected attr to be 'attr1', got '%s'", obj.Value.GetAttr("attr").AsString())
	}

	childObj, ok := qv.ResourceAddrs.GetOk(root.ResourceInstance(addrs.ListResourceMode, "test_child_resource", "test_child", addrs.IntKey(1)))
	if !ok || len(childObj) != 1 {
		t.Fatal("Expected 1 resource object, got none")
	}
	childObj, ok = qv.ResourceAddrs.GetOk(root.ResourceInstance(addrs.ListResourceMode, "test_child_resource", "test_child", addrs.IntKey(0)))
	if !ok || len(childObj) != 1 {
		t.Fatal("Expected 1 resource object, got none")
	}

	objj, err := childObj[0].Decode(p.GetProviderSchemaResponse.ListResourceTypes["test_child_resource"])
	if err != nil {
		t.Fatalf("Failed to decode resource object: %s", err)
	}

	if objj.Value.GetAttr("attr").AsString() != "child_attr" {
		t.Fatalf("Expected child attr to be 'child_attr', got '%s'", objj.Value.GetAttr("attr").AsString())
	}

}

// MockQueryViews is a mock implementation of the QueryViews interface for testing.
type MockQueryViews struct {
	ListCalled     bool
	ListStatesArg  ListStates
	ResourceCalled bool
	ResourceAddrs  addrs.Map[addrs.AbsResourceInstance, []*states.ResourceInstanceObjectSrc]
}

func (m *MockQueryViews) List(states ListStates) {
	m.ListCalled = true
	m.ListStatesArg = states
}

func (m *MockQueryViews) Resource(addr addrs.AbsResourceInstance, obj *states.ResourceInstanceObjectSrc) {
	m.ResourceCalled = true
	if !m.ResourceAddrs.Has(addr) {
		m.ResourceAddrs.Put(addr, []*states.ResourceInstanceObjectSrc{})
	}
	m.ResourceAddrs.Put(addr, append(m.ResourceAddrs.Get(addr), obj))
}
