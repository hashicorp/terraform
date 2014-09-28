package mailgun

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/mailgun"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"mailgun": testAccProvider,
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

	if v := os.Getenv("MAILGUN_API_KEY"); v != "" {
		expectedKey = v
	} else {
		expectedKey = "foo"
	}

	raw := map[string]interface{}{
		"api_key": expectedKey,
	}

	rawConfig, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	rp := Provider().(*schema.Provider)
	err = rp.Configure(terraform.NewResourceConfig(rawConfig))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	config := rp.Meta().(*mailgun.Client)
	if config.ApiKey != expectedKey {
		t.Fatalf("bad: %#v", config)
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("MAILGUN_API_KEY"); v == "" {
		t.Fatal("MAILGUN_API_KEY must be set for acceptance tests")
	}
}
