package aws

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/helper/pgpkeys"
)

func TestAccAWSAccessKey_basic(t *testing.T) {
	var conf iam.AccessKeyMetadata
	rName := fmt.Sprintf("test-user-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAccessKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAccessKeyConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAccessKeyExists("aws_iam_access_key.a_key", &conf),
					testAccCheckAWSAccessKeyAttributes(&conf),
					resource.TestCheckResourceAttrSet("aws_iam_access_key.a_key", "secret"),
				),
			},
		},
	})
}

func TestAccAWSAccessKey_encrypted(t *testing.T) {
	var conf iam.AccessKeyMetadata
	rName := fmt.Sprintf("test-user-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAccessKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAccessKeyConfig_encrypted(rName, testPubAccessKey1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAccessKeyExists("aws_iam_access_key.a_key", &conf),
					testAccCheckAWSAccessKeyAttributes(&conf),
					testDecryptSecretKeyAndTest("aws_iam_access_key.a_key", testPrivKey1),
					resource.TestCheckNoResourceAttr(
						"aws_iam_access_key.a_key", "secret"),
					resource.TestCheckResourceAttrSet(
						"aws_iam_access_key.a_key", "encrypted_secret"),
					resource.TestCheckResourceAttrSet(
						"aws_iam_access_key.a_key", "key_fingerprint"),
				),
			},
		},
	})
}

func testAccCheckAWSAccessKeyDestroy(s *terraform.State) error {
	iamconn := testAccProvider.Meta().(*AWSClient).iamconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_access_key" {
			continue
		}

		// Try to get access key
		resp, err := iamconn.ListAccessKeys(&iam.ListAccessKeysInput{
			UserName: aws.String(rs.Primary.ID),
		})
		if err == nil {
			if len(resp.AccessKeyMetadata) > 0 {
				return fmt.Errorf("still exist.")
			}
			return nil
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

func testAccCheckAWSAccessKeyExists(n string, res *iam.AccessKeyMetadata) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Role name is set")
		}

		iamconn := testAccProvider.Meta().(*AWSClient).iamconn
		name := rs.Primary.Attributes["user"]

		resp, err := iamconn.ListAccessKeys(&iam.ListAccessKeysInput{
			UserName: aws.String(name),
		})
		if err != nil {
			return err
		}

		if len(resp.AccessKeyMetadata) != 1 ||
			*resp.AccessKeyMetadata[0].UserName != name {
			return fmt.Errorf("User not found not found")
		}

		*res = *resp.AccessKeyMetadata[0]

		return nil
	}
}

func testAccCheckAWSAccessKeyAttributes(accessKeyMetadata *iam.AccessKeyMetadata) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !strings.Contains(*accessKeyMetadata.UserName, "test-user") {
			return fmt.Errorf("Bad username: %s", *accessKeyMetadata.UserName)
		}

		if *accessKeyMetadata.Status != "Active" {
			return fmt.Errorf("Bad status: %s", *accessKeyMetadata.Status)
		}

		return nil
	}
}

func testDecryptSecretKeyAndTest(nAccessKey, key string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		keyResource, ok := s.RootModule().Resources[nAccessKey]
		if !ok {
			return fmt.Errorf("Not found: %s", nAccessKey)
		}

		password, ok := keyResource.Primary.Attributes["encrypted_secret"]
		if !ok {
			return errors.New("No password in state")
		}

		// We can't verify that the decrypted password is correct, because we don't
		// have it. We can verify that decrypting it does not error
		_, err := pgpkeys.DecryptBytes(password, key)
		if err != nil {
			return fmt.Errorf("Error decrypting password: %s", err)
		}

		return nil
	}
}

func testAccAWSAccessKeyConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_iam_user" "a_user" {
        name = "%s"
}

resource "aws_iam_access_key" "a_key" {
        user    = "${aws_iam_user.a_user.name}"
}
`, rName)
}

func testAccAWSAccessKeyConfig_encrypted(rName, key string) string {
	return fmt.Sprintf(`
resource "aws_iam_user" "a_user" {
        name = "%s"
}

resource "aws_iam_access_key" "a_key" {
        user    = "${aws_iam_user.a_user.name}"
        pgp_key = <<EOF
%s
EOF
}
`, rName, key)
}

func TestSesSmtpPasswordFromSecretKey(t *testing.T) {
	cases := []struct {
		Input    string
		Expected string
	}{
		{"some+secret+key", "AnkqhOiWEcszZZzTMCQbOY1sPGoLFgMH9zhp4eNgSjo4"},
		{"another+secret+key", "Akwqr0Giwi8FsQFgW3DXWCC2DiiQ/jZjqLDWK8TeTBgL"},
	}

	for _, tc := range cases {
		actual := sesSmtpPasswordFromSecretKey(&tc.Input)
		if actual != tc.Expected {
			t.Fatalf("%q: expected %q, got %q", tc.Input, tc.Expected, actual)
		}
	}
}

const testPubAccessKey1 = `mQENBFXbjPUBCADjNjCUQwfxKL+RR2GA6pv/1K+zJZ8UWIF9S0lk7cVIEfJiprzzwiMwBS5cD0da
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

const testPrivAccessKey1 = `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
7u2CfYyFPu3AlUaGNMBlvy6PEpU=`
