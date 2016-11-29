package statuscake

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/DreamItGetIT/statuscake"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// check to ensure that contact group id is provided before running
// tests on it.
func testAccContactGroupPreCheck(t *testing.T, testAlt bool) {
	if v := os.Getenv("CONTACT_GROUP"); v == "" {
		t.Fatal("CONTACT_GROUP must be set for contact group acceptance tests")
	}
	if testAlt {
		if v := os.Getenv("ALT_CONTACT_GROUP"); v == "" {
			t.Fatal("ALT_CONTACT_GROUP must be set for contact group acceptance tests")
		}
	}
}

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
					resource.TestCheckResourceAttr("statuscake_test.google", "contact_id", "23456"),
				),
			},
		},
	})
}

func TestAccStatusCake_contactGroup_basic(t *testing.T) {
	var test statuscake.Test

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccContactGroupPreCheck(t, false)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccTestCheckDestroy(&test),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccTestConfig_contactGroup,
				Check: resource.ComposeTestCheckFunc(
					testAccTestCheckExists("statuscake_test.google", &test),
				),
			},
		},
	})
}

func TestAccStatusCake_contactGroup_withUpdate(t *testing.T) {
	var test statuscake.Test
	var altContactGroup = os.Getenv("ALT_CONTACT_GROUP")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccContactGroupPreCheck(t, true)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccTestCheckDestroy(&test),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccTestConfig_contactGroup,
				Check: resource.ComposeTestCheckFunc(
					testAccTestCheckExists("statuscake_test.google", &test),
				),
			},
			// make sure to creat
			resource.TestStep{
				Config: testAccTestConfig_contactGroup_update,
				Check: resource.ComposeTestCheckFunc(
					testAccTestCheckExists("statuscake_test.google", &test),
					resource.TestCheckResourceAttr("statuscake_test.google", "contact_id", altContactGroup),
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
	contact_id = 12345
}
`

const testAccTestConfig_update = `
resource "statuscake_test" "google" {
	website_name = "google.com"
	website_url = "www.google.com"
	test_type = "HTTP"
	check_rate = 500
	paused = true
	contact_id = 23456
}
`

var testAccTestConfig_contactGroup string = `` +
	`resource "statuscake_test" "google" {
  		website_name = "google.com"
  		website_url = "www.google.com"
  		test_type = "HTTP"
 		check_rate = 300
		contact_id = ` + os.Getenv("CONTACT_GROUP") + `
	}`

var testAccTestConfig_contactGroup_update string = `` +
	`resource "statuscake_test" "google" {
  		website_name = "google.com"
  		website_url = "www.google.com"
  		test_type = "HTTP"
 		check_rate = 300
		contact_id = ` + os.Getenv("ALT_CONTACT_GROUP") + `
	}`
