package plugin

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvider_impl(t *testing.T) {
	var _ plugin.Plugin = new(ResourceProviderPlugin)
	var _ terraform.ResourceProvider = new(ResourceProvider)
}

func TestResourceProvider_stop(t *testing.T) {
	// Create a mock provider
	p := new(terraform.MockResourceProvider)
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	// Stop
	e := provider.Stop()
	if !p.StopCalled {
		t.Fatal("stop should be called")
	}
	if e != nil {
		t.Fatalf("bad: %#v", e)
	}
}

func TestResourceProvider_stopErrors(t *testing.T) {
	p := new(terraform.MockResourceProvider)
	p.StopReturnError = errors.New("foo")

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	// Stop
	e := provider.Stop()
	if !p.StopCalled {
		t.Fatal("stop should be called")
	}
	if e == nil {
		t.Fatal("should have error")
	}
	if e.Error() != "foo" {
		t.Fatalf("bad: %s", e)
	}
}

func TestResourceProvider_input(t *testing.T) {
	// Create a mock provider
	p := new(terraform.MockResourceProvider)
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	input := new(terraform.MockUIInput)

	expected := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"bar": "baz"},
	}
	p.InputReturnConfig = expected

	// Input
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	actual, err := provider.Input(input, config)
	if !p.InputCalled {
		t.Fatal("input should be called")
	}
	if !reflect.DeepEqual(p.InputConfig, config) {
		t.Fatalf("bad: %#v", p.InputConfig)
	}
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceProvider_configure(t *testing.T) {
	// Create a mock provider
	p := new(terraform.MockResourceProvider)
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	// Configure
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	e := provider.Configure(config)
	if !p.ConfigureCalled {
		t.Fatal("configure should be called")
	}
	if !reflect.DeepEqual(p.ConfigureConfig, config) {
		t.Fatalf("bad: %#v", p.ConfigureConfig)
	}
	if e != nil {
		t.Fatalf("bad: %#v", e)
	}
}

func TestResourceProvider_configure_errors(t *testing.T) {
	p := new(terraform.MockResourceProvider)
	p.ConfigureReturnError = errors.New("foo")

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	// Configure
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	e := provider.Configure(config)
	if !p.ConfigureCalled {
		t.Fatal("configure should be called")
	}
	if !reflect.DeepEqual(p.ConfigureConfig, config) {
		t.Fatalf("bad: %#v", p.ConfigureConfig)
	}
	if e == nil {
		t.Fatal("should have error")
	}
	if e.Error() != "foo" {
		t.Fatalf("bad: %s", e)
	}
}

func TestResourceProvider_configure_warnings(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	// Configure
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	e := provider.Configure(config)
	if !p.ConfigureCalled {
		t.Fatal("configure should be called")
	}
	if !reflect.DeepEqual(p.ConfigureConfig, config) {
		t.Fatalf("bad: %#v", p.ConfigureConfig)
	}
	if e != nil {
		t.Fatalf("bad: %#v", e)
	}
}

func TestResourceProvider_apply(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	p.ApplyReturn = &terraform.InstanceState{
		ID: "bob",
	}

	// Apply
	info := &terraform.InstanceInfo{}
	state := &terraform.InstanceState{}
	diff := &terraform.InstanceDiff{}
	newState, err := provider.Apply(info, state, diff)
	if !p.ApplyCalled {
		t.Fatal("apply should be called")
	}
	if !reflect.DeepEqual(p.ApplyDiff, diff) {
		t.Fatalf("bad: %#v", p.ApplyDiff)
	}
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}
	if !reflect.DeepEqual(p.ApplyReturn, newState) {
		t.Fatalf("bad: %#v", newState)
	}
}

func TestResourceProvider_diff(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	p.DiffReturn = &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				Old: "",
				New: "bar",
			},
		},
	}

	// Diff
	info := &terraform.InstanceInfo{}
	state := &terraform.InstanceState{}
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	diff, err := provider.Diff(info, state, config)
	if !p.DiffCalled {
		t.Fatal("diff should be called")
	}
	if !reflect.DeepEqual(p.DiffDesired, config) {
		t.Fatalf("bad: %#v", p.DiffDesired)
	}
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}
	if !reflect.DeepEqual(p.DiffReturn, diff) {
		t.Fatalf("bad: %#v", diff)
	}
}

func TestResourceProvider_diff_error(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	p.DiffReturnError = errors.New("foo")

	// Diff
	info := &terraform.InstanceInfo{}
	state := &terraform.InstanceState{}
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	diff, err := provider.Diff(info, state, config)
	if !p.DiffCalled {
		t.Fatal("diff should be called")
	}
	if !reflect.DeepEqual(p.DiffDesired, config) {
		t.Fatalf("bad: %#v", p.DiffDesired)
	}
	if err == nil {
		t.Fatal("should have error")
	}
	if diff != nil {
		t.Fatal("should not have diff")
	}
}

func TestResourceProvider_refresh(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	p.RefreshReturn = &terraform.InstanceState{
		ID: "bob",
	}

	// Refresh
	info := &terraform.InstanceInfo{}
	state := &terraform.InstanceState{}
	newState, err := provider.Refresh(info, state)
	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}
	if !reflect.DeepEqual(p.RefreshState, state) {
		t.Fatalf("bad: %#v", p.RefreshState)
	}
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}
	if !reflect.DeepEqual(p.RefreshReturn, newState) {
		t.Fatalf("bad: %#v", newState)
	}
}

func TestResourceProvider_importState(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "bob",
		},
	}

	// ImportState
	info := &terraform.InstanceInfo{}
	states, err := provider.ImportState(info, "foo")
	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
	}
	if !reflect.DeepEqual(p.ImportStateInfo, info) {
		t.Fatalf("bad: %#v", p.ImportStateInfo)
	}
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}
	if !reflect.DeepEqual(p.ImportStateReturn, states) {
		t.Fatalf("bad: %#v", states)
	}
}

func TestResourceProvider_resources(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	expected := []terraform.ResourceType{
		terraform.ResourceType{Name: "foo"},
		terraform.ResourceType{Name: "bar", Importable: true},
	}

	p.ResourcesReturn = expected

	// Resources
	result := provider.Resources()
	if !p.ResourcesCalled {
		t.Fatal("resources should be called")
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestResourceProvider_readdataapply(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	p.ReadDataApplyReturn = &terraform.InstanceState{
		ID: "bob",
	}

	// ReadDataApply
	info := &terraform.InstanceInfo{}
	diff := &terraform.InstanceDiff{}
	newState, err := provider.ReadDataApply(info, diff)
	if !p.ReadDataApplyCalled {
		t.Fatal("ReadDataApply should be called")
	}
	if !reflect.DeepEqual(p.ReadDataApplyDiff, diff) {
		t.Fatalf("bad: %#v", p.ReadDataApplyDiff)
	}
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}
	if !reflect.DeepEqual(p.ReadDataApplyReturn, newState) {
		t.Fatalf("bad: %#v", newState)
	}
}

func TestResourceProvider_datasources(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	expected := []terraform.DataSource{
		{Name: "foo"},
		{Name: "bar"},
	}

	p.DataSourcesReturn = expected

	// DataSources
	result := provider.DataSources()
	if !p.DataSourcesCalled {
		t.Fatal("DataSources should be called")
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestResourceProvider_validate(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	// Configure
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	w, e := provider.Validate(config)
	if !p.ValidateCalled {
		t.Fatal("configure should be called")
	}
	if !reflect.DeepEqual(p.ValidateConfig, config) {
		t.Fatalf("bad: %#v", p.ValidateConfig)
	}
	if w != nil {
		t.Fatalf("bad: %#v", w)
	}
	if e != nil {
		t.Fatalf("bad: %#v", e)
	}
}

func TestResourceProvider_validate_errors(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	p.ValidateReturnErrors = []error{errors.New("foo")}

	// Configure
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	w, e := provider.Validate(config)
	if !p.ValidateCalled {
		t.Fatal("configure should be called")
	}
	if !reflect.DeepEqual(p.ValidateConfig, config) {
		t.Fatalf("bad: %#v", p.ValidateConfig)
	}
	if w != nil {
		t.Fatalf("bad: %#v", w)
	}

	if len(e) != 1 {
		t.Fatalf("bad: %#v", e)
	}
	if e[0].Error() != "foo" {
		t.Fatalf("bad: %#v", e)
	}
}

func TestResourceProvider_validate_warns(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	p.ValidateReturnWarns = []string{"foo"}

	// Configure
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	w, e := provider.Validate(config)
	if !p.ValidateCalled {
		t.Fatal("configure should be called")
	}
	if !reflect.DeepEqual(p.ValidateConfig, config) {
		t.Fatalf("bad: %#v", p.ValidateConfig)
	}
	if e != nil {
		t.Fatalf("bad: %#v", e)
	}

	expected := []string{"foo"}
	if !reflect.DeepEqual(w, expected) {
		t.Fatalf("bad: %#v", w)
	}
}

func TestResourceProvider_validateResource(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	// Configure
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	w, e := provider.ValidateResource("foo", config)
	if !p.ValidateResourceCalled {
		t.Fatal("configure should be called")
	}
	if p.ValidateResourceType != "foo" {
		t.Fatalf("bad: %#v", p.ValidateResourceType)
	}
	if !reflect.DeepEqual(p.ValidateResourceConfig, config) {
		t.Fatalf("bad: %#v", p.ValidateResourceConfig)
	}
	if w != nil {
		t.Fatalf("bad: %#v", w)
	}
	if e != nil {
		t.Fatalf("bad: %#v", e)
	}
}

func TestResourceProvider_validateResource_errors(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	p.ValidateResourceReturnErrors = []error{errors.New("foo")}

	// Configure
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	w, e := provider.ValidateResource("foo", config)
	if !p.ValidateResourceCalled {
		t.Fatal("configure should be called")
	}
	if p.ValidateResourceType != "foo" {
		t.Fatalf("bad: %#v", p.ValidateResourceType)
	}
	if !reflect.DeepEqual(p.ValidateResourceConfig, config) {
		t.Fatalf("bad: %#v", p.ValidateResourceConfig)
	}
	if w != nil {
		t.Fatalf("bad: %#v", w)
	}

	if len(e) != 1 {
		t.Fatalf("bad: %#v", e)
	}
	if e[0].Error() != "foo" {
		t.Fatalf("bad: %#v", e)
	}
}

func TestResourceProvider_validateResource_warns(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	p.ValidateResourceReturnWarns = []string{"foo"}

	// Configure
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	w, e := provider.ValidateResource("foo", config)
	if !p.ValidateResourceCalled {
		t.Fatal("configure should be called")
	}
	if p.ValidateResourceType != "foo" {
		t.Fatalf("bad: %#v", p.ValidateResourceType)
	}
	if !reflect.DeepEqual(p.ValidateResourceConfig, config) {
		t.Fatalf("bad: %#v", p.ValidateResourceConfig)
	}
	if e != nil {
		t.Fatalf("bad: %#v", e)
	}

	expected := []string{"foo"}
	if !reflect.DeepEqual(w, expected) {
		t.Fatalf("bad: %#v", w)
	}
}

func TestResourceProvider_validateDataSource(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	// Configure
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	w, e := provider.ValidateDataSource("foo", config)
	if !p.ValidateDataSourceCalled {
		t.Fatal("configure should be called")
	}
	if p.ValidateDataSourceType != "foo" {
		t.Fatalf("bad: %#v", p.ValidateDataSourceType)
	}
	if !reflect.DeepEqual(p.ValidateDataSourceConfig, config) {
		t.Fatalf("bad: %#v", p.ValidateDataSourceConfig)
	}
	if w != nil {
		t.Fatalf("bad: %#v", w)
	}
	if e != nil {
		t.Fatalf("bad: %#v", e)
	}
}

func TestResourceProvider_close(t *testing.T) {
	p := new(terraform.MockResourceProvider)

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, pluginMap(&ServeOpts{
		ProviderFunc: testProviderFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProviderPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvider)

	var iface interface{} = provider
	pCloser, ok := iface.(terraform.ResourceProviderCloser)
	if !ok {
		t.Fatal("should be a ResourceProviderCloser")
	}

	if err := pCloser.Close(); err != nil {
		t.Fatalf("failed to close provider: %s", err)
	}

	// The connection should be closed now, so if we to make a
	// new call we should get an error.
	err = provider.Configure(&terraform.ResourceConfig{})
	if err == nil {
		t.Fatal("should have error")
	}
}
