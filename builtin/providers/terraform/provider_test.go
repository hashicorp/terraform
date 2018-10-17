package terraform

import (
	"testing"

	backendinit "github.com/hashicorp/terraform/backend/init"
	"github.com/hashicorp/terraform/providers"
)

var testAccProviders map[string]*Provider
var testAccProvider *Provider

func init() {
	testAccProvider = NewProvider()
	testAccProviders = map[string]*Provider{
		"terraform": testAccProvider,
	}
	backendinit.Init(nil)
}

func TestProvider_impl(t *testing.T) {
	var _ providers.Interface = NewProvider()
}

func testAccPreCheck(t *testing.T) {
}
