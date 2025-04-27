// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

func TestQueryContext(t *testing.T) {

	// this should always plan a NoOp change for the output
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
		ResourceAddrs: addrs.MakeSet[addrs.List](),
		ResourceObjs:  []*states.ResourceInstanceObjectSrc{},
	}
	_, diags := ctx.QueryEval(m, &QueryOpts{
		View: qv,
		SetVariables: InputValues{
			"input": &InputValue{
				Value: cty.StringVal("inputed"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if !qv.ResourceCalled {
		t.Fatal("Resource was not called")
	}
	if len(qv.ResourceObjs) != 2 {
		t.Fatalf("Expected 2 resource objects, got %d", len(qv.ResourceObjs))
	}

	obj, err := qv.ResourceObjs[0].Decode(p.GetProviderSchemaResponse.ListResourceTypes["test_resource"])
	if err != nil {
		t.Fatalf("Failed to decode resource object: %s", err)
	}

	if obj.Value.GetAttr("attr").AsString() != "attr1" {
		t.Fatalf("Expected attr to be 'attr1', got '%s'", obj.Value.GetAttr("attr").AsString())
	}

}

// MockQueryViews is a mock implementation of the QueryViews interface for testing.
type MockQueryViews struct {
	ListCalled     bool
	ListStatesArg  ListStates
	ResourceCalled bool
	ResourceAddrs  addrs.Set[addrs.List]
	ResourceObjs   []*states.ResourceInstanceObjectSrc
}

func (m *MockQueryViews) List(states ListStates) {
	m.ListCalled = true
	m.ListStatesArg = states
}

func (m *MockQueryViews) Resource(addr addrs.List, obj *states.ResourceInstanceObjectSrc) {
	m.ResourceCalled = true
	m.ResourceAddrs.Add(addr)
	m.ResourceObjs = append(m.ResourceObjs, obj)
}
