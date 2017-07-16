package b2

import (
	"github.com/hashicorp/terraform/backend"
	"os"
	"testing"
)

// verify that we are doing ACC tests or the B2 tests specifically
func testACC(t *testing.T) {
	if os.Getenv("TF_ACC") == "" && os.Getenv("TF_B2_TEST") == "" {
		t.Skip("b2 backend tests require setting TF_ACC or TF_B2_TEST")
	}
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackendConfig(t *testing.T) {
	testACC(t)
	config := getB2Config(t)

	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	if b.bucketName != "tf-test" {
		t.Fatalf("Incorrect bucketName populated: %s", b.bucketName)
	}
	if b.keyName != "state" {
		t.Fatalf("Incorrect keyName was populated: %s", b.keyName)
	}

	if b.b2.AccountID == "" {
		t.Fatal("No Account ID set")
	}
	if b.b2.ApplicationKey == "" {
		t.Fatal("No Application Key set")
	}
}

func TestBackend(t *testing.T) {
	testACC(t)
	// TODO: mattmcnam
}

func TestBackendLocked(t *testing.T) {
	testACC(t)
	// TODO: mattmcnam
}

func TestBackendFindEnvironments(t *testing.T) {
	testACC(t)
	// TODO: mattmcnam
}

func getB2Config(t *testing.T) map[string]interface{} {
	if os.Getenv("B2_ACCOUNT_ID") == "" {
		t.Skip("skipping; B2_ACCOUNT_ID must be set")
	}

	if os.Getenv("B2_APPLICATION_KEY") == "" {
		t.Skip("skipping; B2_APPLICATION_KEY must be set")
	}

	return map[string]interface{}{
		"bucket":          "tf-test",
		"key":             "state",
		"account_id":      os.Getenv("B2_ACCOUNT_ID"),
		"application_key": os.Getenv("B2_APPLICATION_KEY"),
	}
}
