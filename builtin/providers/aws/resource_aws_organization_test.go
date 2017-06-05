package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSOrganization_basic(t *testing.T) {
	var organization organizations.Organization

	feature_set := "CONSOLIDATED_BILLING"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSOrganizationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSOrganizationConfig(feature_set),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOrganizationExists("aws_organization.test", &organization),
				),
			},
		},
	})
}

func testAccCheckAWSOrganizationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).orgsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_organization" {
			continue
		}

		params := &organizations.DescribeOrganizationInput{}

		resp, err := conn.DescribeOrganization(params)

		if err != nil || resp == nil {
			return nil
		}

		if resp.Organization != nil {
			return fmt.Errorf("Bad: Organization still exists: %q", rs.Primary.ID)
		}
	}

	return nil

}

func testAccCheckAWSOrganizationExists(n string, a *organizations.Organization) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).orgsconn
		params := &organizations.DescribeOrganizationInput{}

		resp, err := conn.DescribeOrganization(params)

		if err != nil || resp == nil {
			return nil
		}

		if resp.Organization == nil {
			return fmt.Errorf("Bad: Organization %q does not exist", rs.Primary.ID)
		}

		a = resp.Organization

		return nil
	}
}

func testAccAWSOrganizationConfig(feature_set string) string {
	return fmt.Sprintf(`
resource "aws_organization" "test" {
  feature_set = "%s"
}
`, feature_set)
}
