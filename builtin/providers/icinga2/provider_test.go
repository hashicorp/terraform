package icinga2

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
		"icinga2": testAccProvider,
	}
}

// Environment variable to export and for testig using Icinga2 Vagrant Image
//   available from https://github.com/Icinga/icinga-vagrant
// export ICINGA2_API_URL=https://192.168.33.5:5665
// export ICINGA2_API_USER=root
// export ICINGA2_API_PASSWORD=icinga

func Test_provider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func testAccPreCheck(t *testing.T) {

	v := os.Getenv("ICINGA2_API_URL")
	if v == "" {
		t.Fatal("ICINGA2_API_URL must be set for acceptance tests")
	}

	v = os.Getenv("ICINGA2_API_USER")
	if v == "" {
		t.Fatal("ICINGA2_API_USER must be set for acceptance tests")
	}

	v = os.Getenv("ICINGA2_API_PASSWORD")
	if v == "" {
		t.Fatal("ICINGA2_API_PASSWORD must be set for acceptance tests")
	}

}
