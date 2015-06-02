package cloudstack

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"cloudstack": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func testSetValueOnResourceData(t *testing.T) {
	d := schema.ResourceData{}
	d.Set("id", "name")

	setValueOrUUID(&d, "id", "name", "54711781-274e-41b2-83c0-17194d0108f7")

	if d.Get("id").(string) != "name" {
		t.Fatal("err: 'id' does not match 'name'")
	}
}

func testSetUuidOnResourceData(t *testing.T) {
	d := schema.ResourceData{}
	d.Set("id", "54711781-274e-41b2-83c0-17194d0108f7")

	setValueOrUUID(&d, "id", "name", "54711781-274e-41b2-83c0-17194d0108f7")

	if d.Get("id").(string) != "54711781-274e-41b2-83c0-17194d0108f7" {
		t.Fatal("err: 'id' doest not match '54711781-274e-41b2-83c0-17194d0108f7'")
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("CLOUDSTACK_API_URL"); v == "" {
		t.Fatal("CLOUDSTACK_API_URL must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_API_KEY"); v == "" {
		t.Fatal("CLOUDSTACK_API_KEY must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_SECRET_KEY"); v == "" {
		t.Fatal("CLOUDSTACK_SECRET_KEY must be set for acceptance tests")
	}
}

// SET THESE VALUES IN ORDER TO RUN THE ACC TESTS!!
var CLOUDSTACK_2ND_NIC_IPADDRESS = ""
var CLOUDSTACK_2ND_NIC_NETWORK = ""
var CLOUDSTACK_DISK_OFFERING_1 = ""
var CLOUDSTACK_DISK_OFFERING_2 = ""
var CLOUDSTACK_HYPERVISOR = ""
var CLOUDSTACK_SERVICE_OFFERING_1 = ""
var CLOUDSTACK_SERVICE_OFFERING_2 = ""
var CLOUDSTACK_NETWORK_1 = ""
var CLOUDSTACK_NETWORK_1_IPADDRESS = ""
var CLOUDSTACK_NETWORK_2 = ""
var CLOUDSTACK_NETWORK_2_CIDR = ""
var CLOUDSTACK_NETWORK_2_OFFERING = ""
var CLOUDSTACK_NETWORK_2_IPADDRESS = ""
var CLOUDSTACK_VPC_CIDR_1 = ""
var CLOUDSTACK_VPC_CIDR_2 = ""
var CLOUDSTACK_VPC_OFFERING = ""
var CLOUDSTACK_VPC_NETWORK_CIDR = ""
var CLOUDSTACK_VPC_NETWORK_OFFERING = ""
var CLOUDSTACK_PUBLIC_IPADDRESS = ""
var CLOUDSTACK_SSH_KEYPAIR = ""
var CLOUDSTACK_SSH_PUBLIC_KEY = ""
var CLOUDSTACK_TEMPLATE = ""
var CLOUDSTACK_TEMPLATE_FORMAT = ""
var CLOUDSTACK_TEMPLATE_URL = ""
var CLOUDSTACK_TEMPLATE_OS_TYPE = ""
var CLOUDSTACK_ZONE = ""
