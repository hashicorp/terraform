package triton

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"triton": testAccProvider,
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

func testAccPreCheck(t *testing.T) {
	sdcURL := os.Getenv("SDC_URL")
	account := os.Getenv("SDC_ACCOUNT")
	keyID := os.Getenv("SDC_KEY_ID")

	if sdcURL == "" {
		sdcURL = "https://us-west-1.api.joyentcloud.com"
	}

	if sdcURL == "" || account == "" || keyID == "" {
		t.Fatal("SDC_ACCOUNT and SDC_KEY_ID must be set for acceptance tests. To test with the SSH" +
			" private key signer, SDC_KEY_MATERIAL must also be set.")
	}
}
