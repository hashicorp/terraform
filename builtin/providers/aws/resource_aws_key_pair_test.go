package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSKeyPair_basic(t *testing.T) {
	var conf ec2.KeyPairInfo

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

func TestAccAWSKeyPair_generatedName(t *testing.T) {
	var conf ec2.KeyPairInfo

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSKeyPairDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSKeyPairConfig_generatedName,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSKeyPairExists("aws_key_pair.a_key_pair", &conf),
					testAccCheckAWSKeyPairFingerprint("d7:ff:a6:63:18:64:9c:57:a1:ee:ca:a4:ad:c2:81:62", &conf),
					func(s *terraform.State) error {
						if conf.KeyName == nil {
							return fmt.Errorf("bad: No SG name")
						}
						if !strings.HasPrefix(*conf.KeyName, "terraform-") {
							return fmt.Errorf("No terraform- prefix: %s", *conf.KeyName)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccCheckAWSKeyPairDestroy(s *terraform.State) error {
	ec2conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_key_pair" {
			continue
		}

		// Try to find key pair
		resp, err := ec2conn.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{
			KeyNames: []*string{aws.String(rs.Primary.ID)},
		})
		if err == nil {
			if len(resp.KeyPairs) > 0 {
				return fmt.Errorf("still exist.")
			}
			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidKeyPair.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSKeyPairFingerprint(expectedFingerprint string, conf *ec2.KeyPairInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.KeyFingerprint != expectedFingerprint {
			return fmt.Errorf("incorrect fingerprint. expected %s, got %s", expectedFingerprint, *conf.KeyFingerprint)
		}
		return nil
	}
}

func testAccCheckAWSKeyPairExists(n string, res *ec2.KeyPairInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No KeyPair name is set")
		}

		ec2conn := testAccProvider.Meta().(*AWSClient).ec2conn

		resp, err := ec2conn.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{
			KeyNames: []*string{aws.String(rs.Primary.ID)},
		})
		if err != nil {
			return err
		}
		if len(resp.KeyPairs) != 1 ||
			*resp.KeyPairs[0].KeyName != rs.Primary.ID {
			return fmt.Errorf("KeyPair not found")
		}

		*res = *resp.KeyPairs[0]

		return nil
	}
}

const testAccAWSKeyPairConfig = `
resource "aws_key_pair" "a_key_pair" {
	key_name   = "tf-acc-key-pair"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}
`

const testAccAWSKeyPairConfig_generatedName = `
resource "aws_key_pair" "a_key_pair" {
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}
`
