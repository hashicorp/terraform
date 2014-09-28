package heroku

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"heroku": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func TestProviderConfigure(t *testing.T) {
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

	rp := Provider()
	err = rp.Configure(terraform.NewResourceConfig(rawConfig))
	if err != nil {
		t.Fatalf("err: %s", err)
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
