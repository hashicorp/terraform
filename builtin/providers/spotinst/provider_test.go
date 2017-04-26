package spotinst

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
		"spotinst": testAccProvider,
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
	c := map[string]string{
		"email":         os.Getenv("SPOTINST_EMAIL"),
		"password":      os.Getenv("SPOTINST_PASSWORD"),
		"client_id":     os.Getenv("SPOTINST_CLIENT_ID"),
		"client_secret": os.Getenv("SPOTINST_CLIENT_SECRET"),
		"token":         os.Getenv("SPOTINST_TOKEN"),
	}
	if c["password"] != "" && c["token"] != "" {
		t.Fatalf("ERR_CONFLICT: Both a password and a token were set, only one is required")
	}
	if c["password"] != "" && (c["email"] == "" || c["client_id"] == "" || c["client_secret"] == "") {
		t.Fatalf("ERR_MISSING: A password was set without email, client_id or client_secret")
	}
	if c["password"] == "" && c["token"] == "" {
		t.Fatalf("ERR_MISSING: A token is required if not using password")
	}
}
