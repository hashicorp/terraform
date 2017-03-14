package ibmcloud

import (
	"fmt"
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

	if v := os.Getenv("IBMID"); v == "" {
		t.Fatal("IBMID must be set for acceptance tests")
	}
	if v := os.Getenv("IBMID_PASSWORD"); v == "" {
		t.Fatal("IBMID_PASSWORD must be set for acceptance tests")
	}

	slAccountNumber, err := schema.MultiEnvDefaultFunc([]string{"SL_ACCOUNT_NUMBER", "SOFTLAYER_ACCOUNT_NUMBER"}, "")()
	if err != nil {
		t.Fatalf("Failed to check env variables SL_ACCOUNT_NUMBER and SOFTLAYER_ACCOUNT_NUMBER %+v", err)
	}
	if slAccountNumber == "" {
		fmt.Println("[WARN] Test: SL_ACCOUNT_NUMBER or SOFTLAYER_ACCOUNT_NUMBER not set.", "Test will use the default SoftLayer account number that you have linked with your IBM ID.",
			"You can check your default account number from the SoftLayer control portal. You can also set the default Account Number in the portal if you have more than one account")
	}
}
