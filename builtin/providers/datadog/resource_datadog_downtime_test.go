package datadog

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"gopkg.in/zorkian/go-datadog-api.v2"
)

func TestAccDatadogDowntime_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogDowntimeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogDowntimeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogDowntimeExists("datadog_downtime.foo"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "scope.0", "*"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "start", "1735707600"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "end", "1735765200"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.type", "days"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.period", "1"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "message", "Example Datadog downtime message."),
				),
			},
		},
	})
}

func TestAccDatadogDowntime_BasicMultiScope(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogDowntimeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogDowntimeConfigMultiScope,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogDowntimeExists("datadog_downtime.foo"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "scope.0", "host:A"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "scope.1", "host:B"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "start", "1735707600"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "end", "1735765200"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.type", "days"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.period", "1"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "message", "Example Datadog downtime message."),
				),
			},
		},
	})
}

func TestAccDatadogDowntime_BasicNoRecurrence(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogDowntimeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogDowntimeConfigNoRecurrence,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogDowntimeExists("datadog_downtime.foo"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "scope.0", "host:NoRecurrence"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "start", "1735707600"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "end", "1735765200"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "message", "Example Datadog downtime message."),
				),
			},
		},
	})
}

func TestAccDatadogDowntime_BasicUntilDateRecurrence(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogDowntimeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogDowntimeConfigUntilDateRecurrence,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogDowntimeExists("datadog_downtime.foo"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "scope.0", "host:UntilDateRecurrence"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "start", "1735707600"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "end", "1735765200"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.type", "days"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.period", "1"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.until_date", "1736226000"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "message", "Example Datadog downtime message."),
				),
			},
		},
	})
}

func TestAccDatadogDowntime_BasicUntilOccurrencesRecurrence(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogDowntimeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogDowntimeConfigUntilOccurrencesRecurrence,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogDowntimeExists("datadog_downtime.foo"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "scope.0", "host:UntilOccurrencesRecurrence"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "start", "1735707600"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "end", "1735765200"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.type", "days"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.period", "1"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.until_occurrences", "5"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "message", "Example Datadog downtime message."),
				),
			},
		},
	})
}

func TestAccDatadogDowntime_WeekDayRecurring(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogDowntimeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogDowntimeConfigWeekDaysRecurrence,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogDowntimeExists("datadog_downtime.foo"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "scope.0", "WeekDaysRecurrence"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "start", "1735646400"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "end", "1735732799"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.type", "weeks"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.period", "1"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.week_days.0", "Sat"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.week_days.1", "Sun"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "message", "Example Datadog downtime message."),
				),
			},
		},
	})
}

func TestAccDatadogDowntime_Updated(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogDowntimeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogDowntimeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogDowntimeExists("datadog_downtime.foo"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "scope.0", "*"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "start", "1735707600"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "end", "1735765200"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.type", "days"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.period", "1"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "message", "Example Datadog downtime message."),
				),
			},
			resource.TestStep{
				Config: testAccCheckDatadogDowntimeConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogDowntimeExists("datadog_downtime.foo"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "scope.0", "Updated"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "start", "1735707600"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "end", "1735765200"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.type", "days"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.period", "3"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "message", "Example Datadog downtime message."),
				),
			},
		},
	})
}

func TestAccDatadogDowntime_TrimWhitespace(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogDowntimeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogDowntimeConfigWhitespace,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogDowntimeExists("datadog_downtime.foo"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "scope.0", "host:Whitespace"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "start", "1735707600"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "end", "1735765200"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.type", "days"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "recurrence.0.period", "1"),
					resource.TestCheckResourceAttr(
						"datadog_downtime.foo", "message", "Example Datadog downtime message."),
				),
			},
		},
	})
}

func testAccCheckDatadogDowntimeDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*datadog.Client)

	if err := datadogDowntimeDestroyHelper(s, client); err != nil {
		return err
	}
	return nil
}

func testAccCheckDatadogDowntimeExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*datadog.Client)
		if err := datadogDowntimeExistsHelper(s, client); err != nil {
			return err
		}
		return nil
	}
}

const testAccCheckDatadogDowntimeConfig = `
resource "datadog_downtime" "foo" {
  scope = ["*"]
  start = 1735707600
  end   = 1735765200

  recurrence {
    type   = "days"
    period = 1
  }

  message = "Example Datadog downtime message."
}
`

const testAccCheckDatadogDowntimeConfigMultiScope = `
resource "datadog_downtime" "foo" {
  scope = ["host:A", "host:B"]
  start = 1735707600
  end   = 1735765200

  recurrence {
    type   = "days"
    period = 1
  }

  message = "Example Datadog downtime message."
}
`

const testAccCheckDatadogDowntimeConfigNoRecurrence = `
resource "datadog_downtime" "foo" {
  scope = ["host:NoRecurrence"]
  start = 1735707600
  end   = 1735765200
  message = "Example Datadog downtime message."
}
`

const testAccCheckDatadogDowntimeConfigUntilDateRecurrence = `
resource "datadog_downtime" "foo" {
  scope = ["host:UntilDateRecurrence"]
  start = 1735707600
  end   = 1735765200

  recurrence {
    type       = "days"
    period     = 1
	until_date = 1736226000
  }

  message = "Example Datadog downtime message."
}
`

const testAccCheckDatadogDowntimeConfigUntilOccurrencesRecurrence = `
resource "datadog_downtime" "foo" {
  scope = ["host:UntilOccurrencesRecurrence"]
  start = 1735707600
  end   = 1735765200

  recurrence {
    type              = "days"
    period            = 1
	until_occurrences = 5
  }

  message = "Example Datadog downtime message."
}
`

const testAccCheckDatadogDowntimeConfigWeekDaysRecurrence = `
resource "datadog_downtime" "foo" {
  scope = ["WeekDaysRecurrence"]
  start = 1735646400
  end   = 1735732799

  recurrence {
    period    = 1
	type      = "weeks"
	week_days = ["Sat", "Sun"]
  }

  message = "Example Datadog downtime message."
}
`

const testAccCheckDatadogDowntimeConfigUpdated = `
resource "datadog_downtime" "foo" {
  scope = ["Updated"]
  start = 1735707600
  end   = 1735765200

  recurrence {
    type   = "days"
    period = 3
  }

  message = "Example Datadog downtime message."
}
`

const testAccCheckDatadogDowntimeConfigWhitespace = `
resource "datadog_downtime" "foo" {
  scope = ["host:Whitespace"]
  start = 1735707600
  end   = 1735765200

  recurrence {
    type   = "days"
    period = 1
  }

  message = <<EOF
Example Datadog downtime message.
EOF
}
`

func TestResourceDatadogDowntimeRecurrenceTypeValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "daily",
			ErrCount: 1,
		},
		{
			Value:    "days",
			ErrCount: 0,
		},
		{
			Value:    "days,weeks",
			ErrCount: 1,
		},
		{
			Value:    "months",
			ErrCount: 0,
		},
		{
			Value:    "years",
			ErrCount: 0,
		},
		{
			Value:    "weeks",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateDatadogDowntimeRecurrenceType(tc.Value, "datadog_downtime_recurrence_type")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected Datadog Downtime Recurrence Type validation to trigger %d error(s) for value %q - instead saw %d",
				tc.ErrCount, tc.Value, len(errors))
		}
	}
}

func TestResourceDatadogDowntimeRecurrenceWeekDaysValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Mon",
			ErrCount: 0,
		},
		{
			Value:    "Mon,",
			ErrCount: 1,
		},
		{
			Value:    "Monday",
			ErrCount: 1,
		},
		{
			Value:    "mon",
			ErrCount: 1,
		},
		{
			Value:    "mon,",
			ErrCount: 1,
		},
		{
			Value:    "monday",
			ErrCount: 1,
		},
		{
			Value:    "mon,Tue",
			ErrCount: 1,
		},
		{
			Value:    "Mon,tue",
			ErrCount: 1,
		},
		{
			Value:    "Mon,Tue",
			ErrCount: 1,
		},
		{
			Value:    "Mon, Tue",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateDatadogDowntimeRecurrenceWeekDays(tc.Value, "datadog_downtime_recurrence_week_days")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected Datadog Downtime Recurrence Week Days validation to trigger %d error(s) for value %q - instead saw %d",
				tc.ErrCount, tc.Value, len(errors))
		}
	}
}

func datadogDowntimeDestroyHelper(s *terraform.State, client *datadog.Client) error {
	for _, r := range s.RootModule().Resources {
		id, _ := strconv.Atoi(r.Primary.ID)
		dt, err := client.GetDowntime(id)

		if err != nil {
			if strings.Contains(err.Error(), "404 Not Found") {
				continue
			}
			return fmt.Errorf("Received an error retrieving downtime %s", err)
		}

		// Datadog only cancels downtime on DELETE
		if !dt.GetActive() {
			continue
		}
		return fmt.Errorf("Downtime still exists")
	}
	return nil
}

func datadogDowntimeExistsHelper(s *terraform.State, client *datadog.Client) error {
	for _, r := range s.RootModule().Resources {
		id, _ := strconv.Atoi(r.Primary.ID)
		if _, err := client.GetDowntime(id); err != nil {
			return fmt.Errorf("Received an error retrieving downtime %s", err)
		}
	}
	return nil
}
