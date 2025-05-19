package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	// When this attribute is set to plan, the values specified in the override
	// block will be used for computed attributes even when planning. It defaults
	// to apply, meaning that the values will only be used during apply.
	overrideDuringCommand = "override_during"
)

func decodeMockProviderBlock(block *hcl.Block) (*Provider, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, config, moreDiags := block.Body.PartialContent(mockProviderSchema)
	diags = append(diags, moreDiags...)

	name := block.Labels[0]
	nameDiags := checkProviderNameNormalized(name, block.DefRange)
	diags = append(diags, nameDiags...)
	if nameDiags.HasErrors() {
		// If the name is invalid then we mustn't produce a result because
		// downstream could try to use it as a provider type and then crash.
		return nil, diags
	}

	provider := &Provider{
		Name:      name,
		NameRange: block.LabelRanges[0],
		DeclRange: block.DefRange,

		// Mock providers shouldn't need any additional data.
		Config: hcl.EmptyBody(),

		// Mark this provider as being mocked.
		Mock: true,
	}

	if attr, exists := content.Attributes["alias"]; exists {
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &provider.Alias)
		diags = append(diags, valDiags...)
		provider.AliasRange = attr.Expr.Range().Ptr()

		if !hclsyntax.ValidIdentifier(provider.Alias) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider configuration alias",
				Detail:   fmt.Sprintf("An alias must be a valid name. %s", badIdentifierDetail),
			})
		}
	}

	useForPlan, useForPlanDiags := useForPlan(content, false)
	diags = append(diags, useForPlanDiags...)
	provider.MockDataDuringPlan = useForPlan

	var dataDiags hcl.Diagnostics
	provider.MockData, dataDiags = decodeMockDataBody(config, useForPlan, MockProviderOverrideSource)
	diags = append(diags, dataDiags...)

	if attr, exists := content.Attributes["source"]; exists {
		sourceDiags := gohcl.DecodeExpression(attr.Expr, nil, &provider.MockDataExternalSource)
		diags = append(diags, sourceDiags...)
	}

	return provider, diags
}

func useForPlan(content *hcl.BodyContent, def bool) (bool, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	if attr, exists := content.Attributes[overrideDuringCommand]; exists {
		switch hcl.ExprAsKeyword(attr.Expr) {
		case "plan":
			return true, diags
		case "apply":
			return false, diags
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Invalid %s value", overrideDuringCommand),
				Detail:   fmt.Sprintf("The %s attribute must be a value of plan or apply.", overrideDuringCommand),
				Subject:  attr.Range.Ptr(),
			})
			return def, diags
		}
	}
	return def, diags
}

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

func decodeMockDataBody(body hcl.Body, useForPlanDefault bool, source OverrideSource) (*MockData, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, contentDiags := body.Content(mockDataSchema)
	diags = append(diags, contentDiags...)

	data := &MockData{
		MockResources:   make(map[string]*MockResource),
		MockDataSources: make(map[string]*MockResource),
		Overrides:       addrs.MakeMap[addrs.Targetable, *Override](),
	}

	for _, block := range content.Blocks {
		switch block.Type {
		case "mock_resource", "mock_data":
			resource, resourceDiags := decodeMockResourceBlock(block, useForPlanDefault)
			diags = append(diags, resourceDiags...)

			if resource != nil {
				switch resource.Mode {
				case addrs.ManagedResourceMode:
					if previous, ok := data.MockResources[resource.Type]; ok {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Duplicate mock_resource block",
							Detail:   fmt.Sprintf("A mock_resource block for %s has already been defined at %s.", resource.Type, previous.Range),
							Subject:  resource.TypeRange.Ptr(),
						})
						continue
					}
					data.MockResources[resource.Type] = resource
				case addrs.DataResourceMode:
					if previous, ok := data.MockDataSources[resource.Type]; ok {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Duplicate mock_data block",
							Detail:   fmt.Sprintf("A mock_data block for %s has already been defined at %s.", resource.Type, previous.Range),
							Subject:  resource.TypeRange.Ptr(),
						})
						continue
					}
					data.MockDataSources[resource.Type] = resource
				}
			}
		case "override_resource":
			override, overrideDiags := decodeOverrideResourceBlock(block, useForPlanDefault, source)
			diags = append(diags, overrideDiags...)

			if override != nil && override.Target != nil {
				subject := override.Target.Subject
				if previous, ok := data.Overrides.GetOk(subject); ok {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate override_resource block",
						Detail:   fmt.Sprintf("An override_resource block targeting %s has already been defined at %s.", subject, previous.Range),
						Subject:  override.Range.Ptr(),
					})
					continue
				}
				data.Overrides.Put(subject, override)
			}
		case "override_data":
			override, overrideDiags := decodeOverrideDataBlock(block, useForPlanDefault, source)
			diags = append(diags, overrideDiags...)

			if override != nil && override.Target != nil {
				subject := override.Target.Subject
				if previous, ok := data.Overrides.GetOk(subject); ok {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate override_data block",
						Detail:   fmt.Sprintf("An override_data block targeting %s has already been defined at %s.", subject, previous.Range),
						Subject:  override.Range.Ptr(),
					})
					continue
				}
				data.Overrides.Put(subject, override)
			}
		}
	}

	return data, diags
}

func decodeMockResourceBlock(block *hcl.Block, useForPlanDefault bool) (*MockResource, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, contentDiags := block.Body.Content(mockResourceSchema)
	diags = append(diags, contentDiags...)

	resource := &MockResource{
		Type:      block.Labels[0],
		Range:     block.DefRange,
		TypeRange: block.LabelRanges[0],
	}

	switch block.Type {
	case "mock_resource":
		resource.Mode = addrs.ManagedResourceMode
	case "mock_data":
		resource.Mode = addrs.DataResourceMode
	}

	if defaults, exists := content.Attributes["defaults"]; exists {
		var defaultDiags hcl.Diagnostics
		resource.DefaultsRange = defaults.Range
		resource.Defaults, defaultDiags = defaults.Expr.Value(nil)
		diags = append(diags, defaultDiags...)
	} else {
		// It's fine if we don't have any defaults, just means we'll generate
		// values for everything ourselves.
		resource.Defaults = cty.NilVal
	}

	useForPlan, useForPlanDiags := useForPlan(content, useForPlanDefault)
	diags = append(diags, useForPlanDiags...)
	resource.UseForPlan = useForPlan

	return resource, diags
}

func decodeOverrideModuleBlock(block *hcl.Block, useForPlanDefault bool, source OverrideSource) (*Override, hcl.Diagnostics) {
	override, diags := decodeOverrideBlock(block, "outputs", "override_module", useForPlanDefault, source)

	if override.Target != nil {
		switch override.Target.Subject.AddrType() {
		case addrs.ModuleAddrType, addrs.ModuleInstanceAddrType:
			// Do nothing, we're good here.
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid override target",
				Detail:   fmt.Sprintf("You can only target modules from override_module blocks, not %s.", override.Target.Subject),
				Subject:  override.TargetRange.Ptr(),
			})
			return nil, diags
		}
	}

	return override, diags
}

func decodeOverrideResourceBlock(block *hcl.Block, useForPlanDefault bool, source OverrideSource) (*Override, hcl.Diagnostics) {
	override, diags := decodeOverrideBlock(block, "values", "override_resource", useForPlanDefault, source)

	if override.Target != nil {
		var mode addrs.ResourceMode

		switch override.Target.Subject.AddrType() {
		case addrs.AbsResourceInstanceAddrType:
			subject := override.Target.Subject.(addrs.AbsResourceInstance)
			mode = subject.Resource.Resource.Mode
		case addrs.AbsResourceAddrType:
			subject := override.Target.Subject.(addrs.AbsResource)
			mode = subject.Resource.Mode
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid override target",
				Detail:   fmt.Sprintf("You can only target resources from override_resource blocks, not %s.", override.Target.Subject),
				Subject:  override.TargetRange.Ptr(),
			})
			return nil, diags
		}

		if mode != addrs.ManagedResourceMode {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid override target",
				Detail:   fmt.Sprintf("You can only target resources from override_resource blocks, not %s.", override.Target.Subject),
				Subject:  override.TargetRange.Ptr(),
			})
			return nil, diags
		}
	}

	return override, diags
}

func decodeOverrideDataBlock(block *hcl.Block, useForPlanDefault bool, source OverrideSource) (*Override, hcl.Diagnostics) {
	override, diags := decodeOverrideBlock(block, "values", "override_data", useForPlanDefault, source)

	if override.Target != nil {
		var mode addrs.ResourceMode

		switch override.Target.Subject.AddrType() {
		case addrs.AbsResourceInstanceAddrType:
			subject := override.Target.Subject.(addrs.AbsResourceInstance)
			mode = subject.Resource.Resource.Mode
		case addrs.AbsResourceAddrType:
			subject := override.Target.Subject.(addrs.AbsResource)
			mode = subject.Resource.Mode
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid override target",
				Detail:   fmt.Sprintf("You can only target data sources from override_data blocks, not %s.", override.Target.Subject),
				Subject:  override.TargetRange.Ptr(),
			})
			return nil, diags
		}

		if mode != addrs.DataResourceMode {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid override target",
				Detail:   fmt.Sprintf("You can only target data sources from override_data blocks, not %s.", override.Target.Subject),
				Subject:  override.TargetRange.Ptr(),
			})
			return nil, diags
		}
	}

	return override, diags
}

func decodeOverrideBlock(block *hcl.Block, attributeName string, blockName string, useForPlanDefault bool, source OverrideSource) (*Override, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, contentDiags := block.Body.Content(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "target"},
			{Name: overrideDuringCommand},
			{Name: attributeName},
		},
	})
	diags = append(diags, contentDiags...)

	override := &Override{
		Source:    source,
		Range:     block.DefRange,
		TypeRange: block.TypeRange,
	}

	if target, exists := content.Attributes["target"]; exists {
		override.TargetRange = target.Range
		traversal, traversalDiags := hcl.AbsTraversalForExpr(target.Expr)
		diags = append(diags, traversalDiags...)
		if traversal != nil {
			var targetDiags tfdiags.Diagnostics
			override.Target, targetDiags = addrs.ParseTarget(traversal)
			diags = append(diags, targetDiags.ToHCL()...)
		}
	} else {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing target attribute",
			Detail:   fmt.Sprintf("%s blocks must specify a target address.", blockName),
			Subject:  override.Range.Ptr(),
		})
	}

	if attribute, exists := content.Attributes[attributeName]; exists {
		var valueDiags hcl.Diagnostics
		override.ValuesRange = attribute.Range
		override.Values, valueDiags = attribute.Expr.Value(nil)
		diags = append(diags, valueDiags...)
	} else {
		// It's fine if we don't have any values, just means we'll generate
		// values for everything ourselves. We set this to an empty object so
		// it's equivalent to `values = {}` which makes later processing easier.
		override.Values = cty.EmptyObjectVal
	}

	useForPlan, useForPlanDiags := useForPlan(content, useForPlanDefault)
	diags = append(diags, useForPlanDiags...)
	override.UseForPlan = useForPlan

	if !override.Values.Type().IsObjectType() {

		var attributePreposition string
		switch attributeName {
		case "outputs":
			attributePreposition = "an"
		default:
			attributePreposition = "a"
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Invalid %s attribute", attributeName),
			Detail:   fmt.Sprintf("%s blocks must specify %s %s attribute that is an object.", blockName, attributePreposition, attributeName),
			Subject:  override.ValuesRange.Ptr(),
		})
	}

	return override, diags
}

var mockProviderSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "alias",
		},
		{
			Name: "source",
		},
		{
			Name: overrideDuringCommand,
		},
	},
}

var mockDataSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "mock_resource", LabelNames: []string{"type"}},
		{Type: "mock_data", LabelNames: []string{"type"}},
		{Type: "override_resource"},
		{Type: "override_data"},
	},
}

var mockResourceSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "defaults"},
		{Name: overrideDuringCommand},
	},
}
