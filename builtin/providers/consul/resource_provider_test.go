package consul

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/consul/testutil"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider
var testConsulHTTPAddr string

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProvider.ConfigureFunc = testProviderConfigure

	testAccProviders = map[string]terraform.ResourceProvider{
		"consul": testAccProvider,
	}
}

// we need to overrride the configured address for the tests
func testProviderConfigure(d *schema.ResourceData) (interface{}, error) {
	var config Config
	configRaw := d.Get("").(map[string]interface{})
	if err := mapstructure.Decode(configRaw, &config); err != nil {
		return nil, err
	}
	config.Address = testConsulHTTPAddr

	log.Printf("[INFO] Initializing Consul test client")
	return config.Client()
}

func TestMain(m *testing.M) {
	t := struct {
		testutil.TestingT
	}{}

	// start and stop the test consul server once for all tests
	srv := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.LogLevel = "warn"
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})

	testConsulHTTPAddr = srv.HTTPAddr

	ret := m.Run()

	srv.Stop()
	os.Exit(ret)
}

func TestResourceProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestResourceProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func TestResourceProvider_Configure(t *testing.T) {
	rp := Provider()

	// these configuration tests don't require an running server
	raw := map[string]interface{}{
		"address":    "example.com:8500",
		"datacenter": "nyc3",
		"scheme":     "https",
	}

	rawConfig, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = rp.Configure(terraform.NewResourceConfig(rawConfig))
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestResourceProvider_ConfigureTLS(t *testing.T) {
	rp := Provider()

	raw := map[string]interface{}{
		"address":    "example.com:8943",
		"ca_file":    "test-fixtures/cacert.pem",
		"cert_file":  "test-fixtures/usercert.pem",
		"datacenter": "nyc3",
		"key_file":   "test-fixtures/userkey.pem",
		"scheme":     "https",
	}

	rawConfig, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = rp.Configure(terraform.NewResourceConfig(rawConfig))
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}
