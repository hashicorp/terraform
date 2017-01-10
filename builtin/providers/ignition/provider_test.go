package ignition

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testProviders = map[string]terraform.ResourceProvider{
	"ignition": Provider(),
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestValidateUnit(t *testing.T) {
	if err := validateUnit(""); err == nil {
		t.Fatalf("error not found, expected error")
	}

	if err := validateUnit("[foo]qux"); err == nil {
		t.Fatalf("error not found, expected error")
	}

	if err := validateUnit("[foo]\nqux=foo\nfoo"); err == nil {
		t.Fatalf("error not found, expected error")
	}
}
