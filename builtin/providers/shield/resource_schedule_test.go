package shield

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestShieldSchedule_basic(t *testing.T) {
	var schedule Schedule

	testShieldScheduleConfig := fmt.Sprintf(`
		resource "shield_schedule" "test_schedule" {
		  name = "Test-Schedule"
		  summary = "Terraform Test Schedule"
		  when = "daily 1am"
		}
	`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testShieldScheduleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testShieldScheduleConfig,
				Check: resource.ComposeTestCheckFunc(
					testShieldCheckScheduleExists("shield_schedule.test_schedule", &schedule),
				),
			},
		},
	})
}

func testShieldScheduleDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ShieldClient)
	rs, ok := s.RootModule().Resources["shield_schedule.test_schedule"]
	if !ok {
		return fmt.Errorf("Not found %s", "shield_schedule.test_schedule")
	}

	response, err := client.Get(fmt.Sprintf("v1/schedule/%s", rs.Primary.Attributes["uuid"]))

	if err != nil {
		return err
	}

	if response.StatusCode != 404 {
		return fmt.Errorf("Schedule still exists")
	}

	return nil
}

func testShieldCheckScheduleExists(n string, schedule *Schedule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Schedule UUID is set")
		}
		return nil
	}
}
