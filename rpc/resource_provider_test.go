package rpc

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = new(ResourceProvider)
}

func TestResourceProvider_configure(t *testing.T) {
	p := new(terraform.MockResourceProvider)
	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

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
	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

	p.ConfigureReturnError = errors.New("foo")

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
	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

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
	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

	p.ApplyReturn = &terraform.ResourceState{
		ID: "bob",
	}

	// Apply
	state := &terraform.ResourceState{}
	diff := &terraform.ResourceDiff{}
	newState, err := provider.Apply(state, diff)
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
	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

	p.DiffReturn = &terraform.ResourceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				Old: "",
				New: "bar",
			},
		},
	}

	// Diff
	state := &terraform.ResourceState{}
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	diff, err := provider.Diff(state, config)
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
	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

	p.DiffReturnError = errors.New("foo")

	// Diff
	state := &terraform.ResourceState{}
	config := &terraform.ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	diff, err := provider.Diff(state, config)
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
	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

	p.RefreshReturn = &terraform.ResourceState{
		ID: "bob",
	}

	// Refresh
	state := &terraform.ResourceState{}
	newState, err := provider.Refresh(state)
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

func TestResourceProvider_resources(t *testing.T) {
	p := new(terraform.MockResourceProvider)
	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

	expected := []terraform.ResourceType{
		{"foo"},
		{"bar"},
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

func TestResourceProvider_validate(t *testing.T) {
	p := new(terraform.MockResourceProvider)
	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

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
	p.ValidateReturnErrors = []error{errors.New("foo")}

	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

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
	p.ValidateReturnWarns = []string{"foo"}

	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

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
	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

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
	p.ValidateResourceReturnErrors = []error{errors.New("foo")}

	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

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
	p.ValidateResourceReturnWarns = []string{"foo"}

	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

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
