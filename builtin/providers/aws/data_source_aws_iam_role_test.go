package aws

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSDataSourceIAMRole_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsIAMRoleConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.aws_iam_role.test", "role_id"),
					resource.TestCheckResourceAttr("data.aws_iam_role.test", "assume_role_policy_document", "%7B%22Version%22%3A%222012-10-17%22%2C%22Statement%22%3A%5B%7B%22Sid%22%3A%22%22%2C%22Effect%22%3A%22Allow%22%2C%22Principal%22%3A%7B%22Service%22%3A%22ec2.amazonaws.com%22%7D%2C%22Action%22%3A%22sts%3AAssumeRole%22%7D%5D%7D"),
					resource.TestCheckResourceAttr("data.aws_iam_role.test", "path", "/testpath/"),
					resource.TestCheckResourceAttr("data.aws_iam_role.test", "role_name", "TestRole"),
					resource.TestMatchResourceAttr("data.aws_iam_role.test", "arn", regexp.MustCompile("^arn:aws:iam::[0-9]{12}:role/testpath/TestRole$")),
				),
			},
		},
	})
}

const testAccAwsIAMRoleConfig = `
provider "aws" {
	region = "us-east-1"
}

resource "aws_iam_role" "test_role" {
  name = "TestRole"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF

  path = "/testpath/"
}

data "aws_iam_role" "test" {
	  role_name = "${aws_iam_role.test_role.name}"
}
`
