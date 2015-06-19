package azure

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureAffinityGroupBasic(t *testing.T) {
	name := "azure_affinity_group.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureAffinityGroupDestroyed,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureAffinityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureAffinityGroupExists(name),
					resource.TestCheckResourceAttr(name, "name", "terraform-testing-group"),
					resource.TestCheckResourceAttr(name, "location", "West US"),
					resource.TestCheckResourceAttr(name, "label", "A nice label."),
					resource.TestCheckResourceAttr(name, "description", "A nice description."),
				),
			},
		},
	})
}

func TestAccAzureAffinityGroupUpdate(t *testing.T) {
	name := "azure_affinity_group.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureAffinityGroupDestroyed,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureAffinityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureAffinityGroupExists(name),
					resource.TestCheckResourceAttr(name, "name", "terraform-testing-group"),
					resource.TestCheckResourceAttr(name, "location", "West US"),
					resource.TestCheckResourceAttr(name, "label", "A nice label."),
					resource.TestCheckResourceAttr(name, "description", "A nice description."),
				),
			},
			resource.TestStep{
				Config: testAccAzureAffinityGroupUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureAffinityGroupExists(name),
					resource.TestCheckResourceAttr(name, "name", "terraform-testing-group"),
					resource.TestCheckResourceAttr(name, "location", "West US"),
					resource.TestCheckResourceAttr(name, "label", "An even nicer label."),
					resource.TestCheckResourceAttr(name, "description", "An even nicer description."),
				),
			},
		},
	})
}

func testAccCheckAzureAffinityGroupExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Affinity Group resource %q doesn't exist.", name)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Affinity Group resource %q ID not set.", name)
		}

		affinityGroupClient := testAccProvider.Meta().(*Client).affinityGroupClient
		_, err := affinityGroupClient.GetAffinityGroup(resource.Primary.ID)
		return err
	}
}

func testAccCheckAzureAffinityGroupDestroyed(s *terraform.State) error {
	var err error
	affinityGroupClient := testAccProvider.Meta().(*Client).affinityGroupClient

	for _, resource := range s.RootModule().Resources {
		if resource.Type != "azure_affinity_group" {
			continue
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Affinity Group resource ID not set.")
		}

		_, err = affinityGroupClient.GetAffinityGroup(resource.Primary.ID)
		if !management.IsResourceNotFoundError(err) {
			return err
		}
	}

	return nil
}

const testAccAzureAffinityGroupConfig = `
resource "azure_affinity_group" "foo" {
	name = "terraform-testing-group"
	location = "West US"
	label = "A nice label."
	description = "A nice description."
}
`

const testAccAzureAffinityGroupUpdateConfig = `
resource "azure_affinity_group" "foo" {
	name = "terraform-testing-group"
	location = "West US"
	label = "An even nicer label."
	description = "An even nicer description."
}
`
