package swift

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
		"swift": testAccProvider,
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

func TestProvider_Configure(t *testing.T) {
	//p := Provider()
	Provider()

	raw := map[string]interface{}{} // empty config. defaults from environment
	//rawConfig, err := config.NewRawConfig(raw)
	_, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	//err = p.Configure(terraform.NewResourceConfig(rawConfig))
	//if err != nil {
	//	t.Fatalf("err: %s", err)
	//}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("SWIFT_USERNAME"); v == "" {
		t.Fatal("SWIFT_USERNAME must be set for acceptance tests")
	}

	if v := os.Getenv("SWIFT_API_KEY"); v == "" {
		t.Fatal("SWIFT_API_KEY must be set for acceptance tests")
	}

	if v := os.Getenv("SWIFT_AUTH_URL"); v == "" {
		t.Fatal("SWIFT_AUTH_URL must be set for acceptance tests")
	}
}
