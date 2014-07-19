package digitalocean

import (
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *ResourceProvider

func init() {
	testAccProvider = new(ResourceProvider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"digitalocean": testAccProvider,
	}
}

func TestResourceProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = new(ResourceProvider)
}

func TestResourceProvider_Configure(t *testing.T) {
	rp := new(ResourceProvider)

	raw := map[string]interface{}{
		"token": "foo",
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
		Token: "foo",
	}

	if !reflect.DeepEqual(rp.Config, expected) {
		t.Fatalf("bad: %#v", rp.Config)
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("DIGITALOCEAN_TOKEN"); v == "" {
		t.Fatal("DIGITALOCEAN_TOKEN must be set for acceptance tests")
	}
}
