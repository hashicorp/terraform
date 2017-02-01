package pagerduty

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourcePagerDutySchedule_Basic(t *testing.T) {
	rName := acctest.RandString(5)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourcePagerDutyScheduleConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourcePagerDutySchedule("pagerduty_schedule.test", "data.pagerduty_schedule.by_name"),
				),
			},
		},
	})
}

func testAccDataSourcePagerDutySchedule(src, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		srcR := s.RootModule().Resources[src]
		srcA := srcR.Primary.Attributes

		r := s.RootModule().Resources[n]
		a := r.Primary.Attributes

		if a["id"] == "" {
			return fmt.Errorf("Expected to get a schedule ID from PagerDuty")
		}

		testAtts := []string{"id", "name"}

		for _, att := range testAtts {
			if a[att] != srcA[att] {
				return fmt.Errorf("Expected the schedule %s to be: %s, but got: %s", att, srcA[att], a[att])
			}
		}

		return nil
	}
}

func testAccDataSourcePagerDutyScheduleConfig(rName string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "test" {
  name  = "TF User %[1]s"
  email = "tf.%[1]s@example.com"
}

resource "pagerduty_schedule" "test" {
  name = "TF Schedule %[1]s"

  time_zone = "America/New_York"

  layer {
    name                         = "foo"
    start                        = "2015-11-06T20:00:00-05:00"
    rotation_virtual_start       = "2015-11-06T20:00:00-05:00"
    rotation_turn_length_seconds = 86400
    users                        = ["${pagerduty_user.test.id}"]

    restriction {
      type              = "weekly_restriction"
      start_time_of_day = "08:00:00"
      start_day_of_week = 5
      duration_seconds  = 32101
    }
  }
}

data "pagerduty_schedule" "by_name" {
  name = "${pagerduty_schedule.test.name}"
}
`, rName)
}
