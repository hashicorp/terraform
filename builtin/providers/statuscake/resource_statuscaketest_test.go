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
			resource.TestStep{
				Config: testAccTestConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccTestCheckExists("statuscake_test.google", &test),
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
			resource.TestStep{
				Config: testAccTestConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccTestCheckExists("statuscake_test.google", &test),
				),
			},

			resource.TestStep{
				Config: testAccTestConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccTestCheckExists("statuscake_test.google", &test),
					resource.TestCheckResourceAttr("statuscake_test.google", "check_rate", "500"),
					resource.TestCheckResourceAttr("statuscake_test.google", "paused", "true"),
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
			return fmt.Errorf("error getting project: %s", err)
		}

		*test = *gotTest

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
}
`

const testAccTestConfig_update = `
resource "statuscake_test" "google" {
  website_name = "google.com"
  website_url = "www.google.com"
  test_type = "HTTP"
  check_rate = 500
  paused = true
}
`
