// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

// MockData packages up all the available mock and override data available to
// a mocked provider.
type MockData struct {
	MockResources   map[string]*MockResource
	MockDataSources map[string]*MockResource
	Overrides       addrs.Map[addrs.Targetable, *Override]
}

// Merge will merge the target MockData object into the current MockData.
//
// If skipCollisions is true, then Merge will simply ignore any entries within
// other that clash with entries already in data. If skipCollisions is false,
// then we will create diagnostics for each duplicate resource.
func (data *MockData) Merge(other *MockData, skipCollisions bool) (diags hcl.Diagnostics) {
	if other == nil {
		return diags
	}

	for name, resource := range other.MockResources {
		current, exists := data.MockResources[name]
		if !exists {
			data.MockResources[name] = resource
			continue
		}

		if skipCollisions {
			continue
		}

		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Duplicate mock resource block",
			Detail:   fmt.Sprintf("A mock_resource %q block already exists at %s.", name, current.Range),
			Subject:  resource.TypeRange.Ptr(),
		})
	}
	for name, datasource := range other.MockDataSources {
		current, exists := data.MockDataSources[name]
		if !exists {
			data.MockDataSources[name] = datasource
			continue
		}

		if skipCollisions {
			continue
		}

		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Duplicate mock resource block",
			Detail:   fmt.Sprintf("A mock_data %q block already exists at %s.", name, current.Range),
			Subject:  datasource.TypeRange.Ptr(),
		})
	}
	for _, elem := range other.Overrides.Elems {
		target, override := elem.Key, elem.Value

		current, exists := data.Overrides.GetOk(target)
		if !exists {
			data.Overrides.Put(target, override)
			continue
		}

		if skipCollisions {
			continue
		}

		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Duplicate override block",
			Detail:   fmt.Sprintf("An override block for %s already exists at %s.", target, current.Range),
			Subject:  override.Range.Ptr(),
		})
	}
	return diags
}

// MockResource maps a resource or data source type and name to a set of values
// for that resource.
type MockResource struct {
	Mode addrs.ResourceMode
	Type string

	Defaults cty.Value
	RawExpr  hcl.Expression

	// UseForPlan is true if the values should be computed during the planning
	// phase.
	UseForPlan bool

	Range         hcl.Range
	TypeRange     hcl.Range
	DefaultsRange hcl.Range
}

type OverrideSource int

const (
	UnknownOverrideSource OverrideSource = iota
	RunBlockOverrideSource
	TestFileOverrideSource
	MockProviderOverrideSource
	MockDataFileOverrideSource
)

// Override targets a specific module, resource or data source with a set of
// replacement values that should be used in place of whatever the underlying
// provider would normally do.
type Override struct {
	Target *addrs.Target
	Values cty.Value

	BlockName string

	// The raw expression of the values/outputs block
	RawExpr hcl.Expression

	// UseForPlan is true if the values should be computed during the planning
	// phase.
	UseForPlan bool

	// Source tells us where this Override was defined.
	Source OverrideSource

	Range       hcl.Range
	TypeRange   hcl.Range
	TargetRange hcl.Range
	ValuesRange hcl.Range
}
