// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackmigrate

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestMigrate(t *testing.T) {
	deposedKey := states.NewDeposedKey()

	tcs := map[string]struct {
		path          string
		state         func(ss *states.SyncState)
		resources     map[string]string
		modules       map[string]string
		expected      []stackstate.AppliedChange
		expectedDiags tfdiags.Diagnostics
	}{
		"module": {
			path: filepath.Join("with-single-input", "valid"),
			state: func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					addrs.AbsResourceInstance{
						Module: addrs.ModuleInstance{
							{
								Name: "child",
							},
						},
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "testing_resource",
								Name: "data",
							},
							Key: addrs.NoKey,
						},
					},
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceDeposed(
					addrs.AbsResourceInstance{
						Module: addrs.ModuleInstance{
							{
								Name: "child",
							},
						},
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "testing_resource",
								Name: "data",
							},
							Key: addrs.NoKey,
						},
					},
					deposedKey,
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
			},
			modules: map[string]string{
				"child": "self",
			},
			expected: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.self"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.self"),
					OutputValues:          map[addrs.OutputValue]cty.Value{},
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "id"}:    cty.DynamicVal,
						{Name: "input"}: cty.DynamicVal,
					},
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.testing_resource.data"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
						Component: mustAbsResourceInstanceObject("component.self.testing_resource.data").Component,
						Item: addrs.AbsResourceInstanceObject{
							ResourceInstance: mustAbsResourceInstanceObject("component.self.testing_resource.data").Item.ResourceInstance,
							DeposedKey:       deposedKey,
						},
					},
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
			},
		},
		"root resources": {
			path: filepath.Join("with-single-input", "valid"),
			state: func(ss *states.SyncState) {
				ss.SetResourceInstanceDeposed(
					addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "testing_resource",
								Name: "data",
							},
							Key: addrs.NoKey,
						},
					},
					deposedKey,
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "testing_resource",
								Name: "data",
							},
							Key: addrs.NoKey,
						},
					},
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
			},
			resources: map[string]string{
				"testing_resource.data": "component.self",
			},
			expected: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.self"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.self"),
					OutputValues:          map[addrs.OutputValue]cty.Value{},
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "id"}:    cty.DynamicVal,
						{Name: "input"}: cty.DynamicVal,
					},
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.testing_resource.data"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
						Component: mustAbsResourceInstanceObject("component.self.testing_resource.data").Component,
						Item: addrs.AbsResourceInstanceObject{
							ResourceInstance: mustAbsResourceInstanceObject("component.self.testing_resource.data").Item.ResourceInstance,
							DeposedKey:       deposedKey,
						},
					},
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
			},
		},
		"component_dependency": {
			path: filepath.Join("for-stacks-migrate", "with-dependency", "input-dependency"),
			state: func(ss *states.SyncState) {
				ss.SetOutputValue(addrs.AbsOutputValue{
					Module:      addrs.RootModuleInstance,
					OutputValue: addrs.OutputValue{Name: "output"},
				}, cty.StringVal("before"), false)
				ss.SetResourceInstanceCurrent(
					addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "testing_resource",
								Name: "data",
							},
							Key: addrs.NoKey,
						},
					},
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "testing_resource",
								Name: "another",
							},
							Key: addrs.IntKey(0),
						},
					},
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "testing_resource",
								Name: "another",
							},
							Key: addrs.IntKey(1),
						},
					},
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
			},
			resources: map[string]string{
				"testing_resource.data":       "component.parent",
				"testing_resource.another[0]": "component.child",
				"testing_resource.another[1]": "component.child",
			},
			modules: map[string]string{},
			expected: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.child"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.child"),
					OutputValues: map[addrs.OutputValue]cty.Value{
						{Name: "id"}: cty.DynamicVal,
					},
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "id"}:    cty.DynamicVal,
						{Name: "input"}: cty.DynamicVal,
					},
					Dependencies: collections.NewSet(mustAbsComponent("component.parent")),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.child.testing_resource.another[0]"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.child.testing_resource.another[1]"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.parent"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.parent"),
					OutputValues: map[addrs.OutputValue]cty.Value{
						{Name: "id"}: cty.DynamicVal,
					},
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "id"}:    cty.DynamicVal,
						{Name: "input"}: cty.DynamicVal,
					},
					Dependents: collections.NewSet(mustAbsComponent("component.child")),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.parent.testing_resource.data"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
			},
		},
		"nested module resources": {
			path: filepath.Join("for-stacks-migrate", "with-nested-module"),
			state: func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "data",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "another",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "another",
					}.Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				for _, child := range []string{"child_mod", "child_mod2"} {
					ss.SetResourceInstanceCurrent(
						addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "testing_resource",
							Name: "child_data",
						}.Instance(addrs.NoKey).Absolute(addrs.ModuleInstance{
							{
								Name: child,
							},
						}),
						&states.ResourceInstanceObjectSrc{
							Status: states.ObjectReady,
							AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
								"id":    "foo",
								"value": "hello",
							}),
						},
						mustDefaultRootProvider("testing"),
					)
					ss.SetResourceInstanceCurrent(
						addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "testing_resource",
							Name: "another_child_data",
						}.Instance(addrs.IntKey(0)).Absolute(addrs.ModuleInstance{
							{
								Name: child,
							},
						}),
						&states.ResourceInstanceObjectSrc{
							Status: states.ObjectReady,
							AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
								"id":    "foo",
								"value": "hello",
							}),
						},
						mustDefaultRootProvider("testing"),
					)
					ss.SetResourceInstanceCurrent(
						addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "testing_resource",
							Name: "another_child_data",
						}.Instance(addrs.IntKey(1)).Absolute(addrs.ModuleInstance{
							{
								Name: child,
							},
						}),
						&states.ResourceInstanceObjectSrc{
							Status: states.ObjectReady,
							AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
								"id":    "foo",
								"value": "hello",
							}),
						},
						mustDefaultRootProvider("testing"),
					)
				}
			},

			resources: map[string]string{
				"testing_resource.data":       "component.parent",
				"testing_resource.another[0]": "component.parent",
				"testing_resource.another[1]": "component.parent",
			},
			modules: map[string]string{
				"child_mod":  "child",
				"child_mod2": "child2",
			},
			expected: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.child"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.child"),
					OutputValues: map[addrs.OutputValue]cty.Value{
						{Name: "id"}: cty.DynamicVal,
					},
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "id"}:    cty.DynamicVal,
						{Name: "input"}: cty.DynamicVal,
					},
					Dependencies: collections.NewSet(mustAbsComponent("component.parent")),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.child.testing_resource.another_child_data[0]"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.child.testing_resource.another_child_data[1]"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.child.testing_resource.child_data"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.child2"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.child2"),
					OutputValues: map[addrs.OutputValue]cty.Value{
						{Name: "id"}: cty.DynamicVal,
					},
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "id"}:    cty.DynamicVal,
						{Name: "input"}: cty.DynamicVal,
					},
					Dependencies: collections.NewSet(mustAbsComponent("component.parent")),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.child2.testing_resource.another_child_data[0]"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.child2.testing_resource.another_child_data[1]"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.child2.testing_resource.child_data"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.parent"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.parent"),
					OutputValues: map[addrs.OutputValue]cty.Value{
						{Name: "id"}: cty.DynamicVal,
					},
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "id"}:    cty.DynamicVal,
						{Name: "input"}: cty.DynamicVal,
					},
					Dependents: collections.NewSet(mustAbsComponent("component.child"), mustAbsComponent("component.child2")),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.parent.testing_resource.another[0]"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.parent.testing_resource.another[1]"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.parent.testing_resource.data"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
			},
		},
		"missing config resource": {
			path: filepath.Join("for-stacks-migrate", "with-nested-module"),
			state: func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "data",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "another",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "another",
					}.Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "for_child",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
			},
			resources: map[string]string{
				"testing_resource.data":       "component.parent",
				"testing_resource.another[0]": "component.parent",
				"testing_resource.another[1]": "component.parent",
				"testing_resource.for_child":  "component.child",
			},
			expected: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.child"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.child"),
					OutputValues: map[addrs.OutputValue]cty.Value{
						{Name: "id"}: cty.DynamicVal,
					},
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "id"}:    cty.DynamicVal,
						{Name: "input"}: cty.DynamicVal,
					},
					Dependencies: collections.NewSet(mustAbsComponent("component.parent")),
				},
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.parent"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.parent"),
					OutputValues: map[addrs.OutputValue]cty.Value{
						{Name: "id"}: cty.DynamicVal,
					},
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "id"}:    cty.DynamicVal,
						{Name: "input"}: cty.DynamicVal,
					},
					Dependents: collections.NewSet(mustAbsComponent("component.child")),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.parent.testing_resource.another[0]"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.parent.testing_resource.another[1]"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.parent.testing_resource.data"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
			},
			expectedDiags: tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Provider not found",
				Detail:   "Resource \"testing_resource.for_child\" not found in root module.",
			}),
		},

		"missing mapping for state resource": {
			path: filepath.Join("for-stacks-migrate", "with-nested-module"),
			state: func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "data",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "another",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "another",
					}.Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "for_child",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
			},
			resources: map[string]string{
				"testing_resource.data":       "component.parent",
				"testing_resource.another[0]": "component.parent",
				"testing_resource.another[1]": "component.parent",
			},
			modules: map[string]string{},
			expected: []stackstate.AppliedChange{
				// this component has a dependent "child", but that other component
				// is not present in the modules mapping, so it is not included here
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.parent"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.parent"),
					OutputValues: map[addrs.OutputValue]cty.Value{
						{Name: "id"}: cty.DynamicVal,
					},
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "id"}:    cty.DynamicVal,
						{Name: "input"}: cty.DynamicVal,
					},
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.parent.testing_resource.another[0]"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.parent.testing_resource.another[1]"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.parent.testing_resource.data"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "hello",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
			},
			expectedDiags: tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Resource not found",
				Detail:   "Resource \"testing_resource.for_child\" not found in mapping.",
			}),
		},
		"config depends on": {
			path: filepath.Join("for-stacks-migrate", "with-depends-on"),
			state: func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "data",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "depends_test",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "second",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "depends_test",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "third",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "depends_test",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
			},
			resources: map[string]string{
				"testing_resource.data":   "component.first",
				"testing_resource.second": "component.second",
				"testing_resource.third":  "component.second",
			},
			modules: map[string]string{},
			expected: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.first"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.first"),
					OutputValues:          make(map[addrs.OutputValue]cty.Value),
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "input"}: cty.DynamicVal,
						{Name: "id"}:    cty.DynamicVal,
					},
					Dependents: collections.NewSet(mustAbsComponent("component.second")),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.first.testing_resource.data"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "depends_test",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.second"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.second"),
					OutputValues:          make(map[addrs.OutputValue]cty.Value),
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "input"}: cty.DynamicVal,
						{Name: "id"}:    cty.DynamicVal,
					},
					Dependencies: collections.NewSet(mustAbsComponent("component.first")),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.second.testing_resource.second"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "depends_test",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.second.testing_resource.third"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "depends_test",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
			},
			expectedDiags: tfdiags.Diagnostics{}.Append(),
		},
		"unsupported component ref": {
			path: filepath.Join("for-stacks-migrate", "with-depends-on"),
			state: func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "data",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "depends_test",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "second",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "depends_test",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "third",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "depends_test",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
			},
			resources: map[string]string{
				"testing_resource.data":   "component.first",
				"testing_resource.second": "component.second",
				"testing_resource.third":  "stack.embedded.component.self",
			},
			modules: map[string]string{},
			expected: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.first"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.first"),
					OutputValues:          make(map[addrs.OutputValue]cty.Value),
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "input"}: cty.DynamicVal,
						{Name: "id"}:    cty.DynamicVal,
					},
					Dependents: collections.NewSet(mustAbsComponent("component.second")),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.first.testing_resource.data"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "depends_test",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.second"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.second"),
					OutputValues:          make(map[addrs.OutputValue]cty.Value),
					InputVariables: map[addrs.InputVariable]cty.Value{
						{Name: "input"}: cty.DynamicVal,
						{Name: "id"}:    cty.DynamicVal,
					},
					Dependencies: collections.NewSet(mustAbsComponent("component.first")),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.second.testing_resource.second"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "foo",
							"value": "depends_test",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
			},
			expectedDiags: tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid component instance",
				Detail:   "Only root component instances are allowed, got \"stack.embedded.component.self\"",
			}),
		},
		"child module as component source": {
			path: filepath.Join("for-stacks-migrate", "child-module-as-component-source"),
			state: func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "root_id",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "root_id",
							"value": "root_output",
						}),
					},
					mustDefaultRootProvider("testing"),
				)

				childProv := mustDefaultRootProvider("testing")
				childProv.Module = addrs.Module{"child_module"}
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "child_data",
					}.Instance(addrs.NoKey).Absolute(addrs.ModuleInstance{
						{
							Name: "child_module",
						},
					}),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "child_data",
							"value": "child_output",
						}),
					},
					childProv,
				)
			},
			resources: map[string]string{
				"testing_resource.root_id":    "component.self",
				"testing_resource.child_data": "component.self", // this should just be ignored
			},
			modules: map[string]string{
				"child_module": "triage",
			},
			expected: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.self"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.self"),
					OutputValues:          map[addrs.OutputValue]cty.Value{},
					InputVariables:        map[addrs.InputVariable]cty.Value{},
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.testing_resource.root_id"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "root_id",
							"value": "root_output",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.triage"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.triage"),
					OutputValues:          map[addrs.OutputValue]cty.Value{},
					InputVariables: map[addrs.InputVariable]cty.Value{
						addrs.InputVariable{Name: "input"}: cty.DynamicVal,
					},
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.triage.testing_resource.child_data"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "child_data",
							"value": "child_output",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
			},
		},
		"unclaimed resources fall into modules": {
			path: filepath.Join("for-stacks-migrate", "multiple-components"),
			state: func(ss *states.SyncState) {
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "one",
					}.Instance(addrs.NoKey).Absolute(addrs.ModuleInstance{
						{
							Name: "self",
						},
					}),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "one",
							"value": "one",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
				ss.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "testing_resource",
						Name: "resource",
					}.Instance(addrs.NoKey).Absolute(addrs.ModuleInstance{
						{
							Name: "self",
						},
					}),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "two",
							"value": "two",
						}),
					},
					mustDefaultRootProvider("testing"),
				)
			},
			resources: map[string]string{
				// this specific resource goes to component.one
				"module.self.testing_resource.one": "component.one.testing_resource.resource",
			},
			modules: map[string]string{
				"self": "two", // all other resources go to component.two
			},
			expected: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.one"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.one"),
					OutputValues:          map[addrs.OutputValue]cty.Value{},
					InputVariables:        map[addrs.InputVariable]cty.Value{},
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.one.testing_resource.resource"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "one",
							"value": "one",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.two"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.two"),
					OutputValues:          map[addrs.OutputValue]cty.Value{},
					InputVariables:        map[addrs.InputVariable]cty.Value{},
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.two.testing_resource.resource"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "two",
							"value": "two",
						}),
						Status:  states.ObjectReady,
						Private: nil,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			cfg := loadMainBundleConfigForTest(t, tc.path)

			lock := depsfile.NewLocks()
			lock.SetProvider(
				addrs.NewDefaultProvider("testing"),
				providerreqs.MustParseVersion("0.0.0"),
				providerreqs.MustParseVersionConstraints("=0.0.0"),
				providerreqs.PreferredHashes([]providerreqs.Hash{}),
			)

			state := states.BuildState(tc.state)

			migration := Migration{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
						return stacks_testing_provider.NewProvider(t), nil
					},
				},
				PreviousState: state,
				Config:        cfg,
			}

			var applied []stackstate.AppliedChange
			var gotDiags tfdiags.Diagnostics

			migration.Migrate(tc.resources, tc.modules, func(change stackstate.AppliedChange) {
				applied = append(applied, change)
			}, func(diagnostic tfdiags.Diagnostic) {
				gotDiags = append(gotDiags, diagnostic)
			})

			sort.SliceStable(applied, func(i, j int) bool {
				key := func(change stackstate.AppliedChange) string {
					switch change := change.(type) {
					case *stackstate.AppliedChangeComponentInstance:
						return change.ComponentInstanceAddr.String()
					case *stackstate.AppliedChangeResourceInstanceObject:
						return change.ResourceInstanceObjectAddr.String()
					default:
						panic("unsupported change type")
					}
				}

				return key(applied[i]) < key(applied[j])
			})

			if diff := cmp.Diff(tc.expected, applied, cmp.Options{
				ctydebug.CmpOptions,
				collections.CmpOptions,
				cmpopts.IgnoreUnexported(addrs.InputVariable{}),
				cmpopts.IgnoreUnexported(states.ResourceInstanceObjectSrc{}),
			}); len(diff) > 0 {
				t.Errorf("unexpected applied changes:\n%s", diff)
			}

			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.expectedDiags)
		})
	}
}

func mustMarshalJSONAttrs(attrs map[string]interface{}) []byte {
	jsonAttrs, err := json.Marshal(attrs)
	if err != nil {
		panic(err)
	}
	return jsonAttrs
}

func mustDefaultRootProvider(provider string) addrs.AbsProviderConfig {
	return addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider(provider),
	}
}

func mustAbsResourceInstanceObject(addr string) stackaddrs.AbsResourceInstanceObject {
	ret, diags := stackaddrs.ParseAbsResourceInstanceObjectStr(addr)
	if len(diags) > 0 {
		panic(fmt.Sprintf("failed to parse resource instance object address %q: %s", addr, diags))
	}
	return ret
}

func mustAbsComponentInstance(addr string) stackaddrs.AbsComponentInstance {
	ret, diags := stackaddrs.ParsePartialComponentInstanceStr(addr)
	if len(diags) > 0 {
		panic(fmt.Sprintf("failed to parse component instance address %q: %s", addr, diags))
	}
	return ret
}

func mustAbsComponent(addr string) stackaddrs.AbsComponent {
	ret, diags := stackaddrs.ParsePartialComponentInstanceStr(addr)
	if len(diags) > 0 {
		panic(fmt.Sprintf("failed to parse component instance address %q: %s", addr, diags))
	}
	return stackaddrs.AbsComponent{
		Stack: ret.Stack,
		Item:  ret.Item.Component,
	}
}

// TODO: Perhaps export this from helper_test instead
func loadMainBundleConfigForTest(t *testing.T, dirName string) *stackconfig.Config {
	t.Helper()
	fullSourceAddr := mainBundleSourceAddrStr(dirName)
	return loadConfigForTest(t, "../stackruntime/testdata/mainbundle", fullSourceAddr)
}

func mainBundleSourceAddrStr(dirName string) string {
	return "git::https://example.com/test.git//" + dirName
}

// loadConfigForTest is a test helper that tries to open bundleRoot as a
// source bundle, and then if successful tries to load the given source address
// from it as a stack configuration. If any part of the operation fails then
// it halts execution of the test and doesn't return.
func loadConfigForTest(t *testing.T, bundleRoot string, configSourceAddr string) *stackconfig.Config {
	t.Helper()
	sources, err := sourcebundle.OpenDir(bundleRoot)
	if err != nil {
		t.Fatalf("cannot load source bundle: %s", err)
	}

	// We force using remote source addresses here because that avoids
	// us having to deal with the extra version constraints argument
	// that registry sources require. Exactly what source address type
	// we use isn't relevant for tests in this package, since it's
	// the sourcebundle package's responsibility to make sure its
	// abstraction works for all of the source types.
	sourceAddr, err := sourceaddrs.ParseRemoteSource(configSourceAddr)
	if err != nil {
		t.Fatalf("invalid config source address: %s", err)
	}

	cfg, diags := stackconfig.LoadConfigDir(sourceAddr, sources)
	reportDiagnosticsForTest(t, diags)
	return cfg
}

// reportDiagnosticsForTest creates a test log entry for every diagnostic in
// the given diags, and halts the test if any of them are error diagnostics.
func reportDiagnosticsForTest(t *testing.T, diags tfdiags.Diagnostics) {
	t.Helper()
	for _, diag := range diags {
		var b strings.Builder
		desc := diag.Description()
		locs := diag.Source()

		switch sev := diag.Severity(); sev {
		case tfdiags.Error:
			b.WriteString("Error: ")
		case tfdiags.Warning:
			b.WriteString("Warning: ")
		default:
			t.Errorf("unsupported diagnostic type %s", sev)
		}
		b.WriteString(desc.Summary)
		if desc.Address != "" {
			b.WriteString("\nwith ")
			b.WriteString(desc.Summary)
		}
		if locs.Subject != nil {
			b.WriteString("\nat ")
			b.WriteString(locs.Subject.StartString())
		}
		if desc.Detail != "" {
			b.WriteString("\n\n")
			b.WriteString(desc.Detail)
		}
		t.Log(b.String())
	}
	if diags.HasErrors() {
		t.FailNow()
	}
}
