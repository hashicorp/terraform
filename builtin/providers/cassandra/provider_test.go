package cassandra

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// To run these acceptance tests, you will need a Cassandra server.
// If you download a Cassandra distribution and run it with its default
// settings, on the same host where the tests are being run, then these tests
// should work with no further configuration.
//
// To run the tests against a remote Cassandra server, set the CASSANDRA_HOSTPORT,
// CASSANDRA_USERNAME and CASSANDRA_PASSWORD environment variables.

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"cassandra": testAccProvider,
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
	if v := os.Getenv("CASSANDRA_HOSTPORT"); v == "" {
		t.Fatal("CASSANDRA_HOSTPORT must be set for cassandra acceptance tests")
	}
}
