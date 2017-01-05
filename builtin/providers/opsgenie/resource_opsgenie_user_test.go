package opsgenie

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/opsgenie/opsgenie-go-sdk/user"
)

func TestAccOpsGenieUserUsername_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "hello",
			ErrCount: 0,
		},
		{
			Value:    acctest.RandString(99),
			ErrCount: 0,
		},
		{
			Value:    acctest.RandString(100),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateOpsGenieUserUsername(tc.Value, "opsgenie_team")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the OpsGenie User Username Validation to trigger a validation error: %v", errors)
		}
	}
}

func TestAccOpsGenieUserFullName_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "hello",
			ErrCount: 0,
		},
		{
			Value:    acctest.RandString(100),
			ErrCount: 0,
		},
		{
			Value:    acctest.RandString(511),
			ErrCount: 0,
		},
		{
			Value:    acctest.RandString(512),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateOpsGenieUserFullName(tc.Value, "opsgenie_team")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the OpsGenie User Full Name Validation to trigger a validation error: %v", errors)
		}
	}
}

func TestAccOpsGenieUserRole_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "hello",
			ErrCount: 0,
		},
		{
			Value:    acctest.RandString(100),
			ErrCount: 0,
		},
		{
			Value:    acctest.RandString(511),
			ErrCount: 0,
		},
		{
			Value:    acctest.RandString(512),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateOpsGenieUserRole(tc.Value, "opsgenie_team")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the OpsGenie User Role Validation to trigger a validation error: %v", errors)
		}
	}
}

func TestAccOpsGenieUser_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOpsGenieUser_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpsGenieUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpsGenieUserExists("opsgenie_user.test"),
				),
			},
		},
	})
}

func TestAccOpsGenieUser_complete(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOpsGenieUser_complete, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpsGenieUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpsGenieUserExists("opsgenie_user.test"),
				),
			},
		},
	})
}

func testCheckOpsGenieUserDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*OpsGenieClient).users

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opsgenie_user" {
			continue
		}

		req := user.GetUserRequest{
			Id: rs.Primary.Attributes["id"],
		}

		result, _ := client.Get(req)
		if result != nil {
			return fmt.Errorf("User still exists:\n%#v", result)
		}
	}

	return nil
}

func testCheckOpsGenieUserExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		id := rs.Primary.Attributes["id"]
		username := rs.Primary.Attributes["username"]

		client := testAccProvider.Meta().(*OpsGenieClient).users

		req := user.GetUserRequest{
			Id: rs.Primary.Attributes["id"],
		}

		result, _ := client.Get(req)
		if result == nil {
			return fmt.Errorf("Bad: User %q (username: %q) does not exist", id, username)
		}

		return nil
	}
}

var testAccOpsGenieUser_basic = `
resource "opsgenie_user" "test" {
  username  = "acctest-%d@example.tld"
  full_name = "Acceptance Test User"
  role      = "User"
}
`

var testAccOpsGenieUser_complete = `
resource "opsgenie_user" "test" {
  username  = "acctest-%d@example.tld"
  full_name = "Acceptance Test User"
  role      = "User"
  locale    = "en_GB"
  timezone  = "Etc/GMT"
}
`
