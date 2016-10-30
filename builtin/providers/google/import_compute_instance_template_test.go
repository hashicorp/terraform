package google

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccComputeInstanceTemplate_importBasic(t *testing.T) {
	resourceName := "google_compute_instance_template.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeInstanceTemplate_basic,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccComputeInstanceTemplate_importIp(t *testing.T) {
	resourceName := "google_compute_instance_template.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeInstanceTemplate_ip,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccComputeInstanceTemplate_importDisks(t *testing.T) {
	resourceName := "google_compute_instance_template.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeInstanceTemplate_disks,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccComputeInstanceTemplate_importSubnetAuto(t *testing.T) {
	resourceName := "google_compute_instance_template.foobar"
	network := "network-" + acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeInstanceTemplate_subnet_auto(network),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccComputeInstanceTemplate_importSubnetCustom(t *testing.T) {
	resourceName := "google_compute_instance_template.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeInstanceTemplate_subnet_custom,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
