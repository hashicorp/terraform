package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func TestAccAWSKeyPair_normal(t *testing.T) {
	var conf ec2.KeyPair

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKeyPairDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSKeyPairConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKeyPairExists("aws_key_pair.a_key_pair", &conf),
					testAccCheckAWSKeyPairFingerprint("d7:ff:a6:63:18:64:9c:57:a1:ee:ca:a4:ad:c2:81:62", &conf),
				),
			},
		},
	})
}

func testAccCheckAWSKeyPairDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_key_pair" {
			continue
		}

		// Try to find key pair
		resp, err := conn.KeyPairs(
			[]string{rs.Primary.ID}, nil)
		if err == nil {
			if len(resp.Keys) > 0 {
				return fmt.Errorf("still exist.")
			}
			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(*ec2.Error)
		if !ok {
			return err
		}
		if ec2err.Code != "InvalidKeyPair.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSKeyPairFingerprint(expectedFingerprint string, conf *ec2.KeyPair) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if conf.Fingerprint != expectedFingerprint {
			return fmt.Errorf("incorrect fingerprint. expected %s, got %s", expectedFingerprint, conf.Fingerprint)
		}
		return nil
	}
}

func testAccCheckAWSKeyPairExists(n string, res *ec2.KeyPair) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No KeyPair name is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		resp, err := conn.KeyPairs(
			[]string{rs.Primary.ID}, nil)
		if err != nil {
			return err
		}
		if len(resp.Keys) != 1 ||
			resp.Keys[0].Name != rs.Primary.ID {
			return fmt.Errorf("KeyPair not found")
		}
		*res = resp.Keys[0]

		return nil
	}
}

const testAccAWSKeyPairConfig = `
resource "aws_key_pair" "a_key_pair" {
	key_name   = "tf-acc-key-pair"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}
`
