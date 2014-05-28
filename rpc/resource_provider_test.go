package rpc

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvider_configure(t *testing.T) {
	p := new(terraform.MockResourceProvider)
	client, server := testClientServer(t)
	server.RegisterName("ResourceProvider", &ResourceProviderServer{
		Provider: p,
	})

	provider := &ResourceProvider{Client: client}

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
