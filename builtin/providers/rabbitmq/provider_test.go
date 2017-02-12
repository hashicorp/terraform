package rabbitmq

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// To run these acceptance tests, you will need access to a RabbitMQ server
// with the management plugin enabled.
//
// Set the RABBITMQ_ENDPOINT, RABBITMQ_USERNAME, and RABBITMQ_PASSWORD
// environment variables before running the tests.
//
// You can run the tests like this:
//    make testacc TEST=./builtin/providers/rabbitmq

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"rabbitmq": testAccProvider,
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
	for _, name := range []string{"RABBITMQ_ENDPOINT", "RABBITMQ_USERNAME", "RABBITMQ_PASSWORD"} {
		if v := os.Getenv(name); v == "" {
			t.Fatal("RABBITMQ_ENDPOINT, RABBITMQ_USERNAME and RABBITMQ_PASSWORD must be set for acceptance tests")
		}
	}
}
