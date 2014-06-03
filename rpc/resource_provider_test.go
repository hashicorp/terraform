package rpc

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvider_configure(t *testing.T) {
	p := new(terraform.MockResourceProvider)
	client, server := testClientServer(t)
	name, err := Register(server, p)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	provider := &ResourceProvider{Client: client, Name: name}

	// Configure
	config := map[string]interface{}{"foo": "bar"}
	w, e := provider.Configure(config)
	if !p.ConfigureCalled {
		t.Fatal("configure should be called")
	}
	if !reflect.DeepEqual(p.ConfigureConfig, config) {
		t.Fatalf("bad: %#v", p.ConfigureConfig)
	}
	if w != nil {
		t.Fatalf("bad: %#v", w)
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
	config := map[string]interface{}{"foo": "bar"}
	w, e := provider.Configure(config)
	if !p.ConfigureCalled {
		t.Fatal("configure should be called")
	}
	if !reflect.DeepEqual(p.ConfigureConfig, config) {
		t.Fatalf("bad: %#v", p.ConfigureConfig)
	}
	if w != nil {
		t.Fatalf("bad: %#v", w)
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

	p.ConfigureReturnWarnings = []string{"foo", "bar"}

	// Configure
	config := map[string]interface{}{"foo": "bar"}
	w, e := provider.Configure(config)
	if !p.ConfigureCalled {
		t.Fatal("configure should be called")
	}
	if !reflect.DeepEqual(p.ConfigureConfig, config) {
		t.Fatalf("bad: %#v", p.ConfigureConfig)
	}
	if w == nil {
		t.Fatal("should have warnings")
	}
	if !reflect.DeepEqual(w, []string{"foo", "bar"}) {
		t.Fatalf("bad: %#v", w)
	}
	if e != nil {
		t.Fatalf("bad: %#v", e)
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
