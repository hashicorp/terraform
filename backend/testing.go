package backend

import (
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

// TestBackendConfig validates and configures the backend with the
// given configuration.
func TestBackendConfig(t *testing.T, b Backend, c map[string]interface{}) Backend {
	// Get the proper config structure
	rc, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	conf := terraform.NewResourceConfig(rc)

	// Validate
	warns, errs := b.Validate(conf)
	if len(warns) > 0 {
		t.Fatalf("warnings: %s", warns)
	}
	if len(errs) > 0 {
		t.Fatalf("errors: %s", errs)
	}

	// Configure
	if err := b.Configure(conf); err != nil {
		t.Fatalf("err: %s", err)
	}

	return b
}
