package consul

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *ResourceProvider

func init() {
	testAccProvider = new(ResourceProvider)
	testAccProvider.Config.Address = "demo.consul.io:80"
	testAccProviders = map[string]terraform.ResourceProvider{
		"consul": testAccProvider,
	}
}

func TestResourceProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = new(ResourceProvider)
}

func TestResourceProvider_Configure(t *testing.T) {
	rp := new(ResourceProvider)

	raw := map[string]interface{}{
		"address":    "demo.consul.io:80",
		"datacenter": "nyc1",
	}

	rawConfig, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = rp.Configure(terraform.NewResourceConfig(rawConfig))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := Config{
		Address:    "demo.consul.io:80",
		Datacenter: "nyc1",
	}

	if !reflect.DeepEqual(rp.Config, expected) {
		t.Fatalf("bad: %#v", rp.Config)
	}
}
