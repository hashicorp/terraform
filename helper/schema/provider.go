package schema

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider represents a resource provider in Terraform, and properly
// implements all of the ResourceProvider API.
//
// By defining a schema for the configuration of the provider, the
// map of supporting resources, and a configuration function, the schema
// framework takes over and handles all the provider operations for you.
//
// After defining the provider structure, it is unlikely that you'll require any
// of the methods on Provider itself.
type Provider struct {
	// Schema is the schema for the configuration of this provider. If this
	// provider has no configuration, this can be omitted.
	//
	// The keys of this map are the configuration keys, and the value is
	// the schema describing the value of the configuration.
	Schema map[string]*Schema

	// ResourcesMap is the list of available resources that this provider
	// can manage, along with their Resource structure defining their
	// own schemas and CRUD operations.
	//
	// Provider automatically handles routing operations such as Apply,
	// Diff, etc. to the proper resource.
	ResourcesMap map[string]*Resource

	// DataSourcesMap is the collection of available data sources that
	// this provider implements, with a Resource instance defining
	// the schema and Read operation of each.
	//
	// Resource instances for data sources must have a Read function
	// and must *not* implement Create, Update or Delete.
	DataSourcesMap map[string]*Resource

	// ConfigureFunc is a function for configuring the provider. If the
	// provider doesn't need to be configured, this can be omitted.
	//
	// See the ConfigureFunc documentation for more information.
	ConfigureFunc ConfigureFunc

	// MetaReset is called by TestReset to reset any state stored in the meta
	// interface.  This is especially important if the StopContext is stored by
	// the provider.
	MetaReset func() error

	meta interface{}

	// a mutex is required because TestReset can directly replace the stopCtx
	stopMu        sync.Mutex
	stopCtx       context.Context
	stopCtxCancel context.CancelFunc
	stopOnce      sync.Once

	TerraformVersion string
}

// ConfigureFunc is the function used to configure a Provider.
//
// The interface{} value returned by this function is stored and passed into
// the subsequent resources as the meta parameter. This return value is
// usually used to pass along a configured API client, a configuration
// structure, etc.
type ConfigureFunc func(*ResourceData) (interface{}, error)

// InternalValidate should be called to validate the structure
// of the provider.
//
// This should be called in a unit test for any provider to verify
// before release that a provider is properly configured for use with
// this library.
func (p *Provider) InternalValidate() error {
	if p == nil {
		return errors.New("provider is nil")
	}

	var validationErrors error
	sm := schemaMap(p.Schema)
	if err := sm.InternalValidate(sm); err != nil {
		validationErrors = multierror.Append(validationErrors, err)
	}

	// Provider-specific checks
	for k, _ := range sm {
		if isReservedProviderFieldName(k) {
			return fmt.Errorf("%s is a reserved field name for a provider", k)
		}
	}

	for k, r := range p.ResourcesMap {
		if err := r.InternalValidate(nil, true); err != nil {
			validationErrors = multierror.Append(validationErrors, fmt.Errorf("resource %s: %s", k, err))
		}
	}

	for k, r := range p.DataSourcesMap {
		if err := r.InternalValidate(nil, false); err != nil {
			validationErrors = multierror.Append(validationErrors, fmt.Errorf("data source %s: %s", k, err))
		}
	}

	return validationErrors
}

func isReservedProviderFieldName(name string) bool {
	for _, reservedName := range config.ReservedProviderFields {
		if name == reservedName {
			return true
		}
	}
	return false
}

// Meta returns the metadata associated with this provider that was
// returned by the Configure call. It will be nil until Configure is called.
func (p *Provider) Meta() interface{} {
	return p.meta
}

// SetMeta can be used to forcefully set the Meta object of the provider.
// Note that if Configure is called the return value will override anything
// set here.
func (p *Provider) SetMeta(v interface{}) {
	p.meta = v
}

// Stopped reports whether the provider has been stopped or not.
func (p *Provider) Stopped() bool {
	ctx := p.StopContext()
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// StopCh returns a channel that is closed once the provider is stopped.
func (p *Provider) StopContext() context.Context {
	p.stopOnce.Do(p.stopInit)

	p.stopMu.Lock()
	defer p.stopMu.Unlock()

	return p.stopCtx
}

func (p *Provider) stopInit() {
	p.stopMu.Lock()
	defer p.stopMu.Unlock()

	p.stopCtx, p.stopCtxCancel = context.WithCancel(context.Background())
}

// Stop implementation of terraform.ResourceProvider interface.
func (p *Provider) Stop() error {
	p.stopOnce.Do(p.stopInit)

	p.stopMu.Lock()
	defer p.stopMu.Unlock()

	p.stopCtxCancel()
	return nil
}

// TestReset resets any state stored in the Provider, and will call TestReset
// on Meta if it implements the TestProvider interface.
// This may be used to reset the schema.Provider at the start of a test, and is
// automatically called by resource.Test.
func (p *Provider) TestReset() error {
	p.stopInit()
	if p.MetaReset != nil {
		return p.MetaReset()
	}
	return nil
}

// GetSchema implementation of terraform.ResourceProvider interface
func (p *Provider) GetSchema(req *terraform.ProviderSchemaRequest) (*terraform.ProviderSchema, error) {
	resourceTypes := map[string]*configschema.Block{}
	dataSources := map[string]*configschema.Block{}

	for _, name := range req.ResourceTypes {
		if r, exists := p.ResourcesMap[name]; exists {
			resourceTypes[name] = r.CoreConfigSchema()
		}
	}
	for _, name := range req.DataSources {
		if r, exists := p.DataSourcesMap[name]; exists {
			dataSources[name] = r.CoreConfigSchema()
		}
	}

	return &terraform.ProviderSchema{
		Provider:      schemaMap(p.Schema).CoreConfigSchema(),
		ResourceTypes: resourceTypes,
		DataSources:   dataSources,
	}, nil
}

// Input implementation of terraform.ResourceProvider interface.
func (p *Provider) Input(
	input terraform.UIInput,
	c *terraform.ResourceConfig) (*terraform.ResourceConfig, error) {
	return schemaMap(p.Schema).Input(input, c)
}

// Validate implementation of terraform.ResourceProvider interface.
func (p *Provider) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	if err := p.InternalValidate(); err != nil {
		return nil, []error{fmt.Errorf(
			"Internal validation of the provider failed! This is always a bug\n"+
				"with the provider itself, and not a user issue. Please report\n"+
				"this bug:\n\n%s", err)}
	}

	return schemaMap(p.Schema).Validate(c)
}

// ValidateResource implementation of terraform.ResourceProvider interface.
func (p *Provider) ValidateResource(
	t string, c *terraform.ResourceConfig) ([]string, []error) {
	r, ok := p.ResourcesMap[t]
	if !ok {
		return nil, []error{fmt.Errorf(
			"Provider doesn't support resource: %s", t)}
	}

	return r.Validate(c)
}

// Configure implementation of terraform.ResourceProvider interface.
func (p *Provider) Configure(c *terraform.ResourceConfig) error {
	// No configuration
	if p.ConfigureFunc == nil {
		return nil
	}

	sm := schemaMap(p.Schema)

	// Get a ResourceData for this configuration. To do this, we actually
	// generate an intermediary "diff" although that is never exposed.
	diff, err := sm.Diff(nil, c, nil, p.meta, true)
	if err != nil {
		return err
	}

	data, err := sm.Data(nil, diff)
	if err != nil {
		return err
	}

	meta, err := p.ConfigureFunc(data)
	if err != nil {
		return err
	}

	p.meta = meta
	return nil
}

// Apply implementation of terraform.ResourceProvider interface.
func (p *Provider) Apply(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	d *terraform.InstanceDiff) (*terraform.InstanceState, error) {
	r, ok := p.ResourcesMap[info.Type]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", info.Type)
	}

	return r.Apply(s, d, p.meta)
}

// Diff implementation of terraform.ResourceProvider interface.
func (p *Provider) Diff(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
	r, ok := p.ResourcesMap[info.Type]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", info.Type)
	}

	return r.Diff(s, c, p.meta)
}

// SimpleDiff is used by the new protocol wrappers to get a diff that doesn't
// attempt to calculate ignore_changes.
func (p *Provider) SimpleDiff(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
	r, ok := p.ResourcesMap[info.Type]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", info.Type)
	}

	return r.simpleDiff(s, c, p.meta)
}

// Refresh implementation of terraform.ResourceProvider interface.
func (p *Provider) Refresh(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState) (*terraform.InstanceState, error) {
	r, ok := p.ResourcesMap[info.Type]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", info.Type)
	}

	return r.Refresh(s, p.meta)
}

// Resources implementation of terraform.ResourceProvider interface.
func (p *Provider) Resources() []terraform.ResourceType {
	keys := make([]string, 0, len(p.ResourcesMap))
	for k := range p.ResourcesMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]terraform.ResourceType, 0, len(keys))
	for _, k := range keys {
		resource := p.ResourcesMap[k]

		// This isn't really possible (it'd fail InternalValidate), but
		// we do it anyways to avoid a panic.
		if resource == nil {
			resource = &Resource{}
		}

		result = append(result, terraform.ResourceType{
			Name:       k,
			Importable: resource.Importer != nil,

			// Indicates that a provider is compiled against a new enough
			// version of core to support the GetSchema method.
			SchemaAvailable: true,
		})
	}

	return result
}

func (p *Provider) ImportState(
	info *terraform.InstanceInfo,
	id string) ([]*terraform.InstanceState, error) {
	// Find the resource
	r, ok := p.ResourcesMap[info.Type]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", info.Type)
	}

	// If it doesn't support import, error
	if r.Importer == nil {
		return nil, fmt.Errorf("resource %s doesn't support import", info.Type)
	}

	// Create the data
	data := r.Data(nil)
	data.SetId(id)
	data.SetType(info.Type)

	// Call the import function
	results := []*ResourceData{data}
	if r.Importer.State != nil {
		var err error
		results, err = r.Importer.State(data, p.meta)
		if err != nil {
			return nil, err
		}
	}

	// Convert the results to InstanceState values and return it
	states := make([]*terraform.InstanceState, len(results))
	for i, r := range results {
		states[i] = r.State()
	}

	// Verify that all are non-nil. If there are any nil the error
	// isn't obvious so we circumvent that with a friendlier error.
	for _, s := range states {
		if s == nil {
			return nil, fmt.Errorf(
				"nil entry in ImportState results. This is always a bug with\n" +
					"the resource that is being imported. Please report this as\n" +
					"a bug to Terraform.")
		}
	}

	return states, nil
}

// ValidateDataSource implementation of terraform.ResourceProvider interface.
func (p *Provider) ValidateDataSource(
	t string, c *terraform.ResourceConfig) ([]string, []error) {
	r, ok := p.DataSourcesMap[t]
	if !ok {
		return nil, []error{fmt.Errorf(
			"Provider doesn't support data source: %s", t)}
	}

	return r.Validate(c)
}

// ReadDataDiff implementation of terraform.ResourceProvider interface.
func (p *Provider) ReadDataDiff(
	info *terraform.InstanceInfo,
	c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {

	r, ok := p.DataSourcesMap[info.Type]
	if !ok {
		return nil, fmt.Errorf("unknown data source: %s", info.Type)
	}

	return r.Diff(nil, c, p.meta)
}

// RefreshData implementation of terraform.ResourceProvider interface.
func (p *Provider) ReadDataApply(
	info *terraform.InstanceInfo,
	d *terraform.InstanceDiff) (*terraform.InstanceState, error) {

	r, ok := p.DataSourcesMap[info.Type]
	if !ok {
		return nil, fmt.Errorf("unknown data source: %s", info.Type)
	}

	return r.ReadDataApply(d, p.meta)
}

// DataSources implementation of terraform.ResourceProvider interface.
func (p *Provider) DataSources() []terraform.DataSource {
	keys := make([]string, 0, len(p.DataSourcesMap))
	for k, _ := range p.DataSourcesMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]terraform.DataSource, 0, len(keys))
	for _, k := range keys {
		result = append(result, terraform.DataSource{
			Name: k,

			// Indicates that a provider is compiled against a new enough
			// version of core to support the GetSchema method.
			SchemaAvailable: true,
		})
	}

	return result
}
