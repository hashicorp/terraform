package azure

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

const (
	testAccSecurityGroupName = "terraform-security-group"
	testAccHostedServiceName = "terraform-testing-service"
)

// testAccStorageServiceName is used as the name for the Storage Service
// created in all storage-related tests.
// It is much more convenient to provide a Storage Service which
// has been created beforehand as the creation of one takes a lot
// and would greatly impede the multitude of tests which rely on one.
// NOTE: the storage container should be located in `West US`.
var testAccStorageServiceName = os.Getenv("AZURE_STORAGE")

const testAccStorageContainerName = "terraform-testing-container"

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"azure": testAccProvider,
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
	if v := os.Getenv("AZURE_SETTINGS_FILE"); v == "" {
		subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
		certificate := os.Getenv("AZURE_CERTIFICATE")

		if subscriptionID == "" || certificate == "" {
			t.Fatal("either AZURE_SETTINGS_FILE, or AZURE_SUBSCRIPTION_ID " +
				"and AZURE_CERTIFICATE must be set for acceptance tests")
		}
	}

	if v := os.Getenv("AZURE_STORAGE"); v == "" {
		t.Fatal("AZURE_STORAGE must be set for acceptance tests")
	}
}

func TestAzure_validateSettingsFile(t *testing.T) {
	f, err := ioutil.TempFile("", "tf-test")
	if err != nil {
		t.Fatalf("Error creating temporary file in TestAzure_validateSettingsFile: %s", err)
	}

	fx, err := ioutil.TempFile("", "tf-test-xml")
	if err != nil {
		t.Fatalf("Error creating temporary file with XML in TestAzure_validateSettingsFile: %s", err)
	}

	_, err = io.WriteString(fx, "<PublishData></PublishData>")
	if err != nil {
		t.Fatalf("Error writing XML File: %s", err)
	}

	log.Printf("fx name: %s", fx.Name())
	fx.Close()

	cases := []struct {
		Input string // String of XML or a path to an XML file
		W     int    // expected count of warnings
		E     int    // expected count of errors
	}{
		{"test", 1, 1},
		{f.Name(), 1, 0},
		{fx.Name(), 1, 0},
		{"<PublishData></PublishData>", 0, 0},
	}

	for _, tc := range cases {
		w, e := validateSettingsFile(tc.Input, "")

		if len(w) != tc.W {
			t.Errorf("Error in TestAzureValidateSettingsFile: input: %s , warnings: %#v, errors: %#v", tc.Input, w, e)
		}
		if len(e) != tc.E {
			t.Errorf("Error in TestAzureValidateSettingsFile: input: %s , warnings: %#v, errors: %#v", tc.Input, w, e)
		}
	}
}

func TestAzure_isFile(t *testing.T) {
	f, err := ioutil.TempFile("", "tf-test-file")
	if err != nil {
		t.Fatalf("Error creating temporary file with XML in TestAzure_isFile: %s", err)
	}
	cases := []struct {
		Input string // String path to file
		B     bool   // expected true/false
		E     bool   // expect error
	}{
		{"test", false, true},
		{f.Name(), true, false},
	}

	for _, tc := range cases {
		x, y := isFile(tc.Input)
		if tc.B != x {
			t.Errorf("Error in TestAzure_isFile: input: %s , returned: %#v, expected: %#v", tc.Input, x, tc.B)
		}

		if tc.E != (y != nil) {
			t.Errorf("Error in TestAzure_isFile: input: %s , returned: %#v, expected: %#v", tc.Input, y, tc.E)
		}
	}
}
