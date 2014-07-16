package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func TestAccAwsSecurityGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSecurityGroupConfig,
				Check:  testAccCheckAWSSecurityGroupExists("aws_security_group.web"),
			},
		},
	})
}

func testAccCheckAWSSecurityGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.ec2conn

	for _, rs := range s.Resources {
		if rs.Type != "aws_security_group" {
			continue
		}

		sgs := []ec2.SecurityGroup{
			ec2.SecurityGroup{
				Id: rs.ID,
			},
		}

		// Retrieve our group
		resp, err := conn.SecurityGroups(sgs, nil)
		if err == nil {
			if len(resp.Groups) > 0 && resp.Groups[0].Id == rs.ID {
				return fmt.Errorf("Security Group (%s) still exists.", rs.ID)
			}

			return nil
		}

		ec2err, ok := err.(*ec2.Error)
		if !ok {
			return err
		}
		// Confirm error code is what we want
		if ec2err.Code != "InvalidGroup.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSSecurityGroupExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No Security Group is set")
		}

		conn := testAccProvider.ec2conn
		sgs := []ec2.SecurityGroup{
			ec2.SecurityGroup{
				Id: rs.ID,
			},
		}
		resp, err := conn.SecurityGroups(sgs, nil)
		if err != nil {
			return err
		}

		if len(resp.Groups) > 0 && resp.Groups[0].Id == rs.ID {
			return nil
		} else {
			return fmt.Errorf("Security Group not found")
		}

		return nil
	}
}

const testAccAWSSecurityGroupConfig = `
resource "aws_security_group" "web" {
    name = "terraform_acceptance_test_example"
    description = "Used in the terraform acceptance tests"

    ingress {
        protocol = "tcp"
        from_port = 80
        to_port = 8000
        cidr_blocks = ["10.0.0.0/0"]
    }
}
`
