package runscope

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
		// provider is called terraform-provider-runscope ie runscope
		"runscope": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}

}

func TestProviderImpl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("RUNSCOPE_ACCESS_TOKEN"); v == "" {
		t.Fatal("RUNSCOPE_ACCESS_TOKEN must be set for acceptance tests")
	}

	if v := os.Getenv("RUNSCOPE_TEAM_ID"); v == "" {
		t.Fatal("RUNSCOPE_TEAM_ID must be set for acceptance tests")
	}
}
