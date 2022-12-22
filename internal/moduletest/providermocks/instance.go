package providermocks

import (
	"bytes"
	"fmt"
	"log"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/dynblock"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Instantiate wraps an unconfigured instance of this configuration's target
// provider to create a mock provider whose validation behaviors pass through
// to the wrapped provider but whose read, plan, and apply operations are
// handled only using the mock configuration.
//
// If the mock configuration in the receiver does not conform to the schema
// of the given provider then this will return error diagnostics and no
// mock provider instance.
//
// After passing a provider instance to this function the returned mock
// provider has full ownership of it. The caller should no longer interact
// directly with that instance or unpredictable things will happen.
func (c *Config) Instantiate(wrapped providers.Interface) (providers.Interface, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	schemaResp := wrapped.GetProviderSchema()
	diags = diags.Append(schemaResp.Diagnostics)
	if schemaResp.Diagnostics.HasErrors() {
		return nil, diags
	}
	schemas, err := schemaResp.Schemas()
	if err != nil {
		// TODO: A better error diagnostic
		diags = diags.Append(err)
		return nil, diags
	}
	diags = diags.Append(
		c.Validate(schemas),
	)
	if diags.HasErrors() {
		return nil, diags
	}

	return &mockProvider{
		Wrapped: wrapped,
		Schemas: schemas,
		Config:  c,
	}, diags
}

type mockProvider struct {
	Wrapped providers.Interface
	Schemas *providers.Schemas
	Config  *Config
}

func (p *mockProvider) resourceTypeInfo(ty ResourceType) (*configschema.Block, *ResourceTypeConfig, tfdiags.Diagnostics) {
	schema, _ := p.Schemas.SchemaForResourceType(ty.Mode, ty.Type)
	if schema == nil {
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.WholeContainingBody(
			tfdiags.Error,
			"Unsupported resource type",
			fmt.Sprintf("This provider does not support the resource type %s.", ty),
		))
		return nil, nil, diags
	}

	config, defined := p.Config.ResourceTypes[ty]
	if !defined {
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.WholeContainingBody(
			tfdiags.Error,
			"No mock definition for resource type",
			fmt.Sprintf("The mock provider does not include a definition for %s.", ty),
		))
		return nil, nil, diags
	}

	return schema, config, nil
}

// The functions that Terraform calls on _unconfigured_ providers will pass
// through directly to the wrapped provider. In particular that means that
// the mocks will still enforce all of the same validation rules that the
// real provider would, and so the author of the mock configuration can
// assume that any incoming configuration will always be valid per the
// provider's rules without having to reimplement all of those rules.

func (p *mockProvider) GetProviderSchema() providers.GetProviderSchemaResponse {
	if p.Wrapped == nil {
		// Since GetProviderSchema is likely to be the first method called
		// when trying to re-use an already-closed mock provider, we'll
		// add a real error message here even though all of the other methods
		// will just panic.
		var resp providers.GetProviderSchemaResponse
		resp.Diagnostics = resp.Diagnostics.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Re-using a closed mock provider",
			fmt.Sprintf("This mock provider for %s appears to have already been closed and so can no longer be used. (This is a bug in Terraform.)", p.Config.ForProvider),
		))
		log.Printf("[DEBUG] mock %s: GetProviderSchema called when provider was already closed", p.Config.ForProvider)
		return resp
	}
	log.Printf("[DEBUG] mock %s: GetProviderSchema delegated to real provider", p.Config.ForProvider)
	return p.Wrapped.GetProviderSchema()
}

func (p *mockProvider) ValidateProviderConfig(req providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse {
	log.Printf("[DEBUG] mock %s: ValidateProviderConfig delegated to real provider", p.Config.ForProvider)
	return p.Wrapped.ValidateProviderConfig(req)
}

func (p *mockProvider) ValidateResourceConfig(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
	log.Printf("[DEBUG] mock %s: ValidateResourceConfig delegated to real provider", p.Config.ForProvider)
	return p.Wrapped.ValidateResourceConfig(req)
}

func (p *mockProvider) ValidateDataResourceConfig(req providers.ValidateDataResourceConfigRequest) providers.ValidateDataResourceConfigResponse {
	log.Printf("[DEBUG] mock %s: ValidateDataSourceConfig delegated to real provider", p.Config.ForProvider)
	return p.Wrapped.ValidateDataResourceConfig(req)
}

func (p *mockProvider) Stop() error {
	log.Printf("[DEBUG] mock %s: Stop delegated to real provider", p.Config.ForProvider)
	return p.Wrapped.Stop()
}

func (p *mockProvider) Close() error {
	if p.Wrapped == nil {
		// We won't try to re-close a provider we already successfully closed,
		// but we'll also just treat it as a no-op so that callers don't
		// need to do so much book-keeping to ensure that all mock providers
		// will be closed only exactly once.
		log.Printf("[DEBUG] mock %s: Close ignored because the wrapped instance was already closed", p.Config.ForProvider)
		return nil
	}
	log.Printf("[DEBUG] mock %s: Close delegated to real provider", p.Config.ForProvider)
	err := p.Wrapped.Close()
	if err == nil {
		// If we successfully closed the wrapped provider then we'll discard
		// it just to make us fail fast if something tries to use it again.
		p.Wrapped = nil
		log.Printf("[DEBUG] mock %s: wrapped provider closed successfully", p.Config.ForProvider)
	} else {
		log.Printf("[DEBUG] mock %s: wrapped provider close failed: %s", p.Config.ForProvider, err)
	}
	return err
}

// All of the methods that would either configure a provider or assume that
// the provider is already configured are stubbed out, which therefore avoids
// any need to configure the real provider and thus allows a protocol-compliant
// provider to be used in tests without the need for credentials.

func (p *mockProvider) ConfigureProvider(req providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	log.Printf("[DEBUG] mock %s: ConfigureProvider ignored", p.Config.ForProvider)

	// A mock provider doesn't need any additional configuration since
	// all aspects of its behavior are defined by the mock configuration.
	return providers.ConfigureProviderResponse{}
}

func (p *mockProvider) UpgradeResourceState(req providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	log.Printf("[DEBUG] mock %s: UpgradeResourceState for %s", p.Config.ForProvider, req.TypeName)

	// This mock mechanism is for testing modules rather than providers,
	// and the idea of upgrading a prior state is an implementation detail
	// of providers that a calling module shouldn't need to worry about,
	// so we just treat upgrading as a no-op here, always just echoing back
	// whatever we were given.
	//
	// However, we do still need to parse the raw representation, since
	// it's UpgradeResourceState's responsibility to make sense of whatever
	// raw value was previously written into the state.

	var resp providers.UpgradeResourceStateResponse

	resourceType := ResourceType{
		Mode: addrs.ManagedResourceMode,
		Type: req.TypeName,
	}
	schema, _, moreDiags := p.resourceTypeInfo(resourceType)
	resp.Diagnostics = resp.Diagnostics.Append(moreDiags)
	if moreDiags.HasErrors() {
		return resp
	}

	switch {
	case len(req.RawStateJSON) != 0:
		ty := schema.ImpliedType()
		v, err := ctyjson.Unmarshal(req.RawStateJSON, ty)
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(tfdiags.WholeContainingBody(
				tfdiags.Error,
				"Invalid prior state data",
				fmt.Sprintf(
					"The prior state for this object is not valid: %s.",
					tfdiags.FormatError(err),
				),
			))
			return resp
		}
		resp.UpgradedState = v
		return resp

	default:
		// We don't expect any other formats because mock providers didn't
		// exist in any Terraform version that would've generated flatmap.
		resp.Diagnostics = resp.Diagnostics.Append(tfdiags.WholeContainingBody(
			tfdiags.Error,
			"Unsupported state object format",
			"The prior state for this object is in a format other than the expected internal JSON format.",
		))
		return resp
	}
}

func (p *mockProvider) ReadResource(req providers.ReadResourceRequest) providers.ReadResourceResponse {
	log.Printf("[DEBUG] mock %s: ReadResource for %s", p.Config.ForProvider, req.TypeName)

	var resp providers.ReadResourceResponse

	resourceType := ResourceType{
		Mode: addrs.ManagedResourceMode,
		Type: req.TypeName,
	}
	schema, config, moreDiags := p.resourceTypeInfo(resourceType)
	resp.Diagnostics = resp.Diagnostics.Append(moreDiags)
	if moreDiags.HasErrors() {
		return resp
	}

	mppIn := unmarshalMockProviderPrivate(req.Private)
	mppOut := make(mockProviderPrivate, 1)
	mockResps := config.Responses[readRequest]

	if len(mockResps) == 0 {
		// If the mock author didn't define any read responses at all then
		// we'll just echo back the input exactly, which will be a reasonable
		// behavior for most tests that aren't intentionally trying to
		// exercise responses to changes outside of Terraform.
		resp.NewState = req.PriorState
		log.Printf("[DEBUG] mock %s: using built-in ReadResource behavior for %s", p.Config.ForProvider, resourceType)
		return resp
	}

	// Otherwise, we need to try each of the configured responses in turn
	// until we find one which has a passing condition. If we find one
	// then we'll use its content to overwrite values in the request
	// to create the effect of changes made outside of Terraform.
	evalVars := map[string]cty.Value{
		"previous_plan_response":  mppIn.ObjectVal(planRequest),
		"previous_apply_response": mppIn.ObjectVal(applyRequest),
		"previous_state":          req.PriorState,
	}
	chosen, modObj, moreDiags := p.chooseMockResponse(readRequest, resourceType, mockResps, evalVars, schema)
	resp.Diagnostics = resp.Diagnostics.Append(moreDiags)
	if moreDiags.HasErrors() {
		return resp
	}

	// FIXME: This isn't actually sufficient. We instead need to deep-merge the
	// modObj data into the prior state so that we'll preserve anything that
	// the configuration didn't set.
	// (also FIXME: we've thrown away the information about whether the
	// arguments were explicitly set in the content block, and so we won't
	// be able to allow the mock to explicitly force something to be null
	// without taking a different approach to resolving these.)
	resp.NewState = modObj

	mppOut[readRequest] = chosen.Name
	resp.Private = marshalMockProviderPrivate(mppOut)

	return resp
}

func (p *mockProvider) PlanResourceChange(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	log.Printf("[DEBUG] mock %s: PlanResourceChange for %s", p.Config.ForProvider, req.TypeName)

	var resp providers.PlanResourceChangeResponse

	// If this is a plan to destroy an object then we just handle it in a
	// standard way, since there's nothing to customize here.
	// (Real providers can potentially block attempts to destroy objects, but
	// that's rare and not something a module developer should typically need
	// to test for.)
	if req.ProposedNewState.IsNull() {
		log.Printf("[DEBUG] mock %s: PlanResourceChange built-in destroy plan behavior for %s", p.Config.ForProvider, req.TypeName)
		resp.PlannedState = req.ProposedNewState
		return resp
	}

	resourceType := ResourceType{
		Mode: addrs.ManagedResourceMode,
		Type: req.TypeName,
	}
	schema, config, moreDiags := p.resourceTypeInfo(resourceType)
	resp.Diagnostics = resp.Diagnostics.Append(moreDiags)
	if moreDiags.HasErrors() {
		return resp
	}

	mppIn := unmarshalMockProviderPrivate(req.PriorPrivate)
	mppOut := make(mockProviderPrivate, 1)
	mockResps := config.Responses[planRequest]

	if len(mockResps) == 0 {
		// If the mock author didn't define any plan responses at all then
		// we'll just echo back Terraform Core's proposed new state, which
		// is the result of Terraform Core's built-in behavior of merging
		// the current config with the prior state.
		resp.PlannedState = req.ProposedNewState
		log.Printf("[DEBUG] mock %s: using built-in PlanResourceChange behavior for %s", p.Config.ForProvider, resourceType)
		return resp
	}

	// Otherwise, we need to try each of the configured responses in turn
	// until we find one which has a passing condition. If we find one
	// then we'll use its content to overwrite values in the request
	// to create the effect of changes made outside of Terraform.
	evalVars := map[string]cty.Value{
		"read_response":  mppIn.ObjectVal(readRequest),
		"current_state":  req.PriorState,
		"config":         req.Config,
		"proposed_state": req.ProposedNewState,
	}
	chosen, modObj, moreDiags := p.chooseMockResponse(planRequest, resourceType, mockResps, evalVars, schema)
	resp.Diagnostics = resp.Diagnostics.Append(moreDiags)
	if moreDiags.HasErrors() {
		return resp
	}

	// FIXME: This isn't actually sufficient. We instead need to deep-merge the
	// modObj data into the prior state so that we'll preserve anything that
	// the configuration didn't set.
	// (also FIXME: we've thrown away the information about whether the
	// arguments were explicitly set in the content block, and so we won't
	// be able to allow the mock to explicitly force something to be null
	// without taking a different approach to resolving these.)
	resp.PlannedState = modObj

	mppOut[planRequest] = chosen.Name
	resp.PlannedPrivate = marshalMockProviderPrivate(mppOut)

	return resp
}

func (p *mockProvider) ApplyResourceChange(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	log.Printf("[DEBUG] mock %s: ApplyResourceChange for %s", p.Config.ForProvider, req.TypeName)

	var resp providers.ApplyResourceChangeResponse

	// If this was a plan to destroy an object then we just handle it in a
	// standard way, since there's nothing to customize here.
	// (Real providers can potentially block attempts to destroy objects, but
	// that's rare and not something a module developer should typically need
	// to test for.)
	if req.PlannedState.IsNull() {
		log.Printf("[DEBUG] mock %s: ApplyResourceChange built-in destroy behavior for %s", p.Config.ForProvider, req.TypeName)
		resp.NewState = req.PlannedState
		return resp
	}

	resourceType := ResourceType{
		Mode: addrs.ManagedResourceMode,
		Type: req.TypeName,
	}
	schema, config, moreDiags := p.resourceTypeInfo(resourceType)
	resp.Diagnostics = resp.Diagnostics.Append(moreDiags)
	if moreDiags.HasErrors() {
		return resp
	}

	mppIn := unmarshalMockProviderPrivate(req.PlannedPrivate)
	mppOut := make(mockProviderPrivate, 2)
	mockResps := config.Responses[applyRequest]

	// Managed resource types are guaranteed by the config loader to always
	// have at least one mock apply response. We need to try each of the
	// configured responses in turn until we find one which has a passing
	// condition. If we find one then we'll use its content to overwrite values
	// in the request to create the effect of changes made outside of Terraform.
	evalVars := map[string]cty.Value{
		"plan_response": mppIn.ObjectVal(planRequest),
		"current_state": req.PriorState,
		"config":        req.Config,
		"planned_state": req.PlannedState,
	}
	chosen, modObj, moreDiags := p.chooseMockResponse(applyRequest, resourceType, mockResps, evalVars, schema)
	resp.Diagnostics = resp.Diagnostics.Append(moreDiags)
	if moreDiags.HasErrors() {
		return resp
	}

	// FIXME: This isn't actually sufficient. We instead need to deep-merge the
	// modObj data into the prior state so that we'll preserve anything that
	// the configuration didn't set.
	// (also FIXME: we've thrown away the information about whether the
	// arguments were explicitly set in the content block, and so we won't
	// be able to allow the mock to explicitly force something to be null
	// without taking a different approach to resolving these.)
	resp.NewState = modObj

	mppOut[planRequest] = mppIn[planRequest]
	mppOut[applyRequest] = chosen.Name
	resp.Private = marshalMockProviderPrivate(mppOut)

	return resp
}

func (p *mockProvider) ImportResourceState(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	log.Printf("[DEBUG] mock %s: ImportResourceState for %s (not implemented)", p.Config.ForProvider, req.TypeName)

	var resp providers.ImportResourceStateResponse
	resp.Diagnostics = resp.Diagnostics.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Mock provider cannot import into state",
		"Mock providers for testing do not support state import.",
	))
	return resp
}

func (p *mockProvider) ReadDataSource(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	log.Printf("[DEBUG] mock %s: ReadDataSource for data.%s", p.Config.ForProvider, req.TypeName)

	panic("not implemented")
}

func (p *mockProvider) chooseMockResponse(reqType requestType, resType ResourceType, candidates []*ResponseConfig, evalVars map[string]cty.Value, contentSchema *configschema.Block) (*ResponseConfig, cty.Value, tfdiags.Diagnostics) {
	log.Printf("[DEBUG] mock %s: choosing mock response to %s %s", p.Config.ForProvider, reqType.BlockTypeName(), resType)

	var diags tfdiags.Diagnostics
	evalCtx := p.Config.exprEvalContext(evalVars)

	// We'll try to evaluate all of the conditions first so that we can
	// immediately report diagnostics for all that are invalid, even if
	// we wouldn't otherwise have chosen them.
	//
	// We intentionally don't evaluate the bodies yet though, because
	// the condition for a response can be used as a guard to guarantee
	// that whatever it tests is true for the expressions inside the
	// content block.
	condResults := make([]checks.Status, len(candidates))
	for i, candidate := range candidates {
		// We'll assume an error unless we decide otherwise below.
		condResults[i] = checks.StatusError

		expr := candidate.Condition
		val, hclDiags := expr.Value(evalCtx)
		diags = diags.Append(hclDiags)
		if hclDiags.HasErrors() {
			continue
		}
		val, err := convert.Convert(val, cty.Bool)
		val, _ = val.Unmark()
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid condition result",
				Detail:   fmt.Sprintf("Invalid result for response condition expression: %s.", tfdiags.FormatError(err)),
				Subject:  expr.Range().Ptr(),
			})
			continue
		}
		if val.IsNull() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Null condition result",
				Detail:   "A response condition expression must always produce a specific boolean value.",
				Subject:  expr.Range().Ptr(),
			})
			continue
		}
		if !val.IsKnown() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unknown condition result",
				Detail:   "The condition result depends on values that are not yet known. A response condition expression must always produce a known boolean value.",
				Subject:  expr.Range().Ptr(),
			})
			continue
		}

		if val == cty.True {
			condResults[i] = checks.StatusPass
		} else {
			// By exclusion, the only possible remaining value is cty.False.
			condResults[i] = checks.StatusFail
		}
	}
	if diags.HasErrors() {
		// If we have any invalid conditions then we'll just halt here and not
		// try to evaluate any content blocks yet.
		return nil, cty.NilVal, diags
	}

	for i, candidate := range candidates {
		if condResults[i] != checks.StatusPass {
			continue
		}

		// If we get here then we've found the response we want to use, so
		// either we'll evaluate it and return it or we'll return an error
		// explaining why we cannot.
		chosen := candidate

		contentBody := dynblock.Expand(chosen.Content, evalCtx)
		decSpec := contentSchema.NoneRequired().DecoderSpec()
		obj, hclDiags := hcldec.Decode(contentBody, decSpec, evalCtx)
		diags = diags.Append(hclDiags)

		log.Printf("[DEBUG] mock %s: chose %s mock response for %s %q", p.Config.ForProvider, reqType.BlockTypeName(), resType, chosen.Name)
		return chosen, obj, diags
	}

	// If we get here then none of the response conditions matched at all.
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "No matching mock response",
		Detail: fmt.Sprintf(
			"None of the mock %s response conditions for %s matched this request.",
			reqType.BlockTypeName(),
			resType.String(),
		),
		// TODO: What's a reasonable range to return here? Maybe we need to
		// save the source locations of the enclosing "read", "plan", and
		// "apply" blocks so we can use the appropriate one here.
		// ...or maybe we'd be better off returning a "whole body" diagnostic
		// and let Terraform Core associate it with the resource block it
		// was trying to evaluate?
	})
	log.Printf("[ERROR] mock %s: no matching %s mock response for %s", p.Config.ForProvider, reqType.BlockTypeName(), resType)
	return nil, cty.NilVal, diags
}

type mockProviderPrivate map[requestType]string

func (mpp mockProviderPrivate) NameVal(reqType requestType) cty.Value {
	name, exists := mpp[reqType]
	if !exists {
		return cty.NullVal(cty.String)
	}
	return cty.StringVal(name)
}

func (mpp mockProviderPrivate) ObjectVal(reqType requestType) cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"name": mpp.NameVal(reqType),
	})
}

func marshalMockProviderPrivate(v mockProviderPrivate) []byte {
	// TODO: use fmt.Appendf once we're on a sufficiently new version of Go.
	return []byte(fmt.Sprintf(
		"MOCK\x31%s\x31%s\x31%s",
		v[readRequest],
		v[planRequest],
		v[applyRequest],
	))
}

func unmarshalMockProviderPrivate(raw []byte) mockProviderPrivate {
	if !bytes.HasPrefix(raw, []byte("MOCK\x31")) {
		return nil
	}
	parts := bytes.Split(raw, []byte{'\x31'})
	if len(parts) != 4 {
		return nil
	}
	ret := make(mockProviderPrivate, 3)
	if len(parts[0]) != 0 {
		ret[readRequest] = string(parts[0])
	}
	if len(parts[1]) != 0 {
		ret[planRequest] = string(parts[1])
	}
	if len(parts[2]) != 0 {
		ret[applyRequest] = string(parts[2])
	}
	return ret
}
