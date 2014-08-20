package aws

import (
	"log"
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
		"aws": testAccProvider,
	}
}

func TestResourceProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = new(ResourceProvider)
}

func TestResourceProvider_Configure(t *testing.T) {
	rp := new(ResourceProvider)

	raw := map[string]interface{}{
		"access_key": "foo",
		"secret_key": "bar",
		"region":     "us-east-1",
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
		AccessKey: "foo",
		SecretKey: "bar",
		Region:    "us-east-1",
	}

	if !reflect.DeepEqual(rp.Config, expected) {
		t.Fatalf("bad: %#v", rp.Config)
	}

	if rp.p == nil {
		t.Fatal("provider should be set")
	}
	if !reflect.DeepEqual(rp, rp.p.Meta()) {
		t.Fatalf("meta should be set")
	}
}

func TestResourceProvider_ConfigureBadRegion(t *testing.T) {
	rp := new(ResourceProvider)

	raw := map[string]interface{}{
		"access_key": "foo",
		"secret_key": "bar",
		"region":     "blah",
	}

	rawConfig, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = rp.Configure(terraform.NewResourceConfig(rawConfig))
	if err == nil {
		t.Fatalf("should have err: bad region")
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("AWS_ACCESS_KEY"); v == "" {
		t.Fatal("AWS_ACCESS_KEY must be set for acceptance tests")
	}
	if v := os.Getenv("AWS_SECRET_KEY"); v == "" {
		t.Fatal("AWS_SECRET_KEY must be set for acceptance tests")
	}
	if v := os.Getenv("AWS_REGION"); v == "" {
		log.Println("[INFO] Test: Using us-west-2 as test region")
		os.Setenv("AWS_REGION", "us-west-2")
	}
}
