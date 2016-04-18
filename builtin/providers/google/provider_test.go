package google

import (
	"io/ioutil"
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

	multiEnvSearch := func(ks []string) string {
		for _, k := range ks {
			if v := os.Getenv(k); v != "" {
				return v
			}
		}
		return ""
	}

	creds := []string{
		"GOOGLE_CREDENTIALS",
		"GOOGLE_CLOUD_KEYFILE_JSON",
		"GCLOUD_KEYFILE_JSON",
	}
	if v := multiEnvSearch(creds); v == "" {
		t.Fatalf("One of %s must be set for acceptance tests", strings.Join(creds, ", "))
	}

	projs := []string{
		"GOOGLE_PROJECT",
		"GCLOUD_PROJECT",
		"CLOUDSDK_CORE_PROJECT",
	}
	if v := multiEnvSearch(projs); v == "" {
		t.Fatalf("One of %s must be set for acceptance tests", strings.Join(creds, ", "))
	}

	regs := []string{
		"GOOGLE_REGION",
		"GCLOUD_REGION",
		"CLOUDSDK_COMPUTE_REGION",
	}
	if v := multiEnvSearch(regs); v != "us-central-1" {
		t.Fatalf("One of %s must be set to us-central-1 for acceptance tests", strings.Join(creds, ", "))
	}
}

func TestProvider_getRegionFromZone(t *testing.T) {
	expected := "us-central1"
	actual := getRegionFromZone("us-central1-f")
	if expected != actual {
		t.Fatalf("Region (%s) did not match expected value: %s", actual, expected)
	}
}
