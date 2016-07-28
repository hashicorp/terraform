package helpers

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var TestAccProviders map[string]terraform.ResourceProvider
var TestAccProvider *schema.Provider

func TestAccPreCheck(t *testing.T) {
	if v := os.Getenv("VSPHERE_USER"); v == "" {
		t.Fatal("VSPHERE_USER must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_PASSWORD"); v == "" {
		t.Fatal("VSPHERE_PASSWORD must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_SERVER"); v == "" {
		t.Fatal("VSPHERE_SERVER must be set for acceptance tests")
	}
}
