package terraform

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/tfdiags"
)

var _ providers.Interface = (*MockProvider)(nil)

// MockProvider implements providers.Interface but mocks out all the
// calls for testing purposes.
type MockProvider struct {
	sync.Mutex

	// Anything you want, in case you need to store extra data with the mock.
	Meta interface{}

	GetSchemaCalled bool
	GetSchemaReturn *ProviderSchema // This is using ProviderSchema directly rather than providers.GetSchemaResponse for compatibility with old tests

	PrepareProviderConfigCalled   bool
	PrepareProviderConfigResponse providers.PrepareProviderConfigResponse
	PrepareProviderConfigRequest  providers.PrepareProviderConfigRequest
	PrepareProviderConfigFn       func(providers.PrepareProviderConfigRequest) providers.PrepareProviderConfigResponse

	ValidateResourceTypeConfigCalled   bool
	ValidateResourceTypeConfigTypeName string
	ValidateResourceTypeConfigResponse providers.ValidateResourceTypeConfigResponse
	ValidateResourceTypeConfigRequest  providers.ValidateResourceTypeConfigRequest
	ValidateResourceTypeConfigFn       func(providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse

	ValidateDataSourceConfigCalled   bool
	ValidateDataSourceConfigTypeName string
	ValidateDataSourceConfigResponse providers.ValidateDataSourceConfigResponse
	ValidateDataSourceConfigRequest  providers.ValidateDataSourceConfigRequest
	ValidateDataSourceConfigFn       func(providers.ValidateDataSourceConfigRequest) providers.ValidateDataSourceConfigResponse

	UpgradeResourceStateCalled   bool
	UpgradeResourceStateTypeName string
	UpgradeResourceStateResponse providers.UpgradeResourceStateResponse
	UpgradeResourceStateRequest  providers.UpgradeResourceStateRequest
	UpgradeResourceStateFn       func(providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse

	ConfigureCalled   bool
	ConfigureResponse providers.ConfigureResponse
	ConfigureRequest  providers.ConfigureRequest
	ConfigureNewFn    func(providers.ConfigureRequest) providers.ConfigureResponse // Named ConfigureNewFn so we can still have the legacy ConfigureFn declared below

	StopCalled   bool
	StopFn       func() error
	StopResponse error

	ReadResourceCalled   bool
	ReadResourceResponse providers.ReadResourceResponse
	ReadResourceRequest  providers.ReadResourceRequest
	ReadResourceFn       func(providers.ReadResourceRequest) providers.ReadResourceResponse

	PlanResourceChangeCalled   bool
	PlanResourceChangeResponse providers.PlanResourceChangeResponse
	PlanResourceChangeRequest  providers.PlanResourceChangeRequest
	PlanResourceChangeFn       func(providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse

	ApplyResourceChangeCalled   bool
	ApplyResourceChangeResponse providers.ApplyResourceChangeResponse
	ApplyResourceChangeRequest  providers.ApplyResourceChangeRequest
	ApplyResourceChangeFn       func(providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse

	ImportResourceStateCalled   bool
	ImportResourceStateResponse providers.ImportResourceStateResponse
	ImportResourceStateRequest  providers.ImportResourceStateRequest
	ImportResourceStateFn       func(providers.ImportResourceStateRequest) providers.ImportResourceStateResponse
	// Legacy return type for existing tests, which will be shimmed into an
	// ImportResourceStateResponse if set
	ImportStateReturn []*InstanceState

	ReadDataSourceCalled   bool
	ReadDataSourceResponse providers.ReadDataSourceResponse
	ReadDataSourceRequest  providers.ReadDataSourceRequest
	ReadDataSourceFn       func(providers.ReadDataSourceRequest) providers.ReadDataSourceResponse

	CloseCalled bool
	CloseError  error

	// Legacy callbacks: if these are set, we will shim incoming calls for
	// new-style methods to these old-fashioned terraform.ResourceProvider
	// mock callbacks, for the benefit of older tests that were written against
	// the old mock API.
	ValidateFn  func(c *ResourceConfig) (ws []string, es []error)
	ConfigureFn func(c *ResourceConfig) error
	DiffFn      func(info *InstanceInfo, s *InstanceState, c *ResourceConfig) (*InstanceDiff, error)
	ApplyFn     func(info *InstanceInfo, s *InstanceState, d *InstanceDiff) (*InstanceState, error)
}

func (p *MockProvider) GetSchema() providers.GetSchemaResponse {
	p.Lock()
	defer p.Unlock()
	p.GetSchemaCalled = true
	return p.getSchema()
}

func (p *MockProvider) getSchema() providers.GetSchemaResponse {
	// This version of getSchema doesn't do any locking, so it's suitable to
	// call from other methods of this mock as long as they are already
	// holding the lock.

	ret := providers.GetSchemaResponse{
		Provider:      providers.Schema{},
		DataSources:   map[string]providers.Schema{},
		ResourceTypes: map[string]providers.Schema{},
	}
	if p.GetSchemaReturn != nil {
		ret.Provider.Block = p.GetSchemaReturn.Provider
		for n, s := range p.GetSchemaReturn.DataSources {
			ret.DataSources[n] = providers.Schema{
				Block: s,
			}
		}
		for n, s := range p.GetSchemaReturn.ResourceTypes {
			ret.ResourceTypes[n] = providers.Schema{
				Version: int64(p.GetSchemaReturn.ResourceTypeSchemaVersions[n]),
				Block:   s,
			}
		}
	}

	return ret
}

func (p *MockProvider) PrepareProviderConfig(r providers.PrepareProviderConfigRequest) providers.PrepareProviderConfigResponse {
	p.Lock()
	defer p.Unlock()

	p.PrepareProviderConfigCalled = true
	p.PrepareProviderConfigRequest = r
	if p.PrepareProviderConfigFn != nil {
		return p.PrepareProviderConfigFn(r)
	}
	return p.PrepareProviderConfigResponse
}

func (p *MockProvider) ValidateResourceTypeConfig(r providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
	p.Lock()
	defer p.Unlock()

	p.ValidateResourceTypeConfigCalled = true
	p.ValidateResourceTypeConfigRequest = r

	if p.ValidateFn != nil {
		resp := p.getSchema()
		schema := resp.Provider.Block
		rc := NewResourceConfigShimmed(r.Config, schema)
		warns, errs := p.ValidateFn(rc)
		ret := providers.ValidateResourceTypeConfigResponse{}
		for _, warn := range warns {
			ret.Diagnostics = ret.Diagnostics.Append(tfdiags.SimpleWarning(warn))
		}
		for _, err := range errs {
			ret.Diagnostics = ret.Diagnostics.Append(err)
		}
	}
	if p.ValidateResourceTypeConfigFn != nil {
		return p.ValidateResourceTypeConfigFn(r)
	}

	return p.ValidateResourceTypeConfigResponse
}

func (p *MockProvider) ValidateDataSourceConfig(r providers.ValidateDataSourceConfigRequest) providers.ValidateDataSourceConfigResponse {
	p.Lock()
	defer p.Unlock()

	p.ValidateDataSourceConfigCalled = true
	p.ValidateDataSourceConfigRequest = r

	if p.ValidateDataSourceConfigFn != nil {
		return p.ValidateDataSourceConfigFn(r)
	}

	return p.ValidateDataSourceConfigResponse
}

func (p *MockProvider) UpgradeResourceState(r providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	p.Lock()
	defer p.Unlock()

	schemas := p.getSchema()
	schema := schemas.ResourceTypes[r.TypeName]
	schemaType := schema.Block.ImpliedType()

	p.UpgradeResourceStateCalled = true
	p.UpgradeResourceStateRequest = r

	if p.UpgradeResourceStateFn != nil {
		return p.UpgradeResourceStateFn(r)
	}

	resp := p.UpgradeResourceStateResponse

	if resp.UpgradedState == cty.NilVal {
		switch {
		case r.RawStateFlatmap != nil:
			v, err := hcl2shim.HCL2ValueFromFlatmap(r.RawStateFlatmap, schemaType)
			if err != nil {
				resp.Diagnostics = resp.Diagnostics.Append(err)
				return resp
			}
			resp.UpgradedState = v
		case len(r.RawStateJSON) > 0:
			v, err := ctyjson.Unmarshal(r.RawStateJSON, schemaType)

			if err != nil {
				resp.Diagnostics = resp.Diagnostics.Append(err)
				return resp
			}
			resp.UpgradedState = v
		}
	}
	return resp
}

func (p *MockProvider) Configure(r providers.ConfigureRequest) providers.ConfigureResponse {
	p.Lock()
	defer p.Unlock()

	p.ConfigureCalled = true
	p.ConfigureRequest = r

	if p.ConfigureFn != nil {
		resp := p.getSchema()
		schema := resp.Provider.Block
		rc := NewResourceConfigShimmed(r.Config, schema)
		ret := providers.ConfigureResponse{}

		err := p.ConfigureFn(rc)
		if err != nil {
			ret.Diagnostics = ret.Diagnostics.Append(err)
		}
		return ret
	}
	if p.ConfigureNewFn != nil {
		return p.ConfigureNewFn(r)
	}

	return p.ConfigureResponse
}

func (p *MockProvider) Stop() error {
	// We intentionally don't lock in this one because the whole point of this
	// method is to be called concurrently with another operation that can
	// be cancelled.  The provider itself is responsible for handling
	// any concurrency concerns in this case.

	p.StopCalled = true
	if p.StopFn != nil {
		return p.StopFn()
	}

	return p.StopResponse
}

func (p *MockProvider) ReadResource(r providers.ReadResourceRequest) providers.ReadResourceResponse {
	p.Lock()
	defer p.Unlock()

	p.ReadResourceCalled = true
	p.ReadResourceRequest = r

	if p.ReadResourceFn != nil {
		return p.ReadResourceFn(r)
	}

	// make sure the NewState fits the schema
	newState, err := p.GetSchemaReturn.ResourceTypes[r.TypeName].CoerceValue(p.ReadResourceResponse.NewState)
	if err != nil {
		panic(err)
	}
	resp := p.ReadResourceResponse
	resp.NewState = newState

	return resp
}

func (p *MockProvider) PlanResourceChange(r providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	p.Lock()
	defer p.Unlock()

	p.PlanResourceChangeCalled = true
	p.PlanResourceChangeRequest = r

	if p.DiffFn != nil {
		ps := p.getSchema()
		if ps.ResourceTypes == nil || ps.ResourceTypes[r.TypeName].Block == nil {
			return providers.PlanResourceChangeResponse{
				Diagnostics: tfdiags.Diagnostics(nil).Append(fmt.Printf("mock provider has no schema for resource type %s", r.TypeName)),
			}
		}
		schema := ps.ResourceTypes[r.TypeName].Block
		info := &InstanceInfo{
			Type: r.TypeName,
		}
		priorState := NewInstanceStateShimmedFromValue(r.PriorState, 0)
		cfg := NewResourceConfigShimmed(r.Config, schema)

		legacyDiff, err := p.DiffFn(info, priorState, cfg)

		var res providers.PlanResourceChangeResponse
		res.PlannedState = r.ProposedNewState
		if err != nil {
			res.Diagnostics = res.Diagnostics.Append(err)
		}
		if legacyDiff != nil {
			newVal, err := legacyDiff.ApplyToValue(r.PriorState, schema)
			if err != nil {
				res.Diagnostics = res.Diagnostics.Append(err)
			}

			res.PlannedState = newVal

			var requiresNew []string
			for attr, d := range legacyDiff.Attributes {
				if d.RequiresNew {
					requiresNew = append(requiresNew, attr)
				}
			}
			requiresReplace, err := hcl2shim.RequiresReplace(requiresNew, schema.ImpliedType())
			if err != nil {
				res.Diagnostics = res.Diagnostics.Append(err)
			}
			res.RequiresReplace = requiresReplace
		}
		return res
	}
	if p.PlanResourceChangeFn != nil {
		return p.PlanResourceChangeFn(r)
	}

	return p.PlanResourceChangeResponse
}

func (p *MockProvider) ApplyResourceChange(r providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	p.Lock()
	p.ApplyResourceChangeCalled = true
	p.ApplyResourceChangeRequest = r
	p.Unlock()

	if p.ApplyFn != nil {
		// ApplyFn is a special callback fashioned after our old provider
		// interface, which expected to be given an actual diff rather than
		// separate old/new values to apply. Therefore we need to approximate
		// a diff here well enough that _most_ of our legacy ApplyFns in old
		// tests still see the behavior they are expecting. New tests should
		// not use this, and should instead use ApplyResourceChangeFn directly.
		providerSchema := p.getSchema()
		schema, ok := providerSchema.ResourceTypes[r.TypeName]
		if !ok {
			return providers.ApplyResourceChangeResponse{
				Diagnostics: tfdiags.Diagnostics(nil).Append(fmt.Errorf("no mocked schema available for resource type %s", r.TypeName)),
			}
		}

		info := &InstanceInfo{
			Type: r.TypeName,
		}

		priorVal := r.PriorState
		plannedVal := r.PlannedState
		priorMap := hcl2shim.FlatmapValueFromHCL2(priorVal)
		plannedMap := hcl2shim.FlatmapValueFromHCL2(plannedVal)
		s := NewInstanceStateShimmedFromValue(priorVal, 0)
		d := &InstanceDiff{
			Attributes: make(map[string]*ResourceAttrDiff),
		}
		if plannedMap == nil { // destroying, then
			d.Destroy = true
			// Destroy diffs don't have any attribute diffs
		} else {
			if priorMap == nil { // creating, then
				// We'll just make an empty prior map to make things easier below.
				priorMap = make(map[string]string)
			}

			for k, new := range plannedMap {
				old := priorMap[k]
				newComputed := false
				if new == hcl2shim.UnknownVariableValue {
					new = ""
					newComputed = true
				}
				d.Attributes[k] = &ResourceAttrDiff{
					Old:         old,
					New:         new,
					NewComputed: newComputed,
					Type:        DiffAttrInput, // not generally used in tests, so just hard-coded
				}
			}
			// Also need any attributes that were removed in "planned"
			for k, old := range priorMap {
				if _, ok := plannedMap[k]; ok {
					continue
				}
				d.Attributes[k] = &ResourceAttrDiff{
					Old:        old,
					NewRemoved: true,
					Type:       DiffAttrInput,
				}
			}
		}
		newState, err := p.ApplyFn(info, s, d)
		resp := providers.ApplyResourceChangeResponse{}
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(err)
		}
		if newState != nil {
			var newVal cty.Value
			if newState != nil {
				var err error
				newVal, err = newState.AttrsAsObjectValue(schema.Block.ImpliedType())
				if err != nil {
					resp.Diagnostics = resp.Diagnostics.Append(err)
				}
			} else {
				// If apply returned a nil new state then that's the old way to
				// indicate that the object was destroyed. Our new interface calls
				// for that to be signalled as a null value.
				newVal = cty.NullVal(schema.Block.ImpliedType())
			}
			resp.NewState = newVal
		}

		return resp
	}
	if p.ApplyResourceChangeFn != nil {
		return p.ApplyResourceChangeFn(r)
	}

	return p.ApplyResourceChangeResponse
}

func (p *MockProvider) ImportResourceState(r providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	p.Lock()
	defer p.Unlock()

	if p.ImportStateReturn != nil {
		for _, is := range p.ImportStateReturn {
			if is.Attributes == nil {
				is.Attributes = make(map[string]string)
			}
			is.Attributes["id"] = is.ID

			typeName := is.Ephemeral.Type
			// Use the requested type if the resource has no type of it's own.
			// We still return the empty type, which will error, but this prevents a panic.
			if typeName == "" {
				typeName = r.TypeName
			}

			schema := p.GetSchemaReturn.ResourceTypes[typeName]
			if schema == nil {
				panic("no schema found for " + typeName)
			}

			private, err := json.Marshal(is.Meta)
			if err != nil {
				panic(err)
			}

			state, err := hcl2shim.HCL2ValueFromFlatmap(is.Attributes, schema.ImpliedType())
			if err != nil {
				panic(err)
			}

			state, err = schema.CoerceValue(state)
			if err != nil {
				panic(err)
			}

			p.ImportResourceStateResponse.ImportedResources = append(
				p.ImportResourceStateResponse.ImportedResources,
				providers.ImportedResource{
					TypeName: is.Ephemeral.Type,
					State:    state,
					Private:  private,
				})
		}
	}

	p.ImportResourceStateCalled = true
	p.ImportResourceStateRequest = r
	if p.ImportResourceStateFn != nil {
		return p.ImportResourceStateFn(r)
	}

	return p.ImportResourceStateResponse
}

func (p *MockProvider) ReadDataSource(r providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	p.Lock()
	defer p.Unlock()

	p.ReadDataSourceCalled = true
	p.ReadDataSourceRequest = r

	if p.ReadDataSourceFn != nil {
		return p.ReadDataSourceFn(r)
	}

	return p.ReadDataSourceResponse
}

func (p *MockProvider) Close() error {
	p.CloseCalled = true
	return p.CloseError
}
