package chefsolo

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

func TestConfig_DecodeConfig_Happy(t *testing.T) {
	config := map[string]interface{}{
		"cookbook_paths": []interface{}{"chef/cookbooks", "chef/bookcooks"},
		"run_list":       []interface{}{"cookbook::recipe", "bookcook::recipe"},
		"json":           `{"hey":"hi", "a":{"b":"c", "d":10, "e":true, "f":null}}`,
	}
	_, err := decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, config),
	)
	if err != nil {
		t.Fatalf("Happy path failed: %v", "Error should not have triggered")
	}
}
