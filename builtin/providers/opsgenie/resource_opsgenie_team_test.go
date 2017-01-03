package opsgenie

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/opsgenie/opsgenie-go-sdk/team"
)

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
  name  = "acctest%d"
  member {
    username = "${opsgenie_user.test.username}"
    role     = "user"
  }
}
`
