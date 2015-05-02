package aws

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCloudFront(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudFrontConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudfrontInitial(
						"aws_cloudfront.main",
					),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudFrontUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudfrontSecondary(
						"aws_cloudfront.main",
					),
				),
			},
		},
	})
}

func testAccCheckCloudFrontDestroy(s *terraform.State) error {
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v", s.RootModule().Resources)
	}

	return nil
}

func testAccCheckCloudfrontInitial(cloudFrontResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cloudFront, ok := s.RootModule().Resources[cloudFrontResource]
		if !ok {
			return fmt.Errorf("Not found: %s", cloudFrontResource)
		}

		if cloudFront.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		cloudfrontconn := testAccProvider.Meta().(*AWSClient).cloudfrontconn

		req := &cloudfront.GetDistributionInput{
			ID: aws.String(cloudFront.Primary.ID),
		}

		_, err := cloudfrontconn.GetDistribution(req)
		if err != nil {
			return fmt.Errorf("Error retrieving CloudFront distribution: %s", err)
		}

		return nil
	}
}

func testAccCheckCloudfrontSecondary(cloudFrontResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cloudFront, ok := s.RootModule().Resources[cloudFrontResource]
		if !ok {
			return fmt.Errorf("Not found: %s", cloudFrontResource)
		}

		cloudfrontconn := testAccProvider.Meta().(*AWSClient).cloudfrontconn

		req := &cloudfront.GetDistributionInput{
			ID: aws.String(cloudFront.Primary.ID),
		}

		res, err := cloudfrontconn.GetDistribution(req)
		if err != nil {
			return fmt.Errorf("Error retrieving CloudFront distribution: %s", err)
		}

		if len(res.Distribution.DistributionConfig.Aliases.Items) != 1 {
			return fmt.Errorf("CloudFront failed updating aliases")
		}

		if *res.Distribution.DistributionConfig.Enabled != false {
			return fmt.Errorf("CloudFront failed updating enabled status")
		}

		return nil
	}
}

const testAccAWSCloudFrontConfig = `
resource "aws_cloudfront" "main" {
  origin_domain_name = "fileserver.example.com"
}
`

// CloudFront does not allow CNAME conflicts on the same account
var testAccAWSCloudFrontUpdate = fmt.Sprintf(`
resource "aws_cloudfront" "main" {
	enabled = false
  origin_domain_name = "fileserver.example.com"
	aliases = ["static-%d.example.com"]
}
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int())
