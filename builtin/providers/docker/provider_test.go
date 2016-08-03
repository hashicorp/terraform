package docker

import (
	"os/exec"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"os"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"docker": testAccProvider,
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
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Docker must be available: %s", err)
	}

	if v := os.Getenv("DOCKER_REGISTRY_ADDRESS"); v == "" {
		t.Fatalf("DOCKER_REGISTRY_ADDRESS must be set for acceptance tests")
	}

	if v := os.Getenv("DOCKER_REGISTRY_USER"); v == "" {
		t.Fatalf("DOCKER_REGISTRY_USER must be set for acceptance tests")
	}

	if v := os.Getenv("DOCKER_REGISTRY_PASS"); v == "" {
		t.Fatalf("DOCKER_REGISTRY_PASS must be set for acceptance tests")
	}

	if v := os.Getenv("DOCKER_PRIVATE_IMAGE"); v == "" {
		t.Fatalf("DOCKER_PRIVATE_IMAGE must be set for acceptance tests")
	}
}
