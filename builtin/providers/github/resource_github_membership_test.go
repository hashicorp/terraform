package github

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGithubMembership_basic(t *testing.T) {
	var membership github.Membership

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubMembershipDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubMembershipConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubMembershipExists("github_membership.test_org_membership", &membership),
					testAccCheckGithubMembershipRoleState("github_membership.test_org_membership", &membership),
				),
			},
		},
	})
}

func TestAccGithubMembership_importBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGithubMembershipDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubMembershipConfig,
			},
			{
				ResourceName:      "github_membership.test_org_membership",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGithubMembershipDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*Organization).client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_membership" {
			continue
		}
		o, u := parseTwoPartID(rs.Primary.ID)

		membership, resp, err := conn.Organizations.GetOrgMembership(context.TODO(), u, o)

		if err == nil {
			if membership != nil &&
				buildTwoPartID(membership.Organization.Login, membership.User.Login) == rs.Primary.ID {
				return fmt.Errorf("Organization membership still exists")
			}
		}
		if resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}

func testAccCheckGithubMembershipExists(n string, membership *github.Membership) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No membership ID is set")
		}

		conn := testAccProvider.Meta().(*Organization).client
		o, u := parseTwoPartID(rs.Primary.ID)

		githubMembership, _, err := conn.Organizations.GetOrgMembership(context.TODO(), u, o)
		if err != nil {
			return err
		}
		*membership = *githubMembership
		return nil
	}
}

func testAccCheckGithubMembershipRoleState(n string, membership *github.Membership) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No membership ID is set")
		}

		conn := testAccProvider.Meta().(*Organization).client
		o, u := parseTwoPartID(rs.Primary.ID)

		githubMembership, _, err := conn.Organizations.GetOrgMembership(context.TODO(), u, o)
		if err != nil {
			return err
		}

		resourceRole := membership.Role
		actualRole := githubMembership.Role

		if *resourceRole != *actualRole {
			return fmt.Errorf("Membership role %v in resource does match actual state of %v", *resourceRole, *actualRole)
		}
		return nil
	}
}

var testAccGithubMembershipConfig string = fmt.Sprintf(`
  resource "github_membership" "test_org_membership" {
    username = "%s"
    role = "member"
  }
`, testUser)
