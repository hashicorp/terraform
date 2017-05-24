package http

import (
	"testing"

	"github.com/r3labs/terraform/helper/schema"
	"github.com/r3labs/terraform/terraform"
)

var testProviders = map[string]terraform.ResourceProvider{
	"http": Provider(),
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
