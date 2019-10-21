package puppet

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
		t.Fatalf("err: %s", err)
	}
}

func TestProvisioner_Validate_good_server(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"server": "puppet.test.com",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestProvisioner_Validate_bad_no_server(t *testing.T) {
	c := testConfig(t, map[string]interface{}{})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have errors")
	}
}

func TestProvisioner_Validate_bad_os_type(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"server":  "puppet.test.com",
		"os_type": "OS/2",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have errors")
	}
}

func TestProvisioner_Validate_good_os_type_linux(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"server":  "puppet.test.com",
		"os_type": "linux",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestProvisioner_Validate_good_os_type_windows(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"server":  "puppet.test.com",
		"os_type": "windows",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestProvisioner_Validate_bad_bolt_timeout(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"server":       "puppet.test.com",
		"bolt_timeout": "123oeau",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have errors")
	}
}

func TestProvisioner_Validate_good_bolt_timeout(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"server":       "puppet.test.com",
		"bolt_timeout": "123m",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", warn)
	}
}

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	return terraform.NewResourceConfigRaw(c)
}
