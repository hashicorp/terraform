package scaleway

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
		"scaleway": testAccProvider,
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
	if v := os.Getenv("SCALEWAY_ORGANIZATION"); v == "" {
		t.Fatal("SCALEWAY_ORGANIZATION must be set for acceptance tests")
	}
	if v := os.Getenv("SCALEWAY_ACCESS_KEY"); v == "" {
		t.Fatal("SCALEWAY_ACCESS_KEY must be set for acceptance tests")
	}
}

func TestEnvWithScwrcFallbackFunc_FromEnv(t *testing.T) {
	os.Setenv("SCALEWAY_TEST", "foo")
	v, err := envWithScwrcFallbackFunc("SCALEWAY_TEST", "test", "dv")()
	if err != nil {
		t.Fatalf("Failed to lookup value from environment")
	}
	if v != "foo" {
		t.Fatalf("Failed to lookup correct value from environment")
	}
}

func TestEnvWithScwrcFallbackFunc_FromFile(t *testing.T) {
	os.Setenv("HOME", "/tmp")
	f, err := os.OpenFile("/tmp/.scwrc", os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"test": "bar"}`)
	f.Close()
	defer func() {
		os.Remove("/tmp/.scwrc")
	}()
	v, err := envWithScwrcFallbackFunc("SCALEWAY_TEST", "test", "dv")()
	if err != nil {
		t.Fatalf("Failed to lookup value from file")
	}
	if v != "bar" {
		t.Fatalf("Failed to lookup correct value from file. Expected %q got %q", "bar", v)
	}
}
