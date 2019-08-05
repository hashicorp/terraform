package habitat

import (
	"testing"

	"github.com/hashicorp/terraform/config"
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
		"peer":           "1.2.3.4",
		"version":        "0.32.0",
		"service_type":   "systemd",
		"accept_license": false,
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
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	//Two errors, one for service_type, other for missing required accept_license argument
	if len(errs) != 2 {
		t.Fatalf("Should have two errors")
	}
}

func TestResourceProvisioner_Validate_bad_service_config(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"accept_license": true,
		"service": []map[string]interface{}{
			map[string]interface{}{"name": "core/foo", "strategy": "bar", "topology": "baz", "url": "badurl"},
		},
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) != 3 {
		t.Fatalf("Should have three errors")
	}
}

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("config error: %s", err)
	}

	return terraform.NewResourceConfig(r)
}
