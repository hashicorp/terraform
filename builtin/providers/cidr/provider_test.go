package cidr

import (
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testProviders = map[string]terraform.ResourceProvider{
	"cidr": Provider(),
}
var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider
var testAccAwsProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"cidr": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("AWS_PROFILE"); v == "" {
		if v := os.Getenv("AWS_ACCESS_KEY_ID"); v == "" {
			t.Fatal("AWS_ACCESS_KEY_ID must be set for acceptance tests")
		}
		if v := os.Getenv("AWS_SECRET_ACCESS_KEY"); v == "" {
			t.Fatal("AWS_SECRET_ACCESS_KEY must be set for acceptance tests")
		}
	}
	if v := os.Getenv("AWS_DEFAULT_REGION"); v == "" {
		log.Println("[INFO] Test: Using us-west-2 as test region")
		os.Setenv("AWS_DEFAULT_REGION", "us-west-2")
	}
}
