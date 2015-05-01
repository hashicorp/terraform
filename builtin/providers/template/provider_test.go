package template

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
)

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
