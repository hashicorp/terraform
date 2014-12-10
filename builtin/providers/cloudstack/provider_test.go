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

	// Testing all environment/installation specific variables which are needed
	// to run all the acceptance tests
	if CLOUDSTACK_DISK_OFFERING_1 == "" {
		if v := os.Getenv("CLOUDSTACK_DISK_OFFERING_1"); v == "" {
			t.Fatal("CLOUDSTACK_DISK_OFFERING_1 must be set for acceptance tests")
		} else {
			CLOUDSTACK_DISK_OFFERING_1 = v
		}
	}
	if CLOUDSTACK_DISK_OFFERING_2 == "" {
		if v := os.Getenv("CLOUDSTACK_DISK_OFFERING_2"); v == "" {
			t.Fatal("CLOUDSTACK_DISK_OFFERING_2 must be set for acceptance tests")
		} else {
			CLOUDSTACK_DISK_OFFERING_2 = v
		}
	}
	if CLOUDSTACK_SERVICE_OFFERING_1 == "" {
		if v := os.Getenv("CLOUDSTACK_SERVICE_OFFERING_1"); v == "" {
			t.Fatal("CLOUDSTACK_SERVICE_OFFERING_1 must be set for acceptance tests")
		} else {
			CLOUDSTACK_SERVICE_OFFERING_1 = v
		}
	}
	if CLOUDSTACK_SERVICE_OFFERING_2 == "" {
		if v := os.Getenv("CLOUDSTACK_SERVICE_OFFERING_2"); v == "" {
			t.Fatal("CLOUDSTACK_SERVICE_OFFERING_2 must be set for acceptance tests")
		} else {
			CLOUDSTACK_SERVICE_OFFERING_2 = v
		}
	}
	if CLOUDSTACK_NETWORK_1 == "" {
		if v := os.Getenv("CLOUDSTACK_NETWORK_1"); v == "" {
			t.Fatal("CLOUDSTACK_NETWORK_1 must be set for acceptance tests")
		} else {
			CLOUDSTACK_NETWORK_1 = v
		}
	}
	if CLOUDSTACK_NETWORK_1_CIDR == "" {
		if v := os.Getenv("CLOUDSTACK_NETWORK_1_CIDR"); v == "" {
			t.Fatal("CLOUDSTACK_NETWORK_1_CIDR must be set for acceptance tests")
		} else {
			CLOUDSTACK_NETWORK_1_CIDR = v
		}
	}
	if CLOUDSTACK_NETWORK_1_OFFERING == "" {
		if v := os.Getenv("CLOUDSTACK_NETWORK_1_OFFERING"); v == "" {
			t.Fatal("CLOUDSTACK_NETWORK_1_OFFERING must be set for acceptance tests")
		} else {
			CLOUDSTACK_NETWORK_1_OFFERING = v
		}
	}
	if CLOUDSTACK_NETWORK_1_IPADDRESS == "" {
		if v := os.Getenv("CLOUDSTACK_NETWORK_1_IPADDRESS"); v == "" {
			t.Fatal("CLOUDSTACK_NETWORK_1_IPADDRESS must be set for acceptance tests")
		} else {
			CLOUDSTACK_NETWORK_1_IPADDRESS = v
		}
	}
	if CLOUDSTACK_NETWORK_2 == "" {
		if v := os.Getenv("CLOUDSTACK_NETWORK_2"); v == "" {
			t.Fatal("CLOUDSTACK_NETWORK_2 must be set for acceptance tests")
		} else {
			CLOUDSTACK_NETWORK_2 = v
		}
	}
	if CLOUDSTACK_NETWORK_2_IPADDRESS == "" {
		if v := os.Getenv("CLOUDSTACK_NETWORK_2_IPADDRESS"); v == "" {
			t.Fatal("CLOUDSTACK_NETWORK_2_IPADDRESS must be set for acceptance tests")
		} else {
			CLOUDSTACK_NETWORK_2_IPADDRESS = v
		}
	}
	if CLOUDSTACK_VPC_CIDR == "" {
		if v := os.Getenv("CLOUDSTACK_VPC_CIDR"); v == "" {
			t.Fatal("CLOUDSTACK_VPC_CIDR must be set for acceptance tests")
		} else {
			CLOUDSTACK_VPC_CIDR = v
		}
	}
	if CLOUDSTACK_VPC_OFFERING == "" {
		if v := os.Getenv("CLOUDSTACK_VPC_OFFERING"); v == "" {
			t.Fatal("CLOUDSTACK_VPC_OFFERING must be set for acceptance tests")
		} else {
			CLOUDSTACK_VPC_OFFERING = v
		}
	}
	if CLOUDSTACK_VPC_NETWORK_CIDR == "" {
		if v := os.Getenv("CLOUDSTACK_VPC_NETWORK_CIDR"); v == "" {
			t.Fatal("CLOUDSTACK_VPC_NETWORK_CIDR must be set for acceptance tests")
		} else {
			CLOUDSTACK_VPC_NETWORK_CIDR = v
		}
	}
	if CLOUDSTACK_VPC_NETWORK_OFFERING == "" {
		if v := os.Getenv("CLOUDSTACK_VPC_NETWORK_OFFERING"); v == "" {
			t.Fatal("CLOUDSTACK_VPC_NETWORK_OFFERING must be set for acceptance tests")
		} else {
			CLOUDSTACK_VPC_NETWORK_OFFERING = v
		}
	}
	if CLOUDSTACK_PUBLIC_IPADDRESS == "" {
		if v := os.Getenv("CLOUDSTACK_PUBLIC_IPADDRESS"); v == "" {
			t.Fatal("CLOUDSTACK_PUBLIC_IPADDRESS must be set for acceptance tests")
		} else {
			CLOUDSTACK_PUBLIC_IPADDRESS = v
		}
	}
	if CLOUDSTACK_TEMPLATE == "" {
		if v := os.Getenv("CLOUDSTACK_TEMPLATE"); v == "" {
			t.Fatal("CLOUDSTACK_TEMPLATE must be set for acceptance tests")
		} else {
			CLOUDSTACK_TEMPLATE = v
		}
	}
	if CLOUDSTACK_ZONE == "" {
		if v := os.Getenv("CLOUDSTACK_ZONE"); v == "" {
			t.Fatal("CLOUDSTACK_ZONE must be set for acceptance tests")
		} else {
			CLOUDSTACK_ZONE = v
		}
	}
}

// EITHER SET THESE, OR ADD THE VALUES TO YOUR ENV!!
var CLOUDSTACK_DISK_OFFERING_1 = ""
var CLOUDSTACK_DISK_OFFERING_2 = ""
var CLOUDSTACK_SERVICE_OFFERING_1 = ""
var CLOUDSTACK_SERVICE_OFFERING_2 = ""
var CLOUDSTACK_NETWORK_1 = ""
var CLOUDSTACK_NETWORK_1_CIDR = ""
var CLOUDSTACK_NETWORK_1_OFFERING = ""
var CLOUDSTACK_NETWORK_1_IPADDRESS = ""
var CLOUDSTACK_NETWORK_2 = ""
var CLOUDSTACK_NETWORK_2_IPADDRESS = ""
var CLOUDSTACK_VPC_CIDR = ""
var CLOUDSTACK_VPC_OFFERING = ""
var CLOUDSTACK_VPC_NETWORK_CIDR = ""
var CLOUDSTACK_VPC_NETWORK_OFFERING = ""
var CLOUDSTACK_PUBLIC_IPADDRESS = ""
var CLOUDSTACK_TEMPLATE = ""
var CLOUDSTACK_ZONE = ""
