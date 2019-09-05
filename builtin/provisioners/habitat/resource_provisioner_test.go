package habitat

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvisioner_impl(t *testing.T) {
	var _ terraform.ResourceProvisioner = Provisioner()
}

func TestProvisioner(t *testing.T) {
	if err := Provisioner().(*schema.Provisioner).InternalValidate(); err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestResourceProvisioner_Validate_good(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"peers":        []interface{}{"1.2.3.4"},
		"version":      "0.32.0",
		"service_type": "systemd",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestResourceProvisioner_Validate_bad(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"service_type": "invalidtype",
		"url":          "badurl",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) != 2 {
		t.Fatalf("Should have two errors")
	}
}

func TestResourceProvisioner_Validate_bad_service_config(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"service": []interface{}{
			map[string]interface{}{"name": "core/foo", "strategy": "bar", "topology": "baz", "url": "badurl"},
		},
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) != 3 {
		t.Fatalf("Should have three errors: %v", errs)
	}
}

func TestResourceProvisioner_Validate_bad_service_definition(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"service": "core/vault",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) != 2 {
		t.Fatalf("Should have three errors: %v", errs)
	}
}

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	return terraform.NewResourceConfigRaw(c)
}
