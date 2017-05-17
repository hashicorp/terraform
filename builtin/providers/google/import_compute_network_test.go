package google

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccComputeNetwork_importBasic(t *testing.T) {
	resourceName := "google_compute_network.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeNetwork_basic,
			}, {
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				//ImportStateVerifyIgnore: []string{"ipv4_range", "name"},
			},
		},
	})
}

func TestAccComputeNetwork_importAuto_subnet(t *testing.T) {
	resourceName := "google_compute_network.bar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeNetwork_auto_subnet,
			}, {
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccComputeNetwork_importCustom_subnet(t *testing.T) {
	resourceName := "google_compute_network.baz"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeNetwork_custom_subnet,
			}, {
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
