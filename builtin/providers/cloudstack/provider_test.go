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

	setValueOrID(&d, "id", "name", "54711781-274e-41b2-83c0-17194d0108f7")

	if d.Get("id").(string) != "name" {
		t.Fatal("err: 'id' does not match 'name'")
	}
}

func testSetIDOnResourceData(t *testing.T) {
	d := schema.ResourceData{}
	d.Set("id", "54711781-274e-41b2-83c0-17194d0108f7")

	setValueOrID(&d, "id", "name", "54711781-274e-41b2-83c0-17194d0108f7")

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
	if v := os.Getenv("CLOUDSTACK_2ND_NIC_IPADDRESS"); v == "" {
		t.Fatal("CLOUDSTACK_2ND_NIC_IPADDRESS must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_2ND_NIC_NETWORK"); v == "" {
		t.Fatal("CLOUDSTACK_2ND_NIC_NETWORK must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_DISK_OFFERING_1"); v == "" {
		t.Fatal("CLOUDSTACK_DISK_OFFERING_1 must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_DISK_OFFERING_2"); v == "" {
		t.Fatal("CLOUDSTACK_DISK_OFFERING_2 must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_HYPERVISOR"); v == "" {
		t.Fatal("CLOUDSTACK_HYPERVISOR must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_SERVICE_OFFERING_1"); v == "" {
		t.Fatal("CLOUDSTACK_SERVICE_OFFERING_1 must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_SERVICE_OFFERING_2"); v == "" {
		t.Fatal("CLOUDSTACK_SERVICE_OFFERING_2 must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_NETWORK_1"); v == "" {
		t.Fatal("CLOUDSTACK_NETWORK_1 must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_NETWORK_1_IPADDRESS1"); v == "" {
		t.Fatal("CLOUDSTACK_NETWORK_1_IPADDRESS1 must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_NETWORK_1_IPADDRESS2"); v == "" {
		t.Fatal("CLOUDSTACK_NETWORK_1_IPADDRESS2 must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_NETWORK_2"); v == "" {
		t.Fatal("CLOUDSTACK_NETWORK_2 must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_NETWORK_2_CIDR"); v == "" {
		t.Fatal("CLOUDSTACK_NETWORK_2_CIDR must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_NETWORK_2_OFFERING"); v == "" {
		t.Fatal("CLOUDSTACK_NETWORK_2_OFFERING must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_NETWORK_2_IPADDRESS"); v == "" {
		t.Fatal("CLOUDSTACK_NETWORK_2_IPADDRESS must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_VPC_CIDR_1"); v == "" {
		t.Fatal("CLOUDSTACK_VPC_CIDR_1 must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_VPC_CIDR_2"); v == "" {
		t.Fatal("CLOUDSTACK_VPC_CIDR_2 must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_VPC_OFFERING"); v == "" {
		t.Fatal("CLOUDSTACK_VPC_OFFERING must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_VPC_NETWORK_CIDR"); v == "" {
		t.Fatal("CLOUDSTACK_VPC_NETWORK_CIDR must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_VPC_NETWORK_OFFERING"); v == "" {
		t.Fatal("CLOUDSTACK_VPC_NETWORK_OFFERING must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_PUBLIC_IPADDRESS"); v == "" {
		t.Fatal("CLOUDSTACK_PUBLIC_IPADDRESS must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_SSH_PUBLIC_KEY"); v == "" {
		t.Fatal("CLOUDSTACK_SSH_PUBLIC_KEY must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_TEMPLATE"); v == "" {
		t.Fatal("CLOUDSTACK_TEMPLATE must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_TEMPLATE_FORMAT"); v == "" {
		t.Fatal("CLOUDSTACK_TEMPLATE_FORMAT must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_TEMPLATE_URL"); v == "" {
		t.Fatal("CLOUDSTACK_TEMPLATE_URL must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_TEMPLATE_OS_TYPE"); v == "" {
		t.Fatal("CLOUDSTACK_TEMPLATE_OS_TYPE must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_PROJECT_NAME"); v == "" {
		t.Fatal("CLOUDSTACK_PROJECT_NAME must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_PROJECT_NETWORK"); v == "" {
		t.Fatal("CLOUDSTACK_PROJECT_NETWORK must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSTACK_ZONE"); v == "" {
		t.Fatal("CLOUDSTACK_ZONE must be set for acceptance tests")
	}
}

// Name of a valid disk offering
var CLOUDSTACK_DISK_OFFERING_1 = os.Getenv("CLOUDSTACK_DISK_OFFERING_1")

// Name of a disk offering that CLOUDSTACK_DISK_OFFERING_1 can resize to
var CLOUDSTACK_DISK_OFFERING_2 = os.Getenv("CLOUDSTACK_DISK_OFFERING_2")

// Name of a valid service offering
var CLOUDSTACK_SERVICE_OFFERING_1 = os.Getenv("CLOUDSTACK_SERVICE_OFFERING_1")

// Name of a service offering that CLOUDSTACK_SERVICE_OFFERING_1 can resize to
var CLOUDSTACK_SERVICE_OFFERING_2 = os.Getenv("CLOUDSTACK_SERVICE_OFFERING_2")

// Name of a network that already exists
var CLOUDSTACK_NETWORK_1 = os.Getenv("CLOUDSTACK_NETWORK_1")

// A valid IP address in CLOUDSTACK_NETWORK_1
var CLOUDSTACK_NETWORK_1_IPADDRESS1 = os.Getenv("CLOUDSTACK_NETWORK_1_IPADDRESS1")

// A valid IP address in CLOUDSTACK_NETWORK_1
var CLOUDSTACK_NETWORK_1_IPADDRESS2 = os.Getenv("CLOUDSTACK_NETWORK_1_IPADDRESS2")

// Name for a network that will be created
var CLOUDSTACK_NETWORK_2 = os.Getenv("CLOUDSTACK_NETWORK_2")

// Any range
var CLOUDSTACK_NETWORK_2_CIDR = os.Getenv("CLOUDSTACK_NETWORK_2_CIDR")

// Name of an available network offering with specifyvlan=false
var CLOUDSTACK_NETWORK_2_OFFERING = os.Getenv("CLOUDSTACK_NETWORK_2_OFFERING")

// An IP address in CLOUDSTACK_NETWORK_2_CIDR
var CLOUDSTACK_NETWORK_2_IPADDRESS = os.Getenv("CLOUDSTACK_NETWORK_2_IPADDRESS")

// A network that already exists and isn't CLOUDSTACK_NETWORK_1
var CLOUDSTACK_2ND_NIC_NETWORK = os.Getenv("CLOUDSTACK_2ND_NIC_NETWORK")

// An IP address in CLOUDSTACK_2ND_NIC_NETWORK
var CLOUDSTACK_2ND_NIC_IPADDRESS = os.Getenv("CLOUDSTACK_2ND_NIC_IPADDRESS")

// Any range
var CLOUDSTACK_VPC_CIDR_1 = os.Getenv("CLOUDSTACK_VPC_CIDR_1")

// Any range that doesn't overlap to CLOUDSTACK_VPC_CIDR_1, will be VPNed
var CLOUDSTACK_VPC_CIDR_2 = os.Getenv("CLOUDSTACK_VPC_CIDR_2")

// An available VPC offering
var CLOUDSTACK_VPC_OFFERING = os.Getenv("CLOUDSTACK_VPC_OFFERING")

// A sub-range of CLOUDSTACK_VPC_CIDR_1 with same starting point
var CLOUDSTACK_VPC_NETWORK_CIDR = os.Getenv("CLOUDSTACK_VPC_NETWORK_CIDR")

// Name of an available network offering with forvpc=true
var CLOUDSTACK_VPC_NETWORK_OFFERING = os.Getenv("CLOUDSTACK_VPC_NETWORK_OFFERING")

// Path to a public IP that exists for CLOUDSTACK_NETWORK_1
var CLOUDSTACK_PUBLIC_IPADDRESS = os.Getenv("CLOUDSTACK_PUBLIC_IPADDRESS")

// Path to a public key on local disk
var CLOUDSTACK_SSH_PUBLIC_KEY = os.Getenv("CLOUDSTACK_SSH_PUBLIC_KEY")

// Name of a template that exists already for building VMs
var CLOUDSTACK_TEMPLATE = os.Getenv("CLOUDSTACK_TEMPLATE")

// Details of a template that will be added
var CLOUDSTACK_TEMPLATE_FORMAT = os.Getenv("CLOUDSTACK_TEMPLATE_FORMAT")
var CLOUDSTACK_HYPERVISOR = os.Getenv("CLOUDSTACK_HYPERVISOR")
var CLOUDSTACK_TEMPLATE_URL = os.Getenv("CLOUDSTACK_TEMPLATE_URL")
var CLOUDSTACK_TEMPLATE_OS_TYPE = os.Getenv("CLOUDSTACK_TEMPLATE_OS_TYPE")

// Name of a project that exists already
var CLOUDSTACK_PROJECT_NAME = os.Getenv("CLOUDSTACK_PROJECT_NAME")

// Name of a network that exists already in CLOUDSTACK_PROJECT_NAME
var CLOUDSTACK_PROJECT_NETWORK = os.Getenv("CLOUDSTACK_PROJECT_NETWORK")

// Name of a zone that exists already
var CLOUDSTACK_ZONE = os.Getenv("CLOUDSTACK_ZONE")
