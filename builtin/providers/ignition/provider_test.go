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
	if err := validateUnitContent(""); err == nil {
		t.Fatalf("error not found, expected error")
	}

	if err := validateUnitContent("[foo]qux"); err == nil {
		t.Fatalf("error not found, expected error")
	}

	if err := validateUnitContent("[foo]\nqux=foo\nfoo"); err == nil {
		t.Fatalf("error not found, expected error")
	}
}
