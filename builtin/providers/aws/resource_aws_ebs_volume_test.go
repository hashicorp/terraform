package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"regexp"
)

func TestAccAWSEBSVolume_basic(t *testing.T) {
	var v ec2.Volume
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_ebs_volume.test",
		Providers:     testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsEbsVolumeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVolumeExists("aws_ebs_volume.test", &v),
				),
			},
		},
	})
}

func TestAccAWSEBSVolume_kmsKey(t *testing.T) {
	var v ec2.Volume
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAwsEbsVolumeConfigWithKmsKey, ri)
	keyRegex := regexp.MustCompile("^arn:aws:([a-zA-Z0-9\\-])+:([a-z]{2}-[a-z]+-\\d{1})?:(\\d{12})?:(.*)$")

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_ebs_volume.test",
		Providers:     testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVolumeExists("aws_ebs_volume.test", &v),
					resource.TestCheckResourceAttr("aws_ebs_volume.test", "encrypted", "true"),
					resource.TestMatchResourceAttr("aws_ebs_volume.test", "kms_key_id", keyRegex),
				),
			},
		},
	})
}

func TestAccAWSEBSVolume_NoIops(t *testing.T) {
	var v ec2.Volume
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsEbsVolumeConfigWithNoIops,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVolumeExists("aws_ebs_volume.iops_test", &v),
				),
			},
		},
	})
}

func TestAccAWSEBSVolume_withTags(t *testing.T) {
	var v ec2.Volume
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_ebs_volume.tags_test",
		Providers:     testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsEbsVolumeConfigWithTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVolumeExists("aws_ebs_volume.tags_test", &v),
				),
			},
		},
	})
}

func testAccCheckVolumeExists(n string, v *ec2.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		request := &ec2.DescribeVolumesInput{
			VolumeIds: []*string{aws.String(rs.Primary.ID)},
		}

		response, err := conn.DescribeVolumes(request)
		if err == nil {
			if response.Volumes != nil && len(response.Volumes) > 0 {
				*v = *response.Volumes[0]
				return nil
			}
		}
		return fmt.Errorf("Error finding EC2 volume %s", rs.Primary.ID)
	}
}

const testAccAwsEbsVolumeConfig = `
resource "aws_ebs_volume" "test" {
  availability_zone = "us-west-2a"
  size = 1
}
`

const testAccAwsEbsVolumeConfigWithKmsKey = `
resource "aws_kms_key" "foo" {
  description = "Terraform acc test %d"
  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Id": "kms-tf-1",
  "Statement": [
    {
      "Sid": "Enable IAM User Permissions",
      "Effect": "Allow",
      "Principal": {
        "AWS": "*"
      },
      "Action": "kms:*",
      "Resource": "*"
    }
  ]
}
POLICY
}

resource "aws_ebs_volume" "test" {
  availability_zone = "us-west-2a"
  size = 1
  encrypted = true
  kms_key_id = "${aws_kms_key.foo.arn}"
}
`

const testAccAwsEbsVolumeConfigWithTags = `
resource "aws_ebs_volume" "tags_test" {
  availability_zone = "us-west-2a"
  size = 1
  tags {
    Name = "TerraformTest"
  }
}
`

const testAccAwsEbsVolumeConfigWithNoIops = `
resource "aws_ebs_volume" "iops_test" {
  availability_zone = "us-west-2a"
  size = 10
  type = "gp2"
  iops = 0
  tags {
    Name = "TerraformTest"
  }
}
`
