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
		t.Fatalf("One of %s must be set for acceptance tests", strings.Join(projs, ", "))
	}

	regs := []string{
		"GOOGLE_REGION",
		"GCLOUD_REGION",
		"CLOUDSDK_COMPUTE_REGION",
	}
	if v := multiEnvSearch(regs); v != "us-central1" {
		t.Fatalf("One of %s must be set to us-central1 for acceptance tests", strings.Join(regs, ", "))
	}

	if v := os.Getenv("GOOGLE_XPN_HOST_PROJECT"); v == "" {
		t.Fatal("GOOGLE_XPN_HOST_PROJECT must be set for acceptance tests")
	}
}

func TestProvider_getRegionFromZone(t *testing.T) {
	expected := "us-central1"
	actual := getRegionFromZone("us-central1-f")
	if expected != actual {
		t.Fatalf("Region (%s) did not match expected value: %s", actual, expected)
	}
}

func TestParseUrl(t *testing.T) {
	cases := map[string]struct {
		Input    string
		Expected resourceInfo
		Error    bool
	}{
		"Project (full url)": {
			Input: "https://www.googleapis.com/compute/beta/projects/myproject",
			Expected: resourceInfo{
				apiVersion:   "beta",
				project:      "myproject",
				region:       "",
				zone:         "",
				resourceType: "projects",
				name:         "myproject",
				url:          "https://www.googleapis.com/compute/beta/projects/myproject",
			},
		},
		"Project (partial url)": {
			Input: "projects/myproject",
			Expected: resourceInfo{
				apiVersion:   "v1",
				project:      "myproject",
				region:       "",
				zone:         "",
				resourceType: "projects",
				name:         "myproject",
				url:          "https://www.googleapis.com/compute/v1/projects/myproject",
			},
		},
		"Region (full url)": {
			Input: "https://www.googleapis.com/compute/beta/projects/myproject/regions/us-central1",
			Expected: resourceInfo{
				apiVersion:   "beta",
				project:      "myproject",
				region:       "us-central1",
				zone:         "",
				resourceType: "regions",
				name:         "us-central1",
				url:          "https://www.googleapis.com/compute/beta/projects/myproject/regions/us-central1",
			},
		},
		"Region (partial url)": {
			Input: "projects/myproject/regions/us-central1",
			Expected: resourceInfo{
				apiVersion:   "v1",
				project:      "myproject",
				region:       "us-central1",
				zone:         "",
				resourceType: "regions",
				name:         "us-central1",
				url:          "https://www.googleapis.com/compute/v1/projects/myproject/regions/us-central1",
			},
		},
		"Zone (full url)": {
			Input: "https://www.googleapis.com/compute/beta/projects/myproject/zones/us-central1-f",
			Expected: resourceInfo{
				apiVersion:   "beta",
				project:      "myproject",
				region:       "us-central1",
				zone:         "us-central1-f",
				resourceType: "zones",
				name:         "us-central1-f",
				url:          "https://www.googleapis.com/compute/beta/projects/myproject/zones/us-central1-f",
			},
		},
		"Zone (partial url)": {
			Input: "projects/myproject/zones/us-central1-f",
			Expected: resourceInfo{
				apiVersion:   "v1",
				project:      "myproject",
				region:       "us-central1",
				zone:         "us-central1-f",
				resourceType: "zones",
				name:         "us-central1-f",
				url:          "https://www.googleapis.com/compute/v1/projects/myproject/zones/us-central1-f",
			},
		},
		"Global resource: image (full url)": {
			Input: "https://www.googleapis.com/compute/beta/projects/myproject/global/images/myimage",
			Expected: resourceInfo{
				apiVersion:   "beta",
				project:      "myproject",
				region:       "",
				zone:         "",
				resourceType: "images",
				name:         "myimage",
				url:          "https://www.googleapis.com/compute/beta/projects/myproject/global/images/myimage",
			},
		},
		"Global resource: image (partial url)": {
			Input: "projects/myproject/global/images/myimage",
			Expected: resourceInfo{
				apiVersion:   "v1",
				project:      "myproject",
				region:       "",
				zone:         "",
				resourceType: "images",
				name:         "myimage",
				url:          "https://www.googleapis.com/compute/v1/projects/myproject/global/images/myimage",
			},
		},
		"Image family (full url)": {
			Input: "https://www.googleapis.com/compute/beta/projects/myproject/global/images/family/myfamily",
			Expected: resourceInfo{
				apiVersion:   "beta",
				project:      "myproject",
				region:       "",
				zone:         "",
				resourceType: "images/family",
				name:         "myfamily",
				url:          "https://www.googleapis.com/compute/beta/projects/myproject/global/images/family/myfamily",
			},
		},
		"Image family (partial url)": {
			Input: "projects/myproject/global/images/family/myfamily",
			Expected: resourceInfo{
				apiVersion:   "v1",
				project:      "myproject",
				region:       "",
				zone:         "",
				resourceType: "images/family",
				name:         "myfamily",
				url:          "https://www.googleapis.com/compute/v1/projects/myproject/global/images/family/myfamily",
			},
		},
		"Regional resource: address (full url)": {
			Input: "https://www.googleapis.com/compute/beta/projects/myproject/regions/us-central1/addresses/myaddress",
			Expected: resourceInfo{
				apiVersion:   "beta",
				project:      "myproject",
				region:       "us-central1",
				zone:         "",
				resourceType: "addresses",
				name:         "myaddress",
				url:          "https://www.googleapis.com/compute/beta/projects/myproject/regions/us-central1/addresses/myaddress",
			},
		},
		"Regional resource: address (partial url)": {
			Input: "projects/myproject/regions/us-central1/addresses/myaddress",
			Expected: resourceInfo{
				apiVersion:   "v1",
				project:      "myproject",
				region:       "us-central1",
				zone:         "",
				resourceType: "addresses",
				name:         "myaddress",
				url:          "https://www.googleapis.com/compute/v1/projects/myproject/regions/us-central1/addresses/myaddress",
			},
		},
		"Zonal resource: instance (full url)": {
			Input: "https://www.googleapis.com/compute/beta/projects/myproject/zones/us-central1-f/instances/myinstance",
			Expected: resourceInfo{
				apiVersion:   "beta",
				project:      "myproject",
				region:       "us-central1",
				zone:         "us-central1-f",
				resourceType: "instances",
				name:         "myinstance",
				url:          "https://www.googleapis.com/compute/beta/projects/myproject/zones/us-central1-f/instances/myinstance",
			},
		},
		"Zonal resource: instance (partial url)": {
			Input: "projects/myproject/zones/us-central1-f/instances/myinstance",
			Expected: resourceInfo{
				apiVersion:   "v1",
				project:      "myproject",
				region:       "us-central1",
				zone:         "us-central1-f",
				resourceType: "instances",
				name:         "myinstance",
				url:          "https://www.googleapis.com/compute/v1/projects/myproject/zones/us-central1-f/instances/myinstance",
			},
		},
		"Invalid URL: only name": {
			Input: "onlyname",
			Error: true,
		},
	}

	for key, tc := range cases {
		got, err := parseUrl(tc.Input)
		if !tc.Error {
			if err != nil {
				t.Fatalf("[%s] Unexpected error happend: %s", key, err)
			}
			if got != tc.Expected {
				t.Fatalf("[%s] Expected '%#v', but got '%#v'", key, tc.Expected, got)
			}
		} else {
			if err == nil {
				t.Fatalf("[%s] Expected error to happen, but got '%#v' without any errors", key, got)
			}
		}
	}
}
