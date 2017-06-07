package kubernetes

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/builtin/providers/google"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"kubernetes": testAccProvider,
		"google":     google.Provider(),
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
	hasFileCfg := (os.Getenv("KUBE_CTX_AUTH_INFO") != "" && os.Getenv("KUBE_CTX_CLUSTER") != "")
	hasStaticCfg := (os.Getenv("KUBE_HOST") != "" &&
		os.Getenv("KUBE_USER") != "" &&
		os.Getenv("KUBE_PASSWORD") != "" &&
		os.Getenv("KUBE_CLIENT_CERT_DATA") != "" &&
		os.Getenv("KUBE_CLIENT_KEY_DATA") != "" &&
		os.Getenv("KUBE_CLUSTER_CA_CERT_DATA") != "")

	if !hasFileCfg && !hasStaticCfg {
		t.Fatalf("File config (KUBE_CTX_AUTH_INFO and KUBE_CTX_CLUSTER) or static configuration"+
			" (%s) must be set for acceptance tests",
			strings.Join([]string{
				"KUBE_HOST",
				"KUBE_USER",
				"KUBE_PASSWORD",
				"KUBE_CLIENT_CERT_DATA",
				"KUBE_CLIENT_KEY_DATA",
				"KUBE_CLUSTER_CA_CERT_DATA",
			}, ", "))
	}

	if os.Getenv("GOOGLE_PROJECT") == "" || os.Getenv("GOOGLE_REGION") == "" || os.Getenv("GOOGLE_ZONE") == "" {
		t.Fatal("GOOGLE_PROJECT, GOOGLE_REGION and GOOGLE_ZONE must be set for acceptance tests")
	}
}
