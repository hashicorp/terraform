package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSUserSSHKey_basic(t *testing.T) {
	var conf iam.GetSSHPublicKeyOutput

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAWSSSHKeyConfig_sshEncoding, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSUserSSHKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSUserSSHKeyExists("aws_iam_user_ssh_key.user", &conf),
				),
			},
		},
	})
}

func TestAccAWSUserSSHKey_pemEncoding(t *testing.T) {
	var conf iam.GetSSHPublicKeyOutput

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAWSSSHKeyConfig_pemEncoding, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSUserSSHKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSUserSSHKeyExists("aws_iam_user_ssh_key.user", &conf),
				),
			},
		},
	})
}

func testAccCheckAWSUserSSHKeyDestroy(s *terraform.State) error {
	iamconn := testAccProvider.Meta().(*AWSClient).iamconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iam_user_ssh_key" {
			continue
		}

		username := rs.Primary.Attributes["username"]
		encoding := rs.Primary.Attributes["encoding"]
		_, err := iamconn.GetSSHPublicKey(&iam.GetSSHPublicKeyInput{
			SSHPublicKeyId: aws.String(rs.Primary.ID),
			UserName:       aws.String(username),
			Encoding:       aws.String(encoding),
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

func testAccCheckAWSUserSSHKeyExists(n string, res *iam.GetSSHPublicKeyOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SSHPublicKeyID is set")
		}

		iamconn := testAccProvider.Meta().(*AWSClient).iamconn

		username := rs.Primary.Attributes["username"]
		encoding := rs.Primary.Attributes["encoding"]
		resp, err := iamconn.GetSSHPublicKey(&iam.GetSSHPublicKeyInput{
			SSHPublicKeyId: aws.String(rs.Primary.ID),
			UserName:       aws.String(username),
			Encoding:       aws.String(encoding),
		})
		if err != nil {
			return err
		}

		*res = *resp

		return nil
	}
}

const testAccAWSSSHKeyConfig_sshEncoding = `
resource "aws_iam_user" "user" {
	name = "test-user-%d"
	path = "/"
}

resource "aws_iam_user_ssh_key" "user" {
	username = "${aws_iam_user.user.name}"
	encoding = "SSH"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}
`

const testAccAWSSSHKeyConfig_pemEncoding = `
resource "aws_iam_user" "user" {
	name = "test-user-%d"
	path = "/"
}

resource "aws_iam_user_ssh_key" "user" {
	username = "${aws_iam_user.user.name}"
	encoding = "PEM"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}
`
