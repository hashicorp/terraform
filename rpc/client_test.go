package rpc

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestClient_ResourceProvider(t *testing.T) {
	clientConn, serverConn := testConn(t)

	p := new(terraform.MockResourceProvider)
	server := &Server{ProviderFunc: testProviderFixed(p)}
	go server.ServeConn(serverConn)

	client, err := NewClient(clientConn)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer client.Close()

	provider, err := client.ResourceProvider()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

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

func TestClient_ResourceProvisioner(t *testing.T) {
	clientConn, serverConn := testConn(t)

	p := new(terraform.MockResourceProvisioner)
	server := &Server{ProvisionerFunc: testProvisionerFixed(p)}
	go server.ServeConn(serverConn)

	client, err := NewClient(clientConn)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer client.Close()

	provisioner, err := client.ResourceProvisioner()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

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
