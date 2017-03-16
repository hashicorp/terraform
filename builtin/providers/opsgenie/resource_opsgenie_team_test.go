package opsgenie

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/opsgenie/opsgenie-go-sdk/team"
)

func TestAccOpsGenieTeamName_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "hello-world",
			ErrCount: 1,
		},
		{
			Value:    "hello_world",
			ErrCount: 0,
		},
		{
			Value:    "helloWorld",
			ErrCount: 0,
		},
		{
			Value:    "helloworld12",
			ErrCount: 0,
		},
		{
			Value:    "hello@world",
			ErrCount: 1,
		},
		{
			Value:    "qfvbdsbvipqdbwsbddbdcwqffewsqwcdw21ddwqwd3324120",
			ErrCount: 0,
		},
		{
			Value:    "qfvbdsbvipqdbwsbddbdcwqffewsqwcdw21ddwqwd33241202qfvbdsbvipqdbwsbddbdcwqffewsqwcdw21ddwqwd33241202",
			ErrCount: 0,
		},
		{
			Value:    "qfvbdsbvipqdbwsbddbdcwqfjjfewsqwcdw21ddwqwd3324120qfvbdsbvipqdbwsbddbdcwqfjjfewsqwcdw21ddwqwd3324120",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateOpsGenieTeamName(tc.Value, "opsgenie_team")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the OpsGenie Team Name to trigger a validation error: %v", errors)
		}
	}
}

func TestAccOpsGenieTeamRole_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "admin",
			ErrCount: 0,
		},
		{
			Value:    "user",
			ErrCount: 0,
		},
		{
			Value:    "custom",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateOpsGenieTeamRole(tc.Value, "opsgenie_team")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the OpsGenie Team Role to trigger a validation error")
		}
	}
}

func TestAccOpsGenieTeam_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOpsGenieTeam_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpsGenieTeamDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpsGenieTeamExists("opsgenie_team.test"),
				),
			},
		},
	})
}

func TestAccOpsGenieTeam_withEmptyDescription(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOpsGenieTeam_withEmptyDescription, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpsGenieTeamDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpsGenieTeamExists("opsgenie_team.test"),
				),
			},
		},
	})
}

func TestAccOpsGenieTeam_withUser(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOpsGenieTeam_withUser, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpsGenieTeamDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpsGenieTeamExists("opsgenie_team.test"),
				),
			},
		},
	})
}

func TestAccOpsGenieTeam_withUserComplete(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOpsGenieTeam_withUserComplete, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpsGenieTeamDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpsGenieTeamExists("opsgenie_team.test"),
				),
			},
		},
	})
}

func TestAccOpsGenieTeam_withMultipleUsers(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOpsGenieTeam_withMultipleUsers, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpsGenieTeamDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpsGenieTeamExists("opsgenie_team.test"),
				),
			},
		},
	})
}

func testCheckOpsGenieTeamDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*OpsGenieClient).teams

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opsgenie_team" {
			continue
		}

		req := team.GetTeamRequest{
			Id: rs.Primary.Attributes["id"],
		}

		result, _ := client.Get(req)
		if result != nil {
			return fmt.Errorf("Team still exists:\n%#v", result)
		}
	}

	return nil
}

func testCheckOpsGenieTeamExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		id := rs.Primary.Attributes["id"]
		name := rs.Primary.Attributes["name"]

		client := testAccProvider.Meta().(*OpsGenieClient).teams

		req := team.GetTeamRequest{
			Id: rs.Primary.Attributes["id"],
		}

		result, _ := client.Get(req)
		if result == nil {
			return fmt.Errorf("Bad: Team %q (name: %q) does not exist", id, name)
		}

		return nil
	}
}

var testAccOpsGenieTeam_basic = `
resource "opsgenie_team" "test" {
  name = "acctest%d"
}
`

var testAccOpsGenieTeam_withEmptyDescription = `
resource "opsgenie_team" "test" {
  name        = "acctest%d"
  description = ""
}
`

var testAccOpsGenieTeam_withUser = `
resource "opsgenie_user" "test" {
  username  = "acctest-%d@example.tld"
  full_name = "Acceptance Test User"
  role      = "User"
}

resource "opsgenie_team" "test" {
  name  = "acctest%d"
  member {
    username = "${opsgenie_user.test.username}"
  }
}
`

var testAccOpsGenieTeam_withUserComplete = `
resource "opsgenie_user" "test" {
  username  = "acctest-%d@example.tld"
  full_name = "Acceptance Test User"
  role      = "User"
}

resource "opsgenie_team" "test" {
  name        = "acctest%d"
  description = "Some exmaple description"
  member {
    username = "${opsgenie_user.test.username}"
    role     = "user"
  }
}
`

var testAccOpsGenieTeam_withMultipleUsers = `
resource "opsgenie_user" "first" {
  username  = "acctest-1-%d@example.tld"
  full_name = "First Acceptance Test User"
  role      = "User"
}
resource "opsgenie_user" "second" {
  username  = "acctest-2-%d@example.tld"
  full_name = "Second Acceptance Test User"
  role      = "User"
}

resource "opsgenie_team" "test" {
  name        = "acctest%d"
  description = "Some exmaple description"
  member {
    username = "${opsgenie_user.first.username}"
  }
  member {
    username = "${opsgenie_user.second.username}"
  }
}
`
