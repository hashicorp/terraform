package moduletest

import (
	"fmt"
	"sync"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/repl"
	"github.com/hashicorp/terraform/tfdiags"
)

// Provider is an implementation of providers.Interface which we're
// using as a likely-only-temporary vehicle for research on an opinionated
// module testing workflow in Terraform.
//
// We expose this to configuration as "terraform.io/builtin/test", but
// any attempt to configure it will emit a warning that it is experimental
// and likely to change or be removed entirely in future Terraform CLI
// releases.
//
// The testing provider exists to gather up test results during a Terraform
// apply operation. Its "test_results" managed resource type doesn't have any
// user-visible effect on its own, but when used in conjunction with the
// "terraform test" experimental command it is the intermediary that holds
// the test results while the test runs, so that the test command can then
// report them.
//
// For correct behavior of the assertion tracking, the "terraform test"
// command must be sure to use the same instance of Provider for both the
// plan and apply steps, so that the assertions that were planned can still
// be tracked during apply. For other commands that don't explicitly support
// test assertions, the provider will still succeed but the assertions data
// may not be complete if the apply step fails.
type Provider struct {
	// components tracks all of the "component" names that have been
	// used in test assertions resources so far. Each resource must have
	// a unique component name.
	components map[string]*Component

	// Must lock mutex in order to interact with the components map, because
	// test assertions can potentially run concurrently.
	mutex sync.RWMutex
}

var _ providers.Interface = (*Provider)(nil)

// NewProvider returns a new instance of the test provider.
func NewProvider() *Provider {
	return &Provider{
		components: make(map[string]*Component),
	}
}

// TestResults returns the current record of test results tracked inside the
// provider.
//
// The result is a direct reference to the internal state of the provider,
// so the caller mustn't modify it nor store it across calls to provider
// operations.
func (p *Provider) TestResults() map[string]*Component {
	return p.components
}

// GetSchema returns the complete schema for the provider.
func (p *Provider) GetSchema() providers.GetSchemaResponse {
	return providers.GetSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_assertions": testAssertionsSchema,
		},
	}
}

// PrepareProviderConfig is used to tweak the configuration values.
func (p *Provider) PrepareProviderConfig(req providers.PrepareProviderConfigRequest) providers.PrepareProviderConfigResponse {
	// This provider has no configurable settings.
	var res providers.PrepareProviderConfigResponse
	res.PreparedConfig = req.Config
	return res
}

// Configure configures and initializes the provider.
func (p *Provider) Configure(providers.ConfigureRequest) providers.ConfigureResponse {
	// This provider has no configurable settings, but we use the configure
	// request as an opportunity to generate a warning about it being
	// experimental.
	var res providers.ConfigureResponse
	res.Diagnostics = res.Diagnostics.Append(tfdiags.AttributeValue(
		tfdiags.Warning,
		"The test provider is experimental",
		"The Terraform team is using the test provider (terraform.io/builtin/test) as part of ongoing research about declarative testing of Terraform modules.\n\nThe availability and behavior of this provider is expected to change significantly even in patch releases, so we recommend using this provider only in test configurations and constraining your test configurations to an exact Terraform version.",
		nil,
	))
	return res
}

// ValidateResourceTypeConfig is used to validate configuration values for a resource.
func (p *Provider) ValidateResourceTypeConfig(req providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
	var res providers.ValidateResourceTypeConfigResponse
	if req.TypeName != "test_assertions" { // we only have one resource type
		res.Diagnostics = res.Diagnostics.Append(fmt.Errorf("unsupported resource type %s", req.TypeName))
		return res
	}

	config := req.Config
	if !config.GetAttr("component").IsKnown() {
		res.Diagnostics = res.Diagnostics.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid component expression",
			"The component name must be a static value given in the configuration, and may not be derived from a resource type attribute that will only be known during the apply step.",
			cty.GetAttrPath("component"),
		))
	}
	if !hclsyntax.ValidIdentifier(config.GetAttr("component").AsString()) {
		res.Diagnostics = res.Diagnostics.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid component name",
			"The component name must be a valid identifier, starting with a letter followed by zero or more letters, digits, and underscores.",
			cty.GetAttrPath("component"),
		))
	}
	for it := config.GetAttr("equal").ElementIterator(); it.Next(); {
		k, obj := it.Element()
		if !hclsyntax.ValidIdentifier(k.AsString()) {
			res.Diagnostics = res.Diagnostics.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid assertion name",
				"An assertion name must be a valid identifier, starting with a letter followed by zero or more letters, digits, and underscores.",
				cty.GetAttrPath("equal").Index(k),
			))
		}
		if !obj.GetAttr("description").IsKnown() {
			res.Diagnostics = res.Diagnostics.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid description expression",
				"The description must be a static value given in the configuration, and may not be derived from a resource type attribute that will only be known during the apply step.",
				cty.GetAttrPath("equal").Index(k).GetAttr("description"),
			))
		}
	}
	for it := config.GetAttr("check").ElementIterator(); it.Next(); {
		k, obj := it.Element()
		if !hclsyntax.ValidIdentifier(k.AsString()) {
			res.Diagnostics = res.Diagnostics.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid assertion name",
				"An assertion name must be a valid identifier, starting with a letter followed by zero or more letters, digits, and underscores.",
				cty.GetAttrPath("check").Index(k),
			))
		}
		if !obj.GetAttr("description").IsKnown() {
			res.Diagnostics = res.Diagnostics.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid description expression",
				"The description must be a static value given in the configuration, and may not be derived from a resource type attribute that will only be known during the apply step.",
				cty.GetAttrPath("equal").Index(k).GetAttr("description"),
			))
		}
	}

	return res
}

// ReadResource refreshes a resource and returns its current state.
func (p *Provider) ReadResource(req providers.ReadResourceRequest) providers.ReadResourceResponse {
	var res providers.ReadResourceResponse
	if req.TypeName != "test_assertions" { // we only have one resource type
		res.Diagnostics = res.Diagnostics.Append(fmt.Errorf("unsupported resource type %s", req.TypeName))
		return res
	}
	// Test assertions are not a real remote object, so there isn't actually
	// anything to refresh here.
	res.NewState = req.PriorState
	return res
}

// UpgradeResourceState is called to allow the provider to adapt the raw value
// stored in the state in case the schema has changed since it was originally
// written.
func (p *Provider) UpgradeResourceState(req providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	var res providers.UpgradeResourceStateResponse
	if req.TypeName != "test_assertions" { // we only have one resource type
		res.Diagnostics = res.Diagnostics.Append(fmt.Errorf("unsupported resource type %s", req.TypeName))
		return res
	}

	// We assume here that there can never be a flatmap version of this
	// resource type's data, because this provider was never included in a
	// version of Terraform that used flatmap and this provider's schema
	// contains attributes that are not flatmap-compatible anyway.
	if len(req.RawStateFlatmap) != 0 {
		res.Diagnostics = res.Diagnostics.Append(fmt.Errorf("can't upgrade a flatmap state for %q", req.TypeName))
		return res
	}
	if req.Version != 0 {
		res.Diagnostics = res.Diagnostics.Append(fmt.Errorf("the state for this %s was created by a newer version of the provider", req.TypeName))
		return res
	}

	v, err := ctyjson.Unmarshal(req.RawStateJSON, testAssertionsSchema.Block.ImpliedType())
	if err != nil {
		res.Diagnostics = res.Diagnostics.Append(fmt.Errorf("failed to decode state for %s: %s", req.TypeName, err))
		return res
	}

	res.UpgradedState = v
	return res
}

// PlanResourceChange takes the current state and proposed state of a
// resource, and returns the planned final state.
func (p *Provider) PlanResourceChange(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	var res providers.PlanResourceChangeResponse
	if req.TypeName != "test_assertions" { // we only have one resource type
		res.Diagnostics = res.Diagnostics.Append(fmt.Errorf("unsupported resource type %s", req.TypeName))
		return res
	}

	// During planning, our job is to gather up all of the planned test
	// assertions marked as pending, which will then allow us to include
	// all of them in test results even if there's a failure during apply
	// that prevents the full completion of the graph walk.
	//
	// In a sense our plan phase is similar to the compile step for a
	// test program written in another language. Planning itself can fail,
	// which means we won't be able to form a complete test plan at all,
	// but if we succeed in planning then subsequent problems can be treated
	// as test failures at "runtime", while still keeping a full manifest
	// of all of the tests that ought to have run if the apply had run to
	// completion.

	proposed := req.ProposedNewState
	res.PlannedState = proposed
	componentName := proposed.GetAttr("component").AsString() // proven known during validate
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if _, exists := p.components[componentName]; exists {
		res.Diagnostics = res.Diagnostics.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Duplicate test component",
			fmt.Sprintf("Another test_assertions resource already declared assertions for the component name %q.", componentName),
			cty.GetAttrPath("component"),
		))
		return res
	}

	component := Component{
		Assertions: make(map[string]*Assertion),
	}

	for it := proposed.GetAttr("equal").ElementIterator(); it.Next(); {
		k, obj := it.Element()
		name := k.AsString()
		if _, exists := component.Assertions[name]; exists {
			// We can't actually get here in practice because so far we've
			// only been pulling keys from one map, and so any duplicates
			// would've been caught during config decoding, but this is here
			// just to make these two blocks symmetrical to avoid mishaps in
			// future refactoring/reorganization.
			res.Diagnostics = res.Diagnostics.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Duplicate test assertion",
				fmt.Sprintf("Another assertion block in this resource already declared an assertion named %q.", name),
				cty.GetAttrPath("equal").Index(k),
			))
			continue
		}

		var desc string
		descVal := obj.GetAttr("description")
		if descVal.IsNull() {
			descVal = cty.StringVal("")
		}
		err := gocty.FromCtyValue(descVal, &desc)
		if err != nil {
			// We shouldn't get here because we've already validated everything
			// that would make FromCtyValue fail above and during validate.
			res.Diagnostics = res.Diagnostics.Append(err)
		}

		component.Assertions[name] = &Assertion{
			Outcome:     Pending,
			Description: desc,
		}
	}

	for it := proposed.GetAttr("check").ElementIterator(); it.Next(); {
		k, obj := it.Element()
		name := k.AsString()
		if _, exists := component.Assertions[name]; exists {
			res.Diagnostics = res.Diagnostics.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Duplicate test assertion",
				fmt.Sprintf("Another assertion block in this resource already declared an assertion named %q.", name),
				cty.GetAttrPath("check").Index(k),
			))
			continue
		}

		var desc string
		descVal := obj.GetAttr("description")
		if descVal.IsNull() {
			descVal = cty.StringVal("")
		}
		err := gocty.FromCtyValue(descVal, &desc)
		if err != nil {
			// We shouldn't get here because we've already validated everything
			// that would make FromCtyValue fail above and during validate.
			res.Diagnostics = res.Diagnostics.Append(err)
		}

		component.Assertions[name] = &Assertion{
			Outcome:     Pending,
			Description: desc,
		}
	}

	p.components[componentName] = &component
	return res
}

// ApplyResourceChange takes the planned state for a resource, which may
// yet contain unknown computed values, and applies the changes returning
// the final state.
func (p *Provider) ApplyResourceChange(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	var res providers.ApplyResourceChangeResponse
	if req.TypeName != "test_assertions" { // we only have one resource type
		res.Diagnostics = res.Diagnostics.Append(fmt.Errorf("unsupported resource type %s", req.TypeName))
		return res
	}

	// During apply we actually check the assertions and record the results.
	// An assertion failure isn't reflected as an error from the apply call
	// because if possible we'd like to continue exercising other objects
	// downstream in case that allows us to gather more information to report.
	// (If something downstream returns an error then that could prevent us
	// from completing other assertions, though.)

	planned := req.PlannedState
	res.NewState = planned
	componentName := planned.GetAttr("component").AsString() // proven known during validate

	p.mutex.Lock()
	defer p.mutex.Unlock()
	component := p.components[componentName]
	if component == nil {
		// We might get here when using this provider outside of the
		// "terraform test" command, where there won't be any mechanism to
		// preserve the test provider instance between the plan and apply
		// phases. In that case, we assume that nobody will come looking to
		// collect the results anyway, and so we can just silently skip
		// checking.
		return res
	}

	for it := planned.GetAttr("equal").ElementIterator(); it.Next(); {
		k, obj := it.Element()
		name := k.AsString()
		var desc string
		if plan, exists := component.Assertions[name]; exists {
			desc = plan.Description
		}
		assert := &Assertion{
			Outcome:     Pending,
			Description: desc,
		}

		gotVal := obj.GetAttr("got")
		wantVal := obj.GetAttr("want")
		switch {
		case wantVal.RawEquals(gotVal):
			assert.Outcome = Passed
			gotStr := repl.FormatValue(gotVal, 4)
			assert.Message = fmt.Sprintf("correct value\n    got: %s\n", gotStr)
		default:
			assert.Outcome = Failed
			gotStr := repl.FormatValue(gotVal, 4)
			wantStr := repl.FormatValue(wantVal, 4)
			assert.Message = fmt.Sprintf("wrong value\n    got:  %s\n    want: %s\n", gotStr, wantStr)
		}

		component.Assertions[name] = assert
	}

	for it := planned.GetAttr("check").ElementIterator(); it.Next(); {
		k, obj := it.Element()
		name := k.AsString()
		var desc string
		if plan, exists := component.Assertions[name]; exists {
			desc = plan.Description
		}
		assert := &Assertion{
			Outcome:     Pending,
			Description: desc,
		}

		condVal := obj.GetAttr("condition")
		switch {
		case condVal.IsNull():
			res.Diagnostics = res.Diagnostics.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid check condition",
				"The condition value must be a boolean expression, not null.",
				cty.GetAttrPath("check").Index(k).GetAttr("condition"),
			))
			continue
		case condVal.True():
			assert.Outcome = Passed
			assert.Message = "condition passed"
		default:
			assert.Outcome = Failed
			// For "check" we can't really return a decent error message
			// because we've lost all of the context by the time we get here.
			// "equal" will be better for most tests for that reason, and also
			// this is one reason why in the long run it would be better for
			// test assertions to be a first-class language feature rather than
			// just a provider-based concept.
			assert.Message = "condition failed"
		}

		component.Assertions[name] = assert
	}

	return res
}

// ImportResourceState requests that the given resource be imported.
func (p *Provider) ImportResourceState(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	var res providers.ImportResourceStateResponse
	res.Diagnostics = res.Diagnostics.Append(fmt.Errorf("%s is not importable", req.TypeName))
	return res
}

// ValidateDataSourceConfig is used to to validate the resource configuration values.
func (p *Provider) ValidateDataSourceConfig(req providers.ValidateDataSourceConfigRequest) providers.ValidateDataSourceConfigResponse {
	// This provider has no data resouce types at all.
	var res providers.ValidateDataSourceConfigResponse
	res.Diagnostics = res.Diagnostics.Append(fmt.Errorf("unsupported data source %s", req.TypeName))
	return res
}

// ReadDataSource returns the data source's current state.
func (p *Provider) ReadDataSource(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	// This provider has no data resouce types at all.
	var res providers.ReadDataSourceResponse
	res.Diagnostics = res.Diagnostics.Append(fmt.Errorf("unsupported data source %s", req.TypeName))
	return res
}

// Stop is called when the provider should halt any in-flight actions.
func (p *Provider) Stop() error {
	// This provider doesn't do anything that can be cancelled.
	return nil
}

// Close is a noop for this provider, since it's run in-process.
func (p *Provider) Close() error {
	return nil
}

var testAssertionsSchema = providers.Schema{
	Block: &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"component": {
				Type:            cty.String,
				Description:     "The name of the component being tested. This is just for namespacing assertions in a result report.",
				DescriptionKind: configschema.StringPlain,
				Required:        true,
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"equal": {
				Nesting: configschema.NestingMap,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"description": {
							Type:            cty.String,
							Description:     "An optional human-readable description of what's being tested by this assertion.",
							DescriptionKind: configschema.StringPlain,
							Required:        true,
						},
						"got": {
							Type:            cty.DynamicPseudoType,
							Description:     "The actual result value generated by the relevant component.",
							DescriptionKind: configschema.StringPlain,
							Required:        true,
						},
						"want": {
							Type:            cty.DynamicPseudoType,
							Description:     "The value that the component is expected to have generated.",
							DescriptionKind: configschema.StringPlain,
							Required:        true,
						},
					},
				},
			},
			"check": {
				Nesting: configschema.NestingMap,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"description": {
							Type:            cty.String,
							Description:     "An optional (but strongly recommended) human-readable description of what's being tested by this assertion.",
							DescriptionKind: configschema.StringPlain,
							Required:        true,
						},
						"condition": {
							Type:            cty.Bool,
							Description:     "An expression that must be true in order for the test to pass.",
							DescriptionKind: configschema.StringPlain,
							Required:        true,
						},
					},
				},
			},
		},
	},
}
