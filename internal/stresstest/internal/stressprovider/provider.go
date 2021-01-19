package stressprovider

import (
	"fmt"
	"sync"
	"time"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/tfdiags"
)

// Provider is an implementation of providers.Interface which provides both a
// managed resource type and a data resource type that have a variety of
// functionality to help with exercising different codepaths in the stress
// testing packages.
//
// The stress-testing provider is stateful in that it remembers in RAM which
// objects have been created, and so it can implement operations such as
// refreshing from remote objects on subsequent plan operations after some
// objects have already been created. It doesn't automatically persist those
// objects anywhere else, but there are some extra methods (in addition to the
// methods required by providers.Interface) to allow callers to access those
// objects directly if needed, for debugging or logging purposes.
type Provider struct {
	configValue cty.Value

	// fakeNetDelay can be set to a nonzero duration in order to create a
	// fake delay in any provider method that would, for a real provider,
	// typically be expected to make a network request
	fakeNetDelay time.Duration

	// managedResources is used to simulate a stateful remote system in this
	// provider, by remembering objects that were previously created.
	//
	// The keys in this map are unique values generated internally within the
	// provider and are thus not predictable to the randomly-generated
	// stresstest configurations. However, if Terraform and the stresstest
	// harness are behaving correctly then these ids should persist in the
	// state throughout a test series and thus allow updating and destroying
	// existing objects on subsequent steps.
	//
	// Because this provider will potentially be called concurrently from
	// many separate graph nodes, we hold managedResourcesMutex whenever
	// accessing managedResources.
	managedResources      map[string]cty.Value
	managedResourcesMutex *sync.RWMutex
}

// New creates and returns a new instance of Provider, ready to use
// as a Terraform provider.
func New() *Provider {
	var mutex sync.RWMutex
	return &Provider{
		managedResources:      map[string]cty.Value{},
		managedResourcesMutex: &mutex,
	}
}

// NewInstance returns a new Provider which has the same fake network delay
// and shares the same repository of fake remote objects as the reciever,
// but that has its own independent provider configuration.
//
// If you intend to use SetFakeNetDelay, call it on the original instance
// of Provider before calling NewInstance, because subsequent calls to
// SetFakeNetDelay will not propagate to other already-existing instances.
// Only the mutable repository of objects is common to all instances created in
// this way.
func (p *Provider) NewInstance() *Provider {
	// We shallow-copy the reciever, which retains the same pointers to
	// managedResources and managedResourcesMutex and just snapshots
	// all of the non-pointer field values.
	ret := *p
	ret.configValue = cty.NilVal // each instance must be configured separately
	return &ret
}

// SetFakeNetDelay allows a caller to request that the provider introduce a
// fixed delay to each of its operations that would, in a real provider,
// typically result in a network request to a remote system.
//
// This is a pretty blunt instrument but can be useful for making race
// conditions in the concurrent graph walk easier to reproduce.
//
// Set the delay to the zero value of time.Duration to disable the fake
// delay, which is also the default behavior.
func (p *Provider) SetFakeNetDelay(delay time.Duration) {
	p.fakeNetDelay = delay
}

// CurrentObjects returns a map of the fake remote objects currently known
// to the provider.
//
// This is intended mainly just as additional context about what's going on
// to include in a debug log to help understand why a test failed.
func (p *Provider) CurrentObjects() map[string]cty.Value {
	// We create a copy of the map because that way the caller can potentially
	// hold on to it while they run other operations that would then mutate
	// the main map.
	p.managedResourcesMutex.RLock()
	ret := make(map[string]cty.Value, len(p.managedResources))
	for k, v := range p.managedResources {
		ret[k] = v
	}
	p.managedResourcesMutex.RUnlock()
	return ret
}

var _ providers.Interface = (*Provider)(nil)

// GetSchema implements providers.Interface.GetSchema.
func (p *Provider) GetSchema() providers.GetSchemaResponse {
	return providers.GetSchemaResponse{
		Provider: providerConfigSchema,
		ResourceTypes: map[string]providers.Schema{
			"stressful": ManagedResourceTypeSchema,
		},
		DataSources: map[string]providers.Schema{
			"stressful": DataResourceTypeSchema,
		},
	}
}

// PrepareProviderConfig implements providers.Interface.PrepareProviderConfig.
func (p *Provider) PrepareProviderConfig(req providers.PrepareProviderConfigRequest) providers.PrepareProviderConfigResponse {
	return providers.PrepareProviderConfigResponse{
		PreparedConfig: req.Config,
	}
}

// ValidateResourceTypeConfig implements providers.Interface.ValidateResourceTypeConfig.
func (p *Provider) ValidateResourceTypeConfig(providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
	// We currently need no validation other than what Terraform does for us
	// by enforcing our schema.
	return providers.ValidateResourceTypeConfigResponse{}
}

// ValidateDataSourceConfig implements providers.Interface.ValidateDataSourceConfig.
func (p *Provider) ValidateDataSourceConfig(providers.ValidateDataSourceConfigRequest) providers.ValidateDataSourceConfigResponse {
	// We currently need no validation other than what Terraform does for us
	// by enforcing our schema.
	return providers.ValidateDataSourceConfigResponse{}
}

// UpgradeResourceState implements providers.Interface.UpgradeResourceState.
func (p *Provider) UpgradeResourceState(req providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	// We don't do any actual upgrading as part of this provider, but this
	// function is also required to be able to translate the raw JSON
	// representation of state data for resource objects into a cty.Value,
	// so we do have to do a little work here.
	//
	// We assume that the state data will always be in JSON format here,
	// because we have no codepaths that would generate the legacy flatmap form.
	var diags tfdiags.Diagnostics
	ctyType := ManagedResourceTypeSchema.Block.ImpliedType()
	ctyVal, err := ctyjson.Unmarshal(req.RawStateJSON, ctyType)
	if err != nil {
		diags = diags.Append(err)
		return providers.UpgradeResourceStateResponse{
			Diagnostics: diags,
		}
	}
	return providers.UpgradeResourceStateResponse{
		UpgradedState: ctyVal,
		Diagnostics:   diags,
	}
}

// Configure implements providers.Interface.Configure.
func (p *Provider) Configure(req providers.ConfigureRequest) providers.ConfigureResponse {
	p.fakeNetRequest()
	p.configValue = req.Config.GetAttr("value")
	return providers.ConfigureResponse{}
}

// Stop implements providers.Interface.Stop, although it doesn't actually do
// anything because the stress test harness never cancels a graph walk.
func (p *Provider) Stop() error {
	// Nothing this provider does is really cancelable
	return nil
}

// ReadResource implements providers.Interface.ReadResource.
func (p *Provider) ReadResource(req providers.ReadResourceRequest) providers.ReadResourceResponse {
	p.fakeNetRequest()
	id := req.PriorState.GetAttr("id").AsString()
	p.managedResourcesMutex.RLock()
	obj := p.managedResources[id]
	p.managedResourcesMutex.RUnlock()
	if obj == cty.NilVal {
		// Providers signal an object being missing by returning a null value,
		// rather than by returning an error.
		obj = cty.NullVal(ManagedResourceTypeSchema.Block.ImpliedType())
	}
	return providers.ReadResourceResponse{
		NewState: obj,
	}
}

// PlanResourceChange implements providers.Interface.PlanResourceChange.
func (p *Provider) PlanResourceChange(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	p.fakeNetRequest()
	prior := req.PriorState
	proposed := req.ProposedNewState
	var planned cty.Value
	var reqReplace []cty.Path

	switch {
	case prior.IsNull():
		// For initial creation we just accept the user's provided name and
		// force_replace value, and signal the id and computed_name as being
		// unknown for now.
		planned = cty.ObjectVal(map[string]cty.Value{
			"id":            cty.UnknownVal(cty.String),
			"name":          proposed.GetAttr("name"),
			"force_replace": proposed.GetAttr("force_replace"),
			"computed_name": cty.UnknownVal(cty.String),
		})
	default:
		// For updates we just accept whatever the user proposed, but if
		// the "force_replace" value has changed then we'll signal that it
		// forces replacement. We don't need to do anything special with our
		// computed attributes in that case, because Terraform Core will
		// just call our PlanResourceChange a second time with req.PriorState
		// set to null, thus causing us to visit the other case above.
		planned = cty.ObjectVal(map[string]cty.Value{
			"id":            proposed.GetAttr("id"),
			"name":          proposed.GetAttr("name"),
			"force_replace": proposed.GetAttr("force_replace"),

			// computed_name is plannable on update, even though it's not plannable on create.
			// This is arbitrary and just to introduce another
			// potentially-interesting difference between update and replace.
			"computed_name": proposed.GetAttr("name"),
		})
		if !prior.GetAttr("force_replace").RawEquals(proposed.GetAttr("force_replace")) {
			reqReplace = append(reqReplace, cty.GetAttrPath("force_replace"))
		}
	}

	return providers.PlanResourceChangeResponse{
		PlannedState:    planned,
		RequiresReplace: reqReplace,
	}
}

// ApplyResourceChange implements providers.Interface.ApplyResourceChange.
func (p *Provider) ApplyResourceChange(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	p.fakeNetRequest()
	planned := req.PlannedState

	if planned.IsNull() {
		// This is a destroy action, then.
		idStr := req.PriorState.GetAttr("id").AsString()
		if _, exists := p.managedResources[idStr]; !exists {
			// Getting this far with a non-existent object suggests a race
			// condition or other bug in Terraform Core, because we should've
			// detected that the object no longer existed during ReadResource.
			return providers.ApplyResourceChangeResponse{
				Diagnostics: tfdiags.Diagnostics{
					tfdiags.Sourceless(
						tfdiags.Error,
						"Object not found",
						fmt.Sprintf("There is no active object with the id %q.", idStr),
					),
				},
			}
		}
		p.managedResourcesMutex.Lock()
		delete(p.managedResources, idStr)
		p.managedResourcesMutex.Unlock()
		return providers.ApplyResourceChangeResponse{
			NewState: planned, // a null value
		}
	}

	idVal := planned.GetAttr("id")
	if !idVal.IsKnown() {
		// On create, the id value will be unknown because we're
		// expected to generate it.
		idStr, err := uuid.GenerateUUID()
		if err != nil {
			// We should not typically end up in here, but could do if the
			// system's random number generator is broken in some way.
			var diags tfdiags.Diagnostics
			diags = diags.Append(err)
			return providers.ApplyResourceChangeResponse{
				Diagnostics: diags,
			}
		}
		idVal = cty.StringVal(idStr)
	}

	new := cty.ObjectVal(map[string]cty.Value{
		"id":            idVal,
		"name":          planned.GetAttr("name"),
		"force_replace": planned.GetAttr("force_replace"),

		// In our final object computed_name matches name, giving the
		// final value to replace the placeholder that we might've
		// written in during PlanResourceChange.
		"computed_name": planned.GetAttr("name"),
	})
	p.managedResourcesMutex.Lock()
	p.managedResources[idVal.AsString()] = new
	p.managedResourcesMutex.Unlock()

	return providers.ApplyResourceChangeResponse{
		NewState: new,
	}
}

// ImportResourceState implements providers.Interface.ImportResourceState by
// immediately returning an error, because stresstest doesn't currently
// exercise the import codepaths.
func (p *Provider) ImportResourceState(providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	return providers.ImportResourceStateResponse{
		Diagnostics: tfdiags.Diagnostics{
			tfdiags.Sourceless(
				tfdiags.Error,
				"Import not supported",
				"The stress provider doesn't support importing.",
			),
		},
	}
}

// ReadDataSource implements providers.Interface.ReadDataSource.
func (p *Provider) ReadDataSource(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	inVal := req.Config.GetAttr("in")
	providerVal := p.configValue
	if providerVal.IsNull() {
		// One of the compromises of the graph stresstest system is that all
		// of the values passed around are non-null strings, because it's
		// aiming to test the graph building/walking rather than expression
		// evaluation, so we'll prefer to return an empty string than a
		// null value for provider_value.
		providerVal = cty.StringVal("")
	}
	result := cty.ObjectVal(map[string]cty.Value{
		"in":             inVal,
		"out":            inVal,
		"provider_value": providerVal,
	})
	return providers.ReadDataSourceResponse{
		State: result,
	}
}

// Close implements providers.Interface.Close, by doing nothing at all because
// stress providers are in memory only.
func (p *Provider) Close() error {
	return nil
}

func (p *Provider) fakeNetRequest() {
	if p.fakeNetDelay != 0 {
		time.Sleep(p.fakeNetDelay)
	}
}

var providerConfigSchema = providers.Schema{
	Block: &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"value": {
				Type:        cty.String,
				Description: "An arbitrary value that we expose in one of the attributes of the data resource type, to help test that provider configuration happens before resource calls.",
				Optional:    true,
			},
		},
	},
}

// ManagedResourceTypeSchema is the schema of the "stressful" managed resource
// type, exported so that the stress testing code can easily decode state data.
var ManagedResourceTypeSchema = providers.Schema{
	Block: &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:        cty.String,
				Description: "A strong-random value chosen during creation and then unchanged in future updates, to help CheckState implemetations distinguish between an update and a replace operation.",
				Computed:    true,
			},
			"name": {
				Type:        cty.String,
				Description: "An arbitrary name for the object, which is mainly just here to give us something to assign random string values into. Gets copied into computed_name in the apply step when creating a new object.",
				Required:    true,
			},
			"force_replace": {
				Type:        cty.String,
				Description: "If planning detects that this argument has changed, the provider will call for the object to be replaced rather than updated.",
				Optional:    true,
			},
			"computed_name": {
				Type:        cty.String,
				Description: "Always the same value as name in the end, but shows as unknown during planning in order to allow exercising behaviors that arise only from unknown values.",
				Computed:    true,
			},
		},
	},
}

// DataResourceTypeSchema is the schema of the "stressful" data resource type,
// exported so that the stress testing code can easily decode state data.
var DataResourceTypeSchema = providers.Schema{
	Block: &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"out": {
				Type:     cty.String,
				Computed: true,
			},
			"in": {
				Type:     cty.String,
				Required: true,
			},
			"provider_value": {
				Type:        cty.String,
				Description: "Exports the value assigned to 'value' in the provider configuration, or an empty string if the provider had no such value.",
				Computed:    true,
			},
		},
	},
}
