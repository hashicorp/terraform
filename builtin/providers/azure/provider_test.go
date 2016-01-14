package azure

import (
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-homedir"
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
	sf := os.Getenv("PUBLISH_SETTINGS_FILE")
	if sf != "" {
		publishSettings, err := ioutil.ReadFile(sf)
		if err != nil {
			t.Fatalf("Error reading AZURE_SETTINGS_FILE path: %s", err)
		}

		os.Setenv("AZURE_PUBLISH_SETTINGS", string(publishSettings))
	}

	if v := os.Getenv("AZURE_PUBLISH_SETTINGS"); v == "" {
		subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
		certificate := os.Getenv("AZURE_CERTIFICATE")

		if subscriptionID == "" || certificate == "" {
			t.Fatal("either AZURE_PUBLISH_SETTINGS, PUBLISH_SETTINGS_FILE, or AZURE_SUBSCRIPTION_ID " +
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
	defer os.Remove(f.Name())

	fx, err := ioutil.TempFile("", "tf-test-xml")
	if err != nil {
		t.Fatalf("Error creating temporary file with XML in TestAzure_validateSettingsFile: %s", err)
	}
	defer os.Remove(fx.Name())
	_, err = io.WriteString(fx, "<PublishData></PublishData>")
	if err != nil {
		t.Fatalf("Error writing XML File: %s", err)
	}
	fx.Close()

	home, err := homedir.Dir()
	if err != nil {
		t.Fatalf("Error fetching homedir: %s", err)
	}
	fh, err := ioutil.TempFile(home, "tf-test-home")
	if err != nil {
		t.Fatalf("Error creating homedir-based temporary file: %s", err)
	}
	defer os.Remove(fh.Name())
	_, err = io.WriteString(fh, "<PublishData></PublishData>")
	if err != nil {
		t.Fatalf("Error writing XML File: %s", err)
	}
	fh.Close()

	r := strings.NewReplacer(home, "~")
	homePath := r.Replace(fh.Name())

	cases := []struct {
		Input string // String of XML or a path to an XML file
		W     int    // expected count of warnings
		E     int    // expected count of errors
	}{
		{"test", 0, 1},
		{f.Name(), 1, 1},
		{fx.Name(), 1, 0},
		{homePath, 1, 0},
		{"<PublishData></PublishData>", 0, 0},
	}

	for _, tc := range cases {
		w, e := validateSettingsFile(tc.Input, "")

		if len(w) != tc.W {
			t.Errorf("Error in TestAzureValidateSettingsFile: input: %s , warnings: %v, errors: %v", tc.Input, w, e)
		}
		if len(e) != tc.E {
			t.Errorf("Error in TestAzureValidateSettingsFile: input: %s , warnings: %v, errors: %v", tc.Input, w, e)
		}
	}
}

func TestAzure_providerConfigure(t *testing.T) {
	rp := Provider()
	raw := map[string]interface{}{
		"publish_settings": testAzurePublishSettingsStr,
	}

	rawConfig, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = rp.Configure(terraform.NewResourceConfig(rawConfig))
	meta := rp.(*schema.Provider).Meta()
	if meta == nil {
		t.Fatalf("Expected metadata, got nil: err: %s", err)
	}
}

func genRandInt() int {
	return rand.New(rand.NewSource(time.Now().UnixNano())).Int() % 100000
}

// testAzurePublishSettingsStr is a revoked publishsettings file
const testAzurePublishSettingsStr = `
<?xml version="1.0" encoding="utf-8"?>
<PublishData>
  <PublishProfile
    SchemaVersion="2.0"
    PublishMethod="AzureServiceManagementAPI">
    <Subscription
      ServiceManagementUrl="https://management.core.windows.net"
      Id="a65bf94f-26b3-4fb1-9d50-6e27c6096df1"
      Name="terraform-testing"
      ManagementCertificate="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" />
  </PublishProfile>
</PublishData>
`
