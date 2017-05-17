package aws

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSLightsailKeyPair_basic(t *testing.T) {
	var conf lightsail.KeyPair
	lightsailName := fmt.Sprintf("tf-test-lightsail-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLightsailKeyPairDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLightsailKeyPairConfig_basic(lightsailName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSLightsailKeyPairExists("aws_lightsail_key_pair.lightsail_key_pair_test", &conf),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test", "arn"),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test", "fingerprint"),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test", "public_key"),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test", "private_key"),
				),
			},
		},
	})
}

func TestAccAWSLightsailKeyPair_imported(t *testing.T) {
	var conf lightsail.KeyPair
	lightsailName := fmt.Sprintf("tf-test-lightsail-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLightsailKeyPairDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLightsailKeyPairConfig_imported(lightsailName, testLightsailKeyPairPubKey1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSLightsailKeyPairExists("aws_lightsail_key_pair.lightsail_key_pair_test", &conf),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test", "arn"),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test", "fingerprint"),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test", "public_key"),
					resource.TestCheckNoResourceAttr("aws_lightsail_key_pair.lightsail_key_pair_test", "encrypted_fingerprint"),
					resource.TestCheckNoResourceAttr("aws_lightsail_key_pair.lightsail_key_pair_test", "encrypted_private_key"),
					resource.TestCheckNoResourceAttr("aws_lightsail_key_pair.lightsail_key_pair_test", "private_key"),
				),
			},
		},
	})
}

func TestAccAWSLightsailKeyPair_encrypted(t *testing.T) {
	var conf lightsail.KeyPair
	lightsailName := fmt.Sprintf("tf-test-lightsail-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLightsailKeyPairDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLightsailKeyPairConfig_encrypted(lightsailName, testLightsailKeyPairPubKey1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSLightsailKeyPairExists("aws_lightsail_key_pair.lightsail_key_pair_test", &conf),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test", "arn"),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test", "fingerprint"),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test", "encrypted_fingerprint"),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test", "encrypted_private_key"),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test", "public_key"),
					resource.TestCheckNoResourceAttr("aws_lightsail_key_pair.lightsail_key_pair_test", "private_key"),
				),
			},
		},
	})
}

func TestAccAWSLightsailKeyPair_nameprefix(t *testing.T) {
	var conf1, conf2 lightsail.KeyPair

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLightsailKeyPairDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLightsailKeyPairConfig_prefixed(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSLightsailKeyPairExists("aws_lightsail_key_pair.lightsail_key_pair_test_omit", &conf1),
					testAccCheckAWSLightsailKeyPairExists("aws_lightsail_key_pair.lightsail_key_pair_test_prefixed", &conf2),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test_omit", "name"),
					resource.TestCheckResourceAttrSet("aws_lightsail_key_pair.lightsail_key_pair_test_prefixed", "name"),
				),
			},
		},
	})
}

func testAccCheckAWSLightsailKeyPairExists(n string, res *lightsail.KeyPair) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No LightsailKeyPair set")
		}

		conn := testAccProvider.Meta().(*AWSClient).lightsailconn

		respKeyPair, err := conn.GetKeyPair(&lightsail.GetKeyPairInput{
			KeyPairName: aws.String(rs.Primary.Attributes["name"]),
		})

		if err != nil {
			return err
		}

		if respKeyPair == nil || respKeyPair.KeyPair == nil {
			return fmt.Errorf("KeyPair (%s) not found", rs.Primary.Attributes["name"])
		}
		*res = *respKeyPair.KeyPair
		return nil
	}
}

func testAccCheckAWSLightsailKeyPairDestroy(s *terraform.State) error {

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_lightsail_key_pair" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).lightsailconn

		respKeyPair, err := conn.GetKeyPair(&lightsail.GetKeyPairInput{
			KeyPairName: aws.String(rs.Primary.Attributes["name"]),
		})

		if err == nil {
			if respKeyPair.KeyPair != nil {
				return fmt.Errorf("LightsailKeyPair %q still exists", rs.Primary.ID)
			}
		}

		// Verify the error
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NotFoundException" {
				return nil
			}
		}
		return err
	}

	return nil
}

func testAccAWSLightsailKeyPairConfig_basic(lightsailName string) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}
resource "aws_lightsail_key_pair" "lightsail_key_pair_test" {
  name = "%s"
}
`, lightsailName)
}

func testAccAWSLightsailKeyPairConfig_imported(lightsailName, key string) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}
resource "aws_lightsail_key_pair" "lightsail_key_pair_test" {
  name = "%s"
	
	public_key = "%s"
}
`, lightsailName, lightsailPubKey)
}

func testAccAWSLightsailKeyPairConfig_encrypted(lightsailName, key string) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}
resource "aws_lightsail_key_pair" "lightsail_key_pair_test" {
  name = "%s"
	
	pgp_key = <<EOF
%s
EOF
}
`, lightsailName, key)
}

func testAccAWSLightsailKeyPairConfig_prefixed() string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}
resource "aws_lightsail_key_pair" "lightsail_key_pair_test_omit" {}
resource "aws_lightsail_key_pair" "lightsail_key_pair_test_prefixed" {
	name_prefix = "cts"
}
`)
}

const lightsailPubKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com`
const testLightsailKeyPairPubKey1 = `mQENBFXbjPUBCADjNjCUQwfxKL+RR2GA6pv/1K+zJZ8UWIF9S0lk7cVIEfJiprzzwiMwBS5cD0da
rGin1FHvIWOZxujA7oW0O2TUuatqI3aAYDTfRYurh6iKLC+VS+F7H+/mhfFvKmgr0Y5kDCF1j0T/
063QZ84IRGucR/X43IY7kAtmxGXH0dYOCzOe5UBX1fTn3mXGe2ImCDWBH7gOViynXmb6XNvXkP0f
sF5St9jhO7mbZU9EFkv9O3t3EaURfHopsCVDOlCkFCw5ArY+DUORHRzoMX0PnkyQb5OzibkChzpg
8hQssKeVGpuskTdz5Q7PtdW71jXd4fFVzoNH8fYwRpziD2xNvi6HABEBAAG0EFZhdWx0IFRlc3Qg
S2V5IDGJATgEEwECACIFAlXbjPUCGy8GCwkIBwMCBhUIAgkKCwQWAgMBAh4BAheAAAoJEOfLr44B
HbeTo+sH/i7bapIgPnZsJ81hmxPj4W12uvunksGJiC7d4hIHsG7kmJRTJfjECi+AuTGeDwBy84TD
cRaOB6e79fj65Fg6HgSahDUtKJbGxj/lWzmaBuTzlN3CEe8cMwIPqPT2kajJVdOyrvkyuFOdPFOE
A7bdCH0MqgIdM2SdF8t40k/ATfuD2K1ZmumJ508I3gF39jgTnPzD4C8quswrMQ3bzfvKC3klXRlB
C0yoArn+0QA3cf2B9T4zJ2qnvgotVbeK/b1OJRNj6Poeo+SsWNc/A5mw7lGScnDgL3yfwCm1gQXa
QKfOt5x+7GqhWDw10q+bJpJlI10FfzAnhMF9etSqSeURBRW5AQ0EVduM9QEIAL53hJ5bZJ7oEDCn
aY+SCzt9QsAfnFTAnZJQrvkvusJzrTQ088eUQmAjvxkfRqnv981fFwGnh2+I1Ktm698UAZS9Jt8y
jak9wWUICKQO5QUt5k8cHwldQXNXVXFa+TpQWQR5yW1a9okjh5o/3d4cBt1yZPUJJyLKY43Wvptb
6EuEsScO2DnRkh5wSMDQ7dTooddJCmaq3LTjOleRFQbu9ij386Do6jzK69mJU56TfdcydkxkWF5N
ZLGnED3lq+hQNbe+8UI5tD2oP/3r5tXKgMy1R/XPvR/zbfwvx4FAKFOP01awLq4P3d/2xOkMu4Lu
9p315E87DOleYwxk+FoTqXEAEQEAAYkCPgQYAQIACQUCVduM9QIbLgEpCRDny6+OAR23k8BdIAQZ
AQIABgUCVduM9QAKCRAID0JGyHtSGmqYB/4m4rJbbWa7dBJ8VqRU7ZKnNRDR9CVhEGipBmpDGRYu
lEimOPzLUX/ZXZmTZzgemeXLBaJJlWnopVUWuAsyjQuZAfdd8nHkGRHG0/DGum0l4sKTta3OPGHN
C1z1dAcQ1RCr9bTD3PxjLBczdGqhzw71trkQRBRdtPiUchltPMIyjUHqVJ0xmg0hPqFic0fICsr0
YwKoz3h9+QEcZHvsjSZjgydKvfLYcm+4DDMCCqcHuJrbXJKUWmJcXR0y/+HQONGrGJ5xWdO+6eJi
oPn2jVMnXCm4EKc7fcLFrz/LKmJ8seXhxjM3EdFtylBGCrx3xdK0f+JDNQaC/rhUb5V2XuX6VwoH
/AtY+XsKVYRfNIupLOUcf/srsm3IXT4SXWVomOc9hjGQiJ3rraIbADsc+6bCAr4XNZS7moViAAcI
PXFv3m3WfUlnG/om78UjQqyVACRZqqAGmuPq+TSkRUCpt9h+A39LQWkojHqyob3cyLgy6z9Q557O
9uK3lQozbw2gH9zC0RqnePl+rsWIUU/ga16fH6pWc1uJiEBt8UZGypQ/E56/343epmYAe0a87sHx
8iDV+dNtDVKfPRENiLOOc19MmS+phmUyrbHqI91c0pmysYcJZCD3a502X1gpjFbPZcRtiTmGnUKd
OIu60YPNE4+h7u2CfYyFPu3AlUaGNMBlvy6PEpU=`
