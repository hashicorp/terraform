package pagerduty

import (
	"fmt"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPagerDutySchedule_Basic(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)
	schedule := fmt.Sprintf("tf-%s", acctest.RandString(5))
	scheduleUpdated := fmt.Sprintf("tf-%s", acctest.RandString(5))
	location := "America/New_York"
	start := "2020-05-12T20:00:00-04:00"
	rotationVirtualStart := "2020-05-12T20:00:00-04:00"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyScheduleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyScheduleConfig(username, email, schedule, location, start, rotationVirtualStart),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleExists("pagerduty_schedule.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "name", schedule),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "description", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "time_zone", location),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.start", start),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.rotation_virtual_start", rotationVirtualStart),
				),
			},
			{
				Config: testAccCheckPagerDutyScheduleConfigUpdated(username, email, scheduleUpdated, location, start, rotationVirtualStart),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleExists("pagerduty_schedule.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "name", scheduleUpdated),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "description", "Managed by Terraform"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "time_zone", location),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.start", start),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.rotation_virtual_start", rotationVirtualStart),
				),
			},
		},
	})
}

func TestAccPagerDutySchedule_BasicWeek(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)
	schedule := fmt.Sprintf("tf-%s", acctest.RandString(5))
	scheduleUpdated := fmt.Sprintf("tf-%s", acctest.RandString(5))
	location := "Australia/Melbourne"
	start := "2020-05-12T20:00:00+10:00"
	rotationVirtualStart := "2020-05-12T20:00:00+10:00"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyScheduleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyScheduleConfigWeek(username, email, schedule, location, start, rotationVirtualStart),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleExists("pagerduty_schedule.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "name", schedule),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "description", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "time_zone", location),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.restriction.0.start_day_of_week", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.start", start),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.rotation_virtual_start", rotationVirtualStart),
				),
			},
			{
				Config: testAccCheckPagerDutyScheduleConfigWeekUpdated(username, email, scheduleUpdated, location, start, rotationVirtualStart),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleExists("pagerduty_schedule.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "name", scheduleUpdated),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "description", "Managed by Terraform"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "time_zone", location),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.restriction.0.start_day_of_week", "5"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.start", start),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.rotation_virtual_start", rotationVirtualStart),
				),
			},
		},
	})
}

func TestAccPagerDutySchedule_Multi(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)
	schedule := fmt.Sprintf("tf-%s", acctest.RandString(5))
	location := "Europe/Berlin"
	start := "2020-05-12T20:00:00+02:00"
	rotationVirtualStart := "2020-05-12T20:00:00+02:00"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyScheduleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyScheduleConfigMulti(username, email, schedule, location, start, rotationVirtualStart),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleExists("pagerduty_schedule.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "name", schedule),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "description", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "time_zone", location),

					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.#", "3"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.restriction.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.restriction.0.duration_seconds", "32101"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.restriction.0.start_time_of_day", "08:00:00"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.rotation_turn_length_seconds", "86400"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.users.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.start", start),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.rotation_virtual_start", rotationVirtualStart),

					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.1.name", "bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.1.restriction.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.1.restriction.0.duration_seconds", "32101"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.1.restriction.0.start_time_of_day", "08:00:00"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.1.restriction.0.start_day_of_week", "5"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.1.rotation_turn_length_seconds", "86400"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.1.users.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.1.start", start),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.1.rotation_virtual_start", rotationVirtualStart),

					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.2.name", "foobar"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.2.restriction.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.2.restriction.0.duration_seconds", "32101"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.2.restriction.0.start_time_of_day", "08:00:00"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.2.restriction.0.start_day_of_week", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.2.rotation_turn_length_seconds", "86400"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.2.users.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.2.start", start),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.2.rotation_virtual_start", rotationVirtualStart),
				),
			},
		},
	})
}

func testAccCheckPagerDutyScheduleDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*pagerduty.Client)
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_schedule" {
			continue
		}

		_, err := client.GetSchedule(r.Primary.ID, pagerduty.GetScheduleOptions{})

		if err == nil {
			return fmt.Errorf("Schedule still exists")
		}

	}
	return nil
}

func testAccCheckPagerDutyScheduleExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Schedule ID is set")
		}

		client := testAccProvider.Meta().(*pagerduty.Client)

		found, err := client.GetSchedule(rs.Primary.ID, pagerduty.GetScheduleOptions{})
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Schedule not found: %v - %v", rs.Primary.ID, found)
		}

		return nil
	}
}

func testAccCheckPagerDutyScheduleConfig(username, email, schedule, location, start, rotationVirtualStart string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_schedule" "foo" {
  name = "%s"

  time_zone   = "%s"
  description = "foo"

  layer {
    name                         = "foo"
    start                        = "%s"
    rotation_virtual_start       = "%s"
    rotation_turn_length_seconds = 86400
    users                        = ["${pagerduty_user.foo.id}"]

    restriction {
      type              = "daily_restriction"
      start_time_of_day = "08:00:00"
      duration_seconds  = 32101
    }
  }
}
`, username, email, schedule, location, start, rotationVirtualStart)
}

func testAccCheckPagerDutyScheduleConfigUpdated(username, email, schedule, location, start, rotationVirtualStart string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name        = "%s"
  email       = "%s"
}

resource "pagerduty_schedule" "foo" {
  name = "%s"

  time_zone = "%s"

  layer {
    name                         = "foo"
    start                        = "%s"
    rotation_virtual_start       = "%s"
    rotation_turn_length_seconds = 86400
    users                        = ["${pagerduty_user.foo.id}"]

    restriction {
      type              = "daily_restriction"
      start_time_of_day = "08:00:00"
      duration_seconds  = 32101
    }
  }
}
`, username, email, schedule, location, start, rotationVirtualStart)
}

func testAccCheckPagerDutyScheduleConfigWeek(username, email, schedule, location, start, rotationVirtualStart string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_schedule" "foo" {
  name = "%s"

  time_zone   = "%s"
  description = "foo"

  layer {
    name                         = "foo"
    start                        = "%s"
    rotation_virtual_start       = "%s"
    rotation_turn_length_seconds = 86400
    users                        = ["${pagerduty_user.foo.id}"]

    restriction {
      type              = "weekly_restriction"
      start_time_of_day = "08:00:00"
			start_day_of_week = 1
      duration_seconds  = 32101
    }
  }
}
`, username, email, schedule, location, start, rotationVirtualStart)
}

func testAccCheckPagerDutyScheduleConfigWeekUpdated(username, email, schedule, location, start, rotationVirtualStart string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name        = "%s"
  email       = "%s"
}

resource "pagerduty_schedule" "foo" {
  name = "%s"

  time_zone = "%s"

  layer {
    name                         = "foo"
    start                        = "%s"
    rotation_virtual_start       = "%s"
    rotation_turn_length_seconds = 86400
    users                        = ["${pagerduty_user.foo.id}"]

		restriction {
      type              = "weekly_restriction"
      start_time_of_day = "08:00:00"
			start_day_of_week = 5
      duration_seconds  = 32101
    }
  }
}
`, username, email, schedule, location, start, rotationVirtualStart)
}

func testAccCheckPagerDutyScheduleConfigMulti(username, email, schedule, location, start, rotationVirtualStart string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name        = "%s"
  email       = "%s"
}

resource "pagerduty_schedule" "foo" {
  name = "%s"

  time_zone   = "%s"
  description = "foo"

  layer {
    name                         = "foo"
    start                        = "%[5]v"
    rotation_virtual_start       = "%[6]v"
    rotation_turn_length_seconds = 86400
    users                        = ["${pagerduty_user.foo.id}"]

    restriction {
      type              = "daily_restriction"
      start_time_of_day = "08:00:00"
      duration_seconds  = 32101
    }
  }

  layer {
    name                         = "bar"
    start                        = "%[5]v"
    rotation_virtual_start       = "%[6]v"
    rotation_turn_length_seconds = 86400
    users                        = ["${pagerduty_user.foo.id}"]

    restriction {
      type              = "weekly_restriction"
      start_time_of_day = "08:00:00"
			start_day_of_week = 5
      duration_seconds  = 32101
    }
  }

  layer {
    name                         = "foobar"
    start                        = "%[5]v"
    rotation_virtual_start       = "%[6]v"
    rotation_turn_length_seconds = 86400
    users                        = ["${pagerduty_user.foo.id}"]

    restriction {
      type              = "weekly_restriction"
      start_time_of_day = "08:00:00"
			start_day_of_week = 1
      duration_seconds  = 32101
    }
  }
}
`, username, email, schedule, location, start, rotationVirtualStart)
}
