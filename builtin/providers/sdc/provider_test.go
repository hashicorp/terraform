package sdc

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProvider *schema.Provider
var testAccProviders map[string]terraform.ResourceProvider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"sdc": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("SDC_ACCOUNT"); v == "" {
		t.Fatal("SDC_ACCOUNT must be set for acceptance tests")
	}

	if v := os.Getenv("SDC_URL"); v == "" {
		t.Fatal("SDC_URL must be set for acceptance tests")
	}

	if v := os.Getenv("SDC_KEY_ID"); v == "" {
		t.Fatal("SDC_KEY_ID must be set for acceptance tests")
	}

	if v := os.Getenv("MANTA_KEY_ID"); v == "" {
		t.Fatal("MANTA_KEY_ID must be set for acceptance tests")
	}
}
