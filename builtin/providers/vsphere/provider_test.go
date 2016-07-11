package vsphere

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"vsphere": testAccProvider,
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
	if v := os.Getenv("VSPHERE_USER"); v == "" {
		t.Fatal("VSPHERE_USER must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_PASSWORD"); v == "" {
		t.Fatal("VSPHERE_PASSWORD must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_SERVER"); v == "" {
		t.Fatal("VSPHERE_SERVER must be set for acceptance tests")
	}
}

// validateEnvArgs is a helper function to verify that required test related environment variables are set.
func validateEnvArgs(t *testing.T, requiredVars ...string) {
	realEnvVars := os.Environ()
	for _, requiredVar := range requiredVars {
		for _, v := range realEnvVars {
			if requiredVar == strings.Split(v, "=")[0] {
				// Remove the variable from the list of required variables if the required variable is defined in the real environment.
				// This way of removing the required variable from the slice preserves the order of the slice (not really needed in this case though).
				requiredVars = requiredVars[:len(requiredVars)-1]
			}
		}
	}

	if len(requiredVars) > 0 {
		t.Fatalf("Some required environment variables are missing: %s\n", requiredVars)
	}
}
