package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIAMInstanceProfile_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsIamInstanceProfileConfig,
			},
		},
	})
}

func TestAccAWSIAMInstanceProfile_namePrefix(t *testing.T) {
	var conf iam.GetInstanceProfileOutput

	resource.Test(t, resource.TestCase{
		PreCheck:        func() { testAccPreCheck(t) },
		IDRefreshName:   "aws_iam_instance_profile.test",
		IDRefreshIgnore: []string{"name_prefix"},
		Providers:       testAccProviders,
		CheckDestroy:    testAccCheckAWSInstanceProfileDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSInstanceProfilePrefixNameConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSInstanceProfileExists("aws_iam_instance_profile.test", &conf),
					testAccCheckAWSInstanceProfileGeneratedNamePrefix(
						"aws_iam_instance_profile.test", "test-"),
				),
			},
		},
	})
}

func testAccCheckAWSInstanceProfileGeneratedNamePrefix(resource, prefix string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		r, ok := s.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("Resource not found")
		}
		name, ok := r.Primary.Attributes["name"]
		if !ok {
			return fmt.Errorf("Name attr not found: %#v", r.Primary.Attributes)
		}
		if !strings.HasPrefix(name, prefix) {
			return fmt.Errorf("Name: %q, does not have prefix: %q", name, prefix)
		}
		return nil
	}
}

func testAccCheckAWSInstanceProfileDestroy(s *terraform.State) error {
	iamconn := testAccProvider.Meta().(*AWSClient).iamconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iam_instance_profile" {
			continue
		}

		// Try to get role
		_, err := iamconn.GetInstanceProfile(&iam.GetInstanceProfileInput{
			InstanceProfileName: aws.String(rs.Primary.ID),
		})
		if err == nil {
			return fmt.Errorf("still exist.")
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "NoSuchEntity" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSInstanceProfileExists(n string, res *iam.GetInstanceProfileOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Instance Profile name is set")
		}

		iamconn := testAccProvider.Meta().(*AWSClient).iamconn

		resp, err := iamconn.GetInstanceProfile(&iam.GetInstanceProfileInput{
			InstanceProfileName: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return err
		}

		*res = *resp

		return nil
	}
}

const testAccAwsIamInstanceProfileConfig = `
resource "aws_iam_role" "test" {
	name = "test"
	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"ec2.amazonaws.com\"},\"Action\":\"sts:AssumeRole\"}]}"
}

resource "aws_iam_instance_profile" "test" {
	name = "test"
	roles = ["${aws_iam_role.test.name}"]
}
`

const testAccAWSInstanceProfilePrefixNameConfig = `
resource "aws_iam_role" "test" {
	name = "test"
	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"ec2.amazonaws.com\"},\"Action\":\"sts:AssumeRole\"}]}"
}

resource "aws_iam_instance_profile" "test" {
	name_prefix = "test-"
	roles = ["${aws_iam_role.test.name}"]
}
`
