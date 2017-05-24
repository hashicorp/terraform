package external

import (
	"testing"

	"github.com/r3labs/terraform/helper/schema"
	"github.com/r3labs/terraform/terraform"
)

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

var testProviders = map[string]terraform.ResourceProvider{
	"external": Provider(),
}
