package statuscake

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/DreamItGetIT/statuscake"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccStatusCake_basic(t *testing.T) {
	var test statuscake.Test

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTestCheckDestroy(&test),
		Steps: []resource.TestStep{
			{
				Config: testAccTestConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccTestCheckExists("statuscake_test.google", &test),
					testAccTestCheckAttributes("statuscake_test.google", &test),
				),
			},
		},
	})
}

func TestAccStatusCake_tcp(t *testing.T) {
	var test statuscake.Test

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTestCheckDestroy(&test),
		Steps: []resource.TestStep{
			{
				Config: testAccTestConfig_tcp,
				Check: resource.ComposeTestCheckFunc(
					testAccTestCheckExists("statuscake_test.google", &test),
					testAccTestCheckAttributes("statuscake_test.google", &test),
				),
			},
		},
	})
}

func TestAccStatusCake_withUpdate(t *testing.T) {
	var test statuscake.Test

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTestCheckDestroy(&test),
		Steps: []resource.TestStep{
			{
				Config: testAccTestConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccTestCheckExists("statuscake_test.google", &test),
				),
			},

			{
				Config: testAccTestConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccTestCheckExists("statuscake_test.google", &test),
					testAccTestCheckAttributes("statuscake_test.google", &test),
					resource.TestCheckResourceAttr("statuscake_test.google", "check_rate", "500"),
					resource.TestCheckResourceAttr("statuscake_test.google", "paused", "true"),
					resource.TestCheckResourceAttr("statuscake_test.google", "timeout", "40"),
					resource.TestCheckResourceAttr("statuscake_test.google", "contact_id", "0"),
					resource.TestCheckResourceAttr("statuscake_test.google", "confirmations", "0"),
					resource.TestCheckResourceAttr("statuscake_test.google", "trigger_rate", "20"),
				),
			},
		},
	})
}

func testAccTestCheckExists(rn string, test *statuscake.Test) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("TestID not set")
		}

		client := testAccProvider.Meta().(*statuscake.Client)
		testId, parseErr := strconv.Atoi(rs.Primary.ID)
		if parseErr != nil {
			return fmt.Errorf("error in statuscake test CheckExists: %s", parseErr)
		}

		gotTest, err := client.Tests().Detail(testId)
		if err != nil {
			return fmt.Errorf("error getting test: %s", err)
		}

		*test = *gotTest

		return nil
	}
}

func testAccTestCheckAttributes(rn string, test *statuscake.Test) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		attrs := s.RootModule().Resources[rn].Primary.Attributes

		check := func(key, stateValue, testValue string) error {
			if testValue != stateValue {
				return fmt.Errorf("different values for %s in state (%s) and in statuscake (%s)",
					key, stateValue, testValue)
			}
			return nil
		}

		for key, value := range attrs {
			var err error

			switch key {
			case "website_name":
				err = check(key, value, test.WebsiteName)
			case "website_url":
				err = check(key, value, test.WebsiteURL)
			case "check_rate":
				err = check(key, value, strconv.Itoa(test.CheckRate))
			case "test_type":
				err = check(key, value, test.TestType)
			case "paused":
				err = check(key, value, strconv.FormatBool(test.Paused))
			case "timeout":
				err = check(key, value, strconv.Itoa(test.Timeout))
			case "contact_id":
				err = check(key, value, strconv.Itoa(test.ContactID))
			case "confirmations":
				err = check(key, value, strconv.Itoa(test.Confirmation))
			case "trigger_rate":
				err = check(key, value, strconv.Itoa(test.TriggerRate))
			}

			if err != nil {
				return err
			}
		}
		return nil
	}
}

func testAccTestCheckDestroy(test *statuscake.Test) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*statuscake.Client)
		err := client.Tests().Delete(test.TestID)
		if err == nil {
			return fmt.Errorf("test still exists")
		}

		return nil
	}
}

const testAccTestConfig_basic = `
resource "statuscake_test" "google" {
	website_name = "google.com"
	website_url = "www.google.com"
	test_type = "HTTP"
	check_rate = 300
	timeout = 10
	contact_id = 43402
	confirmations = 1
	trigger_rate = 10
}
`

const testAccTestConfig_update = `
resource "statuscake_test" "google" {
	website_name = "google.com"
	website_url = "www.google.com"
	test_type = "HTTP"
	check_rate = 500
	paused = true
	trigger_rate = 20
}
`

const testAccTestConfig_tcp = `
resource "statuscake_test" "google" {
	website_name = "google.com"
	website_url = "www.google.com"
	test_type = "TCP"
	check_rate = 300
	timeout = 10
	contact_id = 43402
	confirmations = 1
	port = 80
}
`
