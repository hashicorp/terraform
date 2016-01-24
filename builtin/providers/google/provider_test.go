package google

import (
	"io/ioutil"
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
		"google": testAccProvider,
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
	if v := os.Getenv("GOOGLE_CREDENTIALS_FILE"); v != "" {
		creds, err := ioutil.ReadFile(v)
		if err != nil {
			t.Fatalf("Error reading GOOGLE_CREDENTIALS_FILE path: %s", err)
		}
		os.Setenv("GOOGLE_CREDENTIALS", string(creds))
	}

	if v := os.Getenv("GOOGLE_CREDENTIALS"); v == "" {
		t.Fatal("GOOGLE_CREDENTIALS must be set for acceptance tests")
	}

	if v := os.Getenv("GOOGLE_PROJECT"); v == "" {
		t.Fatal("GOOGLE_PROJECT must be set for acceptance tests")
	}

	if v := os.Getenv("GOOGLE_REGION"); v != "us-central1" {
		t.Fatal("GOOGLE_REGION must be set to us-central1 for acceptance tests")
	}
}
