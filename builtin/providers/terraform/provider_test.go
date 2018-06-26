package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/providers"
)

var testAccProviders map[string]*Provider
var testAccProvider *Provider

func init() {
	testAccProvider = NewProvider()
	testAccProviders = map[string]*Provider{
		"terraform": testAccProvider,
	}
}

// func TestProvider(t *testing.T) {
// 	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
// 		t.Fatalf("err: %s", err)
// 	}
// }

func TestProvider_impl(t *testing.T) {
	var _ providers.Interface = NewProvider()
}

func testAccPreCheck(t *testing.T) {
}
