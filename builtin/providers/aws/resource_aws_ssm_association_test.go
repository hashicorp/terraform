package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSSMAssociation_basic(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMAssociationBasicConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMAssociationExists("aws_ssm_association.foo"),
				),
			},
		},
	})
}

func TestAccAWSSSMAssociation_withTargets(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMAssociationBasicConfigWithTargets(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMAssociationExists("aws_ssm_association.foo"),
				),
			},
		},
	})
}

func testAccCheckAWSSSMAssociationExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SSM Assosciation ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ssmconn

		_, err := conn.DescribeAssociation(&ssm.DescribeAssociationInput{
			AssociationId: aws.String(rs.Primary.Attributes["association_id"]),
		})

		if err != nil {
			if wserr, ok := err.(awserr.Error); ok && wserr.Code() == "AssociationDoesNotExist" {
				return nil
			}
			return err
		}

		return nil
	}
}

func testAccCheckAWSSSMAssociationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ssmconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ssm_association" {
			continue
		}

		out, err := conn.DescribeAssociation(&ssm.DescribeAssociationInput{
			AssociationId: aws.String(rs.Primary.Attributes["association_id"]),
		})

		if err != nil {
			if wserr, ok := err.(awserr.Error); ok && wserr.Code() == "AssociationDoesNotExist" {
				return nil
			}
			return err
		}

		if out != nil {
			return fmt.Errorf("Expected AWS SSM Association to be gone, but was still found")
		}
	}

	return fmt.Errorf("Default error in SSM Association Test")
}

func testAccAWSSSMAssociationBasicConfigWithTargets(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_document" "foo_document" {
  name = "test_document_association-%s",
  document_type = "Command"
  content = <<DOC
  {
    "schemaVersion": "1.2",
    "description": "Check ip configuration of a Linux instance.",
    "parameters": {

    },
    "runtimeConfig": {
      "aws:runShellScript": {
        "properties": [
          {
            "id": "0.aws:runShellScript",
            "runCommand": ["ifconfig"]
          }
        ]
      }
    }
  }
DOC
}

resource "aws_ssm_association" "foo" {
  name = "${aws_ssm_document.foo_document.name}",
  targets {
    key = "tag:Name"
    values = ["acceptanceTest"]
  }
}`, rName)
}

func testAccAWSSSMAssociationBasicConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_security_group" "tf_test_foo" {
  name = "tf_test_foo-%s"
  description = "foo"
  ingress {
    protocol = "icmp"
    from_port = -1
    to_port = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "foo" {
  ami = "ami-4fccb37f"
  availability_zone = "us-west-2a"
  instance_type = "m1.small"
  security_groups = ["${aws_security_group.tf_test_foo.name}"]
}

resource "aws_ssm_document" "foo_document" {
  name    = "test_document_association-%s",
	document_type = "Command"
  content = <<DOC
  {
    "schemaVersion": "1.2",
    "description": "Check ip configuration of a Linux instance.",
    "parameters": {

    },
    "runtimeConfig": {
      "aws:runShellScript": {
        "properties": [
          {
            "id": "0.aws:runShellScript",
            "runCommand": ["ifconfig"]
          }
        ]
      }
    }
  }
DOC
}

resource "aws_ssm_association" "foo" {
  name        = "test_document_association-%s",
  instance_id = "${aws_instance.foo.id}"
}
`, rName, rName, rName)
}
