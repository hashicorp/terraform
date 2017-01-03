package pagerduty

import (
	"fmt"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPagerDutySchedule_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyScheduleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPagerDutyScheduleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleExists("pagerduty_schedule.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "description", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "time_zone", "Europe/Berlin"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.name", "foo"),
				),
			},
			resource.TestStep{
				Config: testAccCheckPagerDutyScheduleConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleExists("pagerduty_schedule.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "name", "bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "description", "Managed by Terraform"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "time_zone", "America/New_York"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.name", "foo"),
				),
			},
		},
	})
}

func TestAccPagerDutySchedule_BasicWeek(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyScheduleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPagerDutyScheduleConfigWeek,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleExists("pagerduty_schedule.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "description", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "time_zone", "Europe/Berlin"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.restriction.0.start_day_of_week", "1"),
				),
			},
			resource.TestStep{
				Config: testAccCheckPagerDutyScheduleConfigWeekUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleExists("pagerduty_schedule.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "name", "bar"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "description", "Managed by Terraform"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "time_zone", "America/New_York"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.#", "1"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.restriction.0.start_day_of_week", "5"),
				),
			},
		},
	})
}

func TestAccPagerDutySchedule_Multi(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyScheduleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPagerDutyScheduleConfigMulti,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleExists("pagerduty_schedule.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "description", "foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "time_zone", "America/New_York"),

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
						"pagerduty_schedule.foo", "layer.0.rotation_virtual_start", "2015-11-06T20:00:00-05:00"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.0.users.#", "1"),

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
						"pagerduty_schedule.foo", "layer.1.rotation_virtual_start", "2015-11-06T20:00:00-05:00"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.1.users.#", "1"),

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
						"pagerduty_schedule.foo", "layer.2.rotation_virtual_start", "2015-11-06T20:00:00-05:00"),
					resource.TestCheckResourceAttr(
						"pagerduty_schedule.foo", "layer.2.users.#", "1"),
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

const testAccCheckPagerDutyScheduleConfig = `
resource "pagerduty_user" "foo" {
  name  = "foo"
  email = "foo@bar.com"
}

resource "pagerduty_schedule" "foo" {
  name = "foo"

  time_zone   = "Europe/Berlin"
  description = "foo"

  layer {
    name                         = "foo"
    start                        = "2015-11-06T20:00:00-05:00"
    rotation_virtual_start       = "2015-11-06T20:00:00-05:00"
    rotation_turn_length_seconds = 86400
    users                        = ["${pagerduty_user.foo.id}"]

    restriction {
      type              = "daily_restriction"
      start_time_of_day = "08:00:00"
      duration_seconds  = 32101
    }
  }
}
`

const testAccCheckPagerDutyScheduleConfigUpdated = `
resource "pagerduty_user" "foo" {
  name        = "foo"
  email       = "foo@bar.com"
}

resource "pagerduty_schedule" "foo" {
  name = "bar"

  time_zone = "America/New_York"

  layer {
    name                         = "foo"
    start                        = "2015-11-06T20:00:00-05:00"
    rotation_virtual_start       = "2015-11-06T20:00:00-05:00"
    rotation_turn_length_seconds = 86400
    users                        = ["${pagerduty_user.foo.id}"]

    restriction {
      type              = "daily_restriction"
      start_time_of_day = "08:00:00"
      duration_seconds  = 32101
    }
  }
}
`

const testAccCheckPagerDutyScheduleConfigWeek = `
resource "pagerduty_user" "foo" {
  name  = "foo"
  email = "foo@bar.com"
}

resource "pagerduty_schedule" "foo" {
  name = "foo"

  time_zone   = "Europe/Berlin"
  description = "foo"

  layer {
    name                         = "foo"
    start                        = "2015-11-06T20:00:00-05:00"
    rotation_virtual_start       = "2015-11-06T20:00:00-05:00"
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
`

const testAccCheckPagerDutyScheduleConfigWeekUpdated = `
resource "pagerduty_user" "foo" {
  name        = "foo"
  email       = "foo@bar.com"
}

resource "pagerduty_schedule" "foo" {
  name = "bar"

  time_zone = "America/New_York"

  layer {
    name                         = "foo"
    start                        = "2015-11-06T20:00:00-05:00"
    rotation_virtual_start       = "2015-11-06T20:00:00-05:00"
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
`

const testAccCheckPagerDutyScheduleConfigMulti = `
resource "pagerduty_user" "foo" {
  name        = "foo"
  email       = "foo@bar.com"
}

resource "pagerduty_schedule" "foo" {
  name = "foo"

  time_zone   = "America/New_York"
  description = "foo"

  layer {
    name                         = "foo"
    start                        = "2015-11-06T20:00:00-05:00"
    rotation_virtual_start       = "2015-11-06T20:00:00-05:00"
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
    start                        = "2015-11-06T20:00:00-05:00"
    rotation_virtual_start       = "2015-11-06T20:00:00-05:00"
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
    start                        = "2015-11-06T20:00:00-05:00"
    rotation_virtual_start       = "2015-11-06T20:00:00-05:00"
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
`
