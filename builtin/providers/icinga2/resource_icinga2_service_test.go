package icinga2

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/lrsmith/go-icinga2-api/iapi"
)

func TestAccCreateService(t *testing.T) {

	var testAccCreateService = fmt.Sprintf(`
		resource "icinga2_service" "tf-service-1" {
		hostname      = "docker-icinga2"
		name          = "ssh3"
		check_command = "ssh"
	}`)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCreateService,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceExists("icinga2_service.tf-service-1"),
					testAccCheckResourceState("icinga2_service.tf-service-1", "hostname", "docker-icinga2"),
					testAccCheckResourceState("icinga2_service.tf-service-1", "name", "ssh3"),
					testAccCheckResourceState("icinga2_service.tf-service-1", "check_command", "ssh"),
				),
			},
		},
	})
}

func testAccCheckServiceExists(rn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("Service resource not found: %s", rn)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*iapi.Server)
		tokens := strings.Split(resource.Primary.ID, "!")

		_, err := client.GetService(tokens[1], tokens[0])
		if err != nil {
			return fmt.Errorf("error getting getting Service: %s", err)
		}

		return nil
	}

}
