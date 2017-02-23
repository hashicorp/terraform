package ibmcloud

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
		"ibmcloud": testAccProvider,
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

	requiredEnv := map[string]string{
		"ibmid":              "IBMID",
		"password":           "IBMID_PASSWORD",
		"softlayer_username": "SL_USERNAME or SOFTLAYER_USERNAME",
		"softlayer_api_key":  "SL_API_KEY or SOFTLAYER_API_KEY",
	}

	imageID := os.Getenv("IBMCLOUD_VIRTUAL_GUEST_IMAGE_ID")
	if imageID == "" {
		t.Logf("[WARN] Set the environment variable IBMCLOUD_VIRTUAL_GUEST_IMAGE_ID for testing " +
			"the ibmcloud_infra_virtual_guest resource. Some tests for that resource will fail if this is not set correctly")
	}

	for _, param := range []string{"ibmid", "password", "softlayer_username", "softlayer_api_key"} {
		value, _ := testAccProvider.Schema[param].DefaultFunc()
		if value == "" {
			t.Fatalf("%s must be set for acceptance test", requiredEnv[param])
		}
	}
}
