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

			list "test_resource" "test" {
				provider = test
			}
	`,
	})

	p := simpleMockProvider()
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		DataSources: map[string]*configschema.Block{
			"test_data_source": {
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
					"attr": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	})
	p.ListResourceFn = func(request providers.ListResourceRequest) error {
		request.ResourceEmitter(providers.ListResult{
			ResourceObject: cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("test"),
			}),
		})
		request.DoneCh <- struct{}{}
		return nil
	}
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	runner, diags := ctx.QueryEval(m, &QueryOpts{
		View: &MockQueryViews{},
	})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	fmt.Println("Runner:", runner)
}

// MockQueryViews is a mock implementation of the QueryViews interface for testing.
type MockQueryViews struct {
	ListCalled     bool
	ListStatesArg  ListStates
	ResourceCalled bool
	ResourceAddr   addrs.List
	ResourceObj    *states.ResourceInstanceObjectSrc
}

func (m *MockQueryViews) List(states ListStates) {
	m.ListCalled = true
	m.ListStatesArg = states
}

func (m *MockQueryViews) Resource(addr addrs.List, obj *states.ResourceInstanceObjectSrc) {
	m.ResourceCalled = true
	m.ResourceAddr = addr
	m.ResourceObj = obj
}
