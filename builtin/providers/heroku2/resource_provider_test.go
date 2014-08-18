package heroku

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
		"heroku": testAccProvider,
	}
}

func TestResourceProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = new(ResourceProvider)
}

func TestResourceProvider_Configure(t *testing.T) {
	rp := new(ResourceProvider)
	var expectedKey string
	var expectedEmail string

	if v := os.Getenv("HEROKU_EMAIL"); v != "" {
		expectedEmail = v
	} else {
		expectedEmail = "foo"
	}

	if v := os.Getenv("HEROKU_API_KEY"); v != "" {
		expectedKey = v
	} else {
		expectedKey = "foo"
	}

	raw := map[string]interface{}{
		"api_key": expectedKey,
		"email":   expectedEmail,
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
		APIKey: expectedKey,
		Email:  expectedEmail,
	}

	if !reflect.DeepEqual(rp.Config, expected) {
		t.Fatalf("bad: %#v", rp.Config)
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("HEROKU_EMAIL"); v == "" {
		t.Fatal("HEROKU_EMAIL must be set for acceptance tests")
	}

	if v := os.Getenv("HEROKU_API_KEY"); v == "" {
		t.Fatal("HEROKU_API_KEY must be set for acceptance tests")
	}
}
