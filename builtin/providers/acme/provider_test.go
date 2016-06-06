package acme

import (
	"testing"

	"github.com/hashicorp/terraform/builtin/providers/aws"
	"github.com/hashicorp/terraform/builtin/providers/tls"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProvider *schema.Provider
var testAccProviders map[string]terraform.ResourceProvider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"acme": testAccProvider,
		"tls":  tls.Provider().(*schema.Provider),
		"aws":  aws.Provider().(*schema.Provider),
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
