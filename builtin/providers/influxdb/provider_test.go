package influxdb

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// To run these acceptance tests, you will need an InfluxDB server.
// If you download an InfluxDB distribution and run it with its default
// settings, on the same host where the tests are being run, then these tests
// should work with no further configuration.
//
// To run the tests against a remote InfluxDB server, set the INFLUXDB_URL,
// INFLUXDB_USERNAME and INFLUXDB_PASSWORD environment variables.

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"influxdb": testAccProvider,
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
