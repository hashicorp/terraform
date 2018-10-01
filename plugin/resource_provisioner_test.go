package plugin

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvisioner_impl(t *testing.T) {
	var _ plugin.Plugin = new(ResourceProvisionerPlugin)
	var _ terraform.ResourceProvisioner = new(ResourceProvisioner)
}

func TestResourceProvisioner_stop(t *testing.T) {
	// Create a mock provider
	p := new(terraform.MockResourceProvisioner)
	client, _ := plugin.TestPluginRPCConn(t, legacyPluginMap(&ServeOpts{
		ProvisionerFunc: testProvisionerFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProvisionerPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvisioner)

	// Stop
	e := provider.Stop()
	if !p.StopCalled {
		t.Fatal("stop should be called")
	}
	if e != nil {
		t.Fatalf("bad: %#v", e)
	}
}

func TestResourceProvisioner_stopErrors(t *testing.T) {
	p := new(terraform.MockResourceProvisioner)
	p.StopReturnError = errors.New("foo")

	// Create a mock provider
	client, _ := plugin.TestPluginRPCConn(t, legacyPluginMap(&ServeOpts{
		ProvisionerFunc: testProvisionerFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProvisionerPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := raw.(terraform.ResourceProvisioner)

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

func TestResourceProvisioner_apply(t *testing.T) {
	// Create a mock provider
	p := new(terraform.MockResourceProvisioner)
	client, _ := plugin.TestPluginRPCConn(t, legacyPluginMap(&ServeOpts{
		ProvisionerFunc: testProvisionerFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProvisionerPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provisioner := raw.(terraform.ResourceProvisioner)

	// Apply
	output := &terraform.MockUIOutput{}
	state := &terraform.InstanceState{}
	conf := &terraform.ResourceConfig{}
	err = provisioner.Apply(output, state, conf)
	if !p.ApplyCalled {
		t.Fatal("apply should be called")
	}
	if !reflect.DeepEqual(p.ApplyConfig, conf) {
		t.Fatalf("bad: %#v", p.ApplyConfig)
	}
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}
}

func TestResourceProvisioner_validate(t *testing.T) {
	// Create a mock provider
	p := new(terraform.MockResourceProvisioner)
	client, _ := plugin.TestPluginRPCConn(t, legacyPluginMap(&ServeOpts{
		ProvisionerFunc: testProvisionerFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProvisionerPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provisioner := raw.(terraform.ResourceProvisioner)

	// Configure
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	w, e := provisioner.Validate(config)
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

func TestResourceProvisioner_validate_errors(t *testing.T) {
	// Create a mock provider
	p := new(terraform.MockResourceProvisioner)
	client, _ := plugin.TestPluginRPCConn(t, legacyPluginMap(&ServeOpts{
		ProvisionerFunc: testProvisionerFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProvisionerPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provisioner := raw.(terraform.ResourceProvisioner)

	p.ValidateReturnErrors = []error{errors.New("foo")}

	// Configure
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	w, e := provisioner.Validate(config)
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

func TestResourceProvisioner_validate_warns(t *testing.T) {
	// Create a mock provider
	p := new(terraform.MockResourceProvisioner)
	client, _ := plugin.TestPluginRPCConn(t, legacyPluginMap(&ServeOpts{
		ProvisionerFunc: testProvisionerFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProvisionerPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provisioner := raw.(terraform.ResourceProvisioner)

	p.ValidateReturnWarns = []string{"foo"}

	// Configure
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	w, e := provisioner.Validate(config)
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

func TestResourceProvisioner_close(t *testing.T) {
	// Create a mock provider
	p := new(terraform.MockResourceProvisioner)
	client, _ := plugin.TestPluginRPCConn(t, legacyPluginMap(&ServeOpts{
		ProvisionerFunc: testProvisionerFixed(p),
	}), nil)
	defer client.Close()

	// Request the provider
	raw, err := client.Dispense(ProvisionerPluginName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provisioner := raw.(terraform.ResourceProvisioner)

	pCloser, ok := raw.(terraform.ResourceProvisionerCloser)
	if !ok {
		t.Fatal("should be a ResourceProvisionerCloser")
	}

	if err := pCloser.Close(); err != nil {
		t.Fatalf("failed to close provisioner: %s", err)
	}

	// The connection should be closed now, so if we to make a
	// new call we should get an error.
	o := &terraform.MockUIOutput{}
	s := &terraform.InstanceState{}
	c := &terraform.ResourceConfig{}
	err = provisioner.Apply(o, s, c)
	if err == nil {
		t.Fatal("should have error")
	}
}
