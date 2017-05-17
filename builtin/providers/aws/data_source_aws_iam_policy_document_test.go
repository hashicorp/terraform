package aws

import (
	"testing"

	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIAMPolicyDocument(t *testing.T) {
	// This really ought to be able to be a unit test rather than an
	// acceptance test, but just instantiating the AWS provider requires
	// some AWS API calls, and so this needs valid AWS credentials to work.
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSIAMPolicyDocumentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStateValue(
						"data.aws_iam_policy_document.test",
						"json",
						testAccAWSIAMPolicyDocumentExpectedJSON,
					),
				),
			},
		},
	})
}

func testAccCheckStateValue(id, name, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("Not found: %s", id)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		v := rs.Primary.Attributes[name]
		if v != value {
			return fmt.Errorf(
				"Value for %s is %s, not %s", name, v, value)
		}

		return nil
	}
}

var testAccAWSIAMPolicyDocumentConfig = `
data "aws_iam_policy_document" "test" {
    policy_id = "policy_id"
    statement {
    	sid = "1"
        actions = [
            "s3:ListAllMyBuckets",
            "s3:GetBucketLocation",
        ]
        resources = [
            "arn:aws:s3:::*",
        ]
    }

    statement {
        actions = [
            "s3:ListBucket",
        ]
        resources = [
            "arn:aws:s3:::foo",
        ]
        condition {
            test = "StringLike"
            variable = "s3:prefix"
            values = [
                "home/",
                "home/&{aws:username}/",
            ]
        }

        not_principals {
            type = "AWS"
            identifiers = ["arn:blahblah:example"]
        }
    }

    statement {
        actions = [
            "s3:*",
        ]
        resources = [
            "arn:aws:s3:::foo/home/&{aws:username}",
            "arn:aws:s3:::foo/home/&{aws:username}/*",
        ]
        principals {
            type = "AWS"
            identifiers = ["arn:blahblah:example"]
        }
    }

    statement {
        effect = "Deny"
        not_actions = ["s3:*"]
        not_resources = ["arn:aws:s3:::*"]
    }

    # Normalization of wildcard principals
    statement {
        effect = "Allow"
        actions = ["kinesis:*"]
        principals {
            type = "AWS"
            identifiers = ["*"]
        }
    }
    statement {
        effect = "Allow"
        actions = ["firehose:*"]
        principals {
            type = "*"
            identifiers = ["*"]
        }
    }

}
`

var testAccAWSIAMPolicyDocumentExpectedJSON = `{
  "Version": "2012-10-17",
  "Id": "policy_id",
  "Statement": [
    {
      "Sid": "1",
      "Effect": "Allow",
      "Action": [
        "s3:ListAllMyBuckets",
        "s3:GetBucketLocation"
      ],
      "Resource": "arn:aws:s3:::*"
    },
    {
      "Sid": "",
      "Effect": "Allow",
      "Action": "s3:ListBucket",
      "Resource": "arn:aws:s3:::foo",
      "NotPrincipal": {
        "AWS": "arn:blahblah:example"
      },
      "Condition": {
        "StringLike": {
          "s3:prefix": [
            "home/${aws:username}/",
            "home/"
          ]
        }
      }
    },
    {
      "Sid": "",
      "Effect": "Allow",
      "Action": "s3:*",
      "Resource": [
        "arn:aws:s3:::foo/home/${aws:username}/*",
        "arn:aws:s3:::foo/home/${aws:username}"
      ],
      "Principal": {
        "AWS": "arn:blahblah:example"
      }
    },
    {
      "Sid": "",
      "Effect": "Deny",
      "NotAction": "s3:*",
      "NotResource": "arn:aws:s3:::*"
    },
    {
      "Sid": "",
      "Effect": "Allow",
      "Action": "kinesis:*",
      "Principal": "*"
    },
    {
      "Sid": "",
      "Effect": "Allow",
      "Action": "firehose:*",
      "Principal": "*"
    }
  ]
}`
